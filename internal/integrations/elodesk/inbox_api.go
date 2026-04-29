package elodesk

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"wzap/internal/logger"
	"wzap/internal/model"
	"wzap/internal/wautil"
)

// isStaleConvError detecta o caso "a conversation que tínhamos em cache já
// não existe no elodesk" — tipicamente porque um agente apagou a conversa
// pela UI. O elodesk retorna 404 com `error: "Not Found"` tanto pra conv
// inexistente quanto pra contato/inbox inexistente; em qualquer um dos
// casos a recuperação é a mesma: invalidar cache e refazer o
// upsertContact + getOrCreateConversation, que recria o que faltar.
func isStaleConvError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == http.StatusNotFound
}

type apiInboxHandler struct {
	svc *Service
}

func newAPIInboxHandler(svc *Service) *apiInboxHandler {
	return &apiInboxHandler{svc: svc}
}

// HandleMessage faz a ponte WA → Elodesk:
//  1. prologue (parse + idempotência)
//  2. persiste mensagem no DB (se ainda não estiver)
//  3. upsert contact + conversation no elodesk
//  4. POST mensagem no elodesk via client.CreateMessage
//  5. atualiza refs elodesk_* no wz_messages
//
// Media, polls, reactions, buttons, list, etc. ficam como follow-up —
// caem em no-op silencioso (body vazio após extractText).
func (h *apiInboxHandler) HandleMessage(ctx context.Context, cfg *Config, payload []byte) error {
	res, skip := h.svc.inboxPrologue(ctx, cfg, payload)
	if skip {
		return nil
	}
	data := res.data
	chatJID := res.chatJID
	sourceID := res.sourceID

	pushName := data.Info.PushName
	fromMe := data.Info.IsFromMe
	msgID := data.Info.ID

	isForwarded, fwdScore := extractForwardingFromMap(data.Message)

	if msgID != "" {
		h.svc.cache.SetIdempotent(ctx, cfg.SessionID, sourceID)

		msgBody := extractText(data.Message)
		msgType := detectMessageType(data.Message)
		msgToSave := &model.Message{
			ID:              msgID,
			SessionID:       cfg.SessionID,
			ChatJID:         chatJID,
			SenderJID:       data.Info.Sender,
			FromMe:          fromMe,
			MsgType:         msgType,
			Body:            msgBody,
			IsForwarded:     isForwarded,
			ForwardingScore: fwdScore,
			Timestamp:       time.Now(),
			CreatedAt:       time.Now(),
		}
		if err := h.svc.msgRepo.Save(ctx, msgToSave); err != nil {
			logger.Warn().Str("component", "elodesk").Err(err).Str("msgID", msgID).Msg("failed to save message to DB")
		}
	}

	contactPushName := pushName
	if fromMe {
		contactPushName = ""
	}

	text := extractText(data.Message)
	// Converte o dialeto WhatsApp (`*bold*`, `_italic_`, `~strike~`) pra
	// markdown padrão antes de mandar pro elodesk; a UI renderiza markdown
	// quando content_attributes.format == "markdown" (ver
	// MessageBubble.vue:isMarkdown). Sem isso, os asteriscos aparecem
	// literais no balão.
	text = wautil.WAToMarkdown(text)
	mediaMeta := extractMediaInfo(data.Message)

	// Sem texto e sem mídia suportada → polls/reactions/buttons, etc. Ignora
	// (silent no-op por ora).
	if text == "" && mediaMeta == nil {
		return nil
	}

	messageType := "outgoing"
	if !fromMe {
		messageType = "incoming"
	}

	client := h.svc.clientFn(cfg)

	var contentAttrs map[string]any
	if isForwarded {
		// Espelha o content_attributes que o elodesk já consome em outras
		// integrações; inclui forwarding_score só quando o WhatsApp informou
		// um valor não-zero, pra UI ainda diferenciar "encaminhada" de
		// "encaminhada várias vezes" (score >= 5).
		contentAttrs = map[string]any{"is_forwarded": true}
		if fwdScore > 0 {
			contentAttrs["forwarding_score"] = fwdScore
		}
	}
	if text != "" {
		if contentAttrs == nil {
			contentAttrs = make(map[string]any)
		}
		contentAttrs["format"] = "markdown"
	}

	dispatch := messageDispatch{
		client:        client,
		cfg:           cfg,
		chatJID:       chatJID,
		contactPushN:  contactPushName,
		sourceID:      sourceID,
		messageType:   messageType,
		text:          text,
		mediaMeta:     mediaMeta,
		contentAttrs:  contentAttrs,
		fromMe:        fromMe,
		pushName:      pushName,
		senderJIDFull: data.Info.Sender,
	}

	out, convID, err := h.dispatchWithStaleRetry(ctx, dispatch)
	if err != nil {
		if mediaMeta != nil {
			logger.Warn().Str("component", "elodesk").Err(err).Str("session", cfg.SessionID).Str("mediaType", mediaMeta.MediaType).Msg("Failed to upload media to elodesk")
		} else {
			logger.Warn().Str("component", "elodesk").Err(err).Str("session", cfg.SessionID).Msg("Failed to create elodesk message")
		}
		return err
	}

	if msgID != "" && out != nil {
		_ = h.svc.msgRepo.UpdateElodeskRef(ctx, cfg.SessionID, msgID, out.ID, convID, out.SourceID)
		h.svc.cache.SetIdempotent(ctx, cfg.SessionID, sourceID)
	}
	return nil
}

// messageDispatch agrupa tudo que dispatchWithStaleRetry precisa pra
// resolver a conversa e postar a mensagem (ou anexo). Mantido aqui pra
// não poluir a assinatura da função — ela já tem lógica de retry e seria
// um bloco de 10 parâmetros posicionais.
type messageDispatch struct {
	client        Client
	cfg           *Config
	chatJID       string
	contactPushN  string
	sourceID      string
	messageType   string
	text          string
	mediaMeta     *mediaInfo
	contentAttrs  map[string]any
	fromMe        bool
	pushName      string
	senderJIDFull string
}

// dispatchWithStaleRetry resolve a conversa e posta a mensagem. Em caso
// de 404 do elodesk (conv apagada pela UI mas ainda em cache no wzap),
// invalida o cache e refaz o caminho exatamente uma vez — o segundo
// upsertContact + getOrCreateConversation reconstrói o que falta. Mais de
// uma retentativa indicaria que o erro não é cache stale e sim falha
// recorrente, então melhor falhar e devolver pro caller decidir (drop ou
// retry pelo dispatcher externo).
func (h *apiInboxHandler) dispatchWithStaleRetry(ctx context.Context, d messageDispatch) (*Message, int64, error) {
	out, convID, err := h.dispatchOnce(ctx, d)
	if err == nil {
		return out, convID, nil
	}
	if !isStaleConvError(err) {
		return nil, convID, err
	}
	logger.Info().Str("component", "elodesk").Str("session", d.cfg.SessionID).Str("chatJID", d.chatJID).Int64("staleConvID", convID).Msg("conversa apagada no elodesk; invalidando cache e recriando")
	h.svc.cache.DeleteConv(ctx, d.cfg.SessionID, d.chatJID)
	out, convID, err = h.dispatchOnce(ctx, d)
	if err != nil {
		return nil, convID, fmt.Errorf("after stale retry: %w", err)
	}
	return out, convID, nil
}

// dispatchOnce executa um ciclo único de "resolver conv + postar".
// Retorna o convID usado mesmo no caminho de erro pra o caller logar
// qual conv foi usada/falhou.
func (h *apiInboxHandler) dispatchOnce(ctx context.Context, d messageDispatch) (*Message, int64, error) {
	convID, contactSrcID, err := h.svc.findOrCreateConversation(ctx, d.cfg, d.chatJID, d.contactPushN)
	if err != nil {
		return nil, 0, fmt.Errorf("find or create conversation: %w", err)
	}

	// In group chats, the chat-level contact is the group itself; the actual
	// message author is the member identified by data.Info.Sender. Upsert that
	// member as a separate contact and pass its id as sender_contact_id so the
	// Elodesk UI can render the per-message sender avatar/name. Best-effort:
	// failures fall back to the chat-level contact (no sender attribution).
	var senderContactID *int64
	isGroup := strings.HasSuffix(d.chatJID, "@g.us")
	if isGroup && !d.fromMe && d.senderJIDFull != "" && d.senderJIDFull != d.chatJID {
		memberName := d.pushName
		if memberName == "" {
			memberName = extractPhone(d.senderJIDFull)
		}
		memberReq := UpsertContactReq{
			SourceID:    contactSourceIDFromJID(d.senderJIDFull),
			Identifier:  d.senderJIDFull,
			Name:        memberName,
			PhoneNumber: "+" + extractPhone(d.senderJIDFull),
		}
		if c, mErr := d.client.UpsertContact(ctx, d.cfg.InboxIdentifier, memberReq); mErr == nil && c != nil && c.ID > 0 {
			id := int64(c.ID)
			senderContactID = &id
		} else if mErr != nil {
			logger.Warn().Str("component", "elodesk").Err(mErr).Str("memberJID", d.senderJIDFull).Msg("failed to upsert group member contact")
		}
	}

	if d.mediaMeta != nil {
		out, err := h.uploadMedia(ctx, d.cfg, d.client, contactSrcID, convID, d.sourceID, d.messageType, d.text, d.mediaMeta, d.contentAttrs)
		return out, convID, err
	}
	msgReq := MessageReq{
		Content:           d.text,
		MessageType:       d.messageType,
		SourceID:          d.sourceID,
		SenderContactID:   senderContactID,
		ContentAttributes: d.contentAttrs,
	}
	out, err := d.client.CreateMessage(ctx, d.cfg.InboxIdentifier, contactSrcID, convID, msgReq)
	return out, convID, err
}

// uploadMedia baixa o blob do WhatsApp via mediaDownloader e encaminha pra
// elodesk via CreateAttachment (multipart). Para arquivos acima do limite
// (256 MB) cai em uma mensagem privada de aviso ao agente — espelha o que a
// integração com chatwoot faz.
func (h *apiInboxHandler) uploadMedia(
	ctx context.Context,
	cfg *Config,
	client Client,
	contactSrcID string,
	convID int64,
	sourceID, messageType, caption string,
	info *mediaInfo,
	contentAttrs map[string]any,
) (*Message, error) {
	if int64(info.FileLength) > maxMediaBytes {
		warn := fmt.Sprintf("⚠️ Arquivo muito grande (%d MB) para download (limite: 256 MB)", info.FileLength/(1024*1024))
		// Mensagem privada — não reflete pro contato, só sinaliza ao agente.
		return client.CreateMessage(ctx, cfg.InboxIdentifier, contactSrcID, convID, MessageReq{
			Content:     warn,
			MessageType: messageType,
			SourceID:    sourceID,
			Private:     true,
		})
	}
	if h.svc.mediaDownloader == nil {
		return nil, fmt.Errorf("media downloader não configurado")
	}

	timeout := time.Duration(cfg.MediaTimeout) * time.Second
	if cfg.MediaTimeout == 0 {
		timeout = 60 * time.Second
	}
	if info.FileLength > 10*1024*1024 {
		timeout = time.Duration(cfg.LargeTimeout) * time.Second
		if cfg.LargeTimeout == 0 {
			timeout = 5 * time.Minute
		}
	}
	mediaCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	data, err := h.svc.mediaDownloader.DownloadMediaByPath(
		mediaCtx, cfg.SessionID, info.DirectPath, info.FileEncSHA256, info.FileSHA256,
		info.MediaKey, info.FileLength, info.MediaType,
	)
	if err != nil {
		return nil, fmt.Errorf("download media: %w", err)
	}

	mime := info.MimeType
	if mime == "" {
		mime = detectMIME(data)
	}

	filename := resolveFilename(info, mime)

	return client.CreateAttachment(ctx, cfg.InboxIdentifier, contactSrcID, convID,
		caption, filename, data, mime, messageType, sourceID, contentAttrs)
}
