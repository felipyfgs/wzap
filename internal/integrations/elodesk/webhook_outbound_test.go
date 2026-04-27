package elodesk

import (
	"context"
	"testing"

	"wzap/internal/dto"
)

// stubMessageSvc implementa MessageService apenas retornando IDs forjados.
type stubMessageSvc struct {
	sentText  []dto.SendTextReq
	sentImage []dto.SendMediaReq
	sentAudio []dto.SendMediaReq
	sentVideo []dto.SendMediaReq
	sentDoc   []dto.SendMediaReq
	nextID    int
	sendErr   error
}

func (s *stubMessageSvc) SendText(_ context.Context, _ string, req dto.SendTextReq) (string, error) {
	if s.sendErr != nil {
		return "", s.sendErr
	}
	s.sentText = append(s.sentText, req)
	s.nextID++
	return waMsgID(s.nextID), nil
}
func (s *stubMessageSvc) SendImage(_ context.Context, _ string, req dto.SendMediaReq) (string, error) {
	if s.sendErr != nil {
		return "", s.sendErr
	}
	s.sentImage = append(s.sentImage, req)
	s.nextID++
	return waMsgID(s.nextID), nil
}
func (s *stubMessageSvc) SendVideo(_ context.Context, _ string, req dto.SendMediaReq) (string, error) {
	if s.sendErr != nil {
		return "", s.sendErr
	}
	s.sentVideo = append(s.sentVideo, req)
	s.nextID++
	return waMsgID(s.nextID), nil
}
func (s *stubMessageSvc) SendDocument(_ context.Context, _ string, req dto.SendMediaReq) (string, error) {
	if s.sendErr != nil {
		return "", s.sendErr
	}
	s.sentDoc = append(s.sentDoc, req)
	s.nextID++
	return waMsgID(s.nextID), nil
}
func (s *stubMessageSvc) SendAudio(_ context.Context, _ string, req dto.SendMediaReq) (string, error) {
	if s.sendErr != nil {
		return "", s.sendErr
	}
	s.sentAudio = append(s.sentAudio, req)
	s.nextID++
	return waMsgID(s.nextID), nil
}
func (*stubMessageSvc) SendContact(_ context.Context, _ string, _ dto.SendContactReq) (string, error) {
	return "", nil
}
func (*stubMessageSvc) SendLocation(_ context.Context, _ string, _ dto.SendLocationReq) (string, error) {
	return "", nil
}
func (*stubMessageSvc) DeleteMessage(_ context.Context, _ string, _ dto.DeleteMessageReq) (string, error) {
	return "", nil
}
func (*stubMessageSvc) EditMessage(_ context.Context, _ string, _ dto.EditMessageReq) (string, error) {
	return "", nil
}
func (*stubMessageSvc) MarkRead(_ context.Context, _ string, _ dto.MarkReadReq) error { return nil }

func waMsgID(i int) string {
	return "wa-msg-" + string(rune('0'+i))
}

// MessageType: 0=Incoming, 1=Outgoing, 2=Activity, 3=Template (mesma convenção do Chatwoot)
func makePayload(msgType int, content, sourceID string, convID int64, isPrivate bool) dto.ElodeskWebhookPayload {
	return dto.ElodeskWebhookPayload{
		EventType: "message_created",
		Message: &dto.ElodeskWebhookMessage{
			ID:          100,
			Content:     content,
			MessageType: msgType,
			SourceID:    sourceID,
			Private:     isPrivate,
		},
		Conversation: &dto.ElodeskWebhookConversation{
			ID:        convID,
			ContactID: 1,
			InboxID:   1,
			Status:    ConversationStatusOpen,
		},
	}
}

func TestHandleIncomingWebhook_OutgoingText(t *testing.T) {
	repo := newInMemRepo()
	repo.configs["sess"] = &Config{SessionID: "sess", Enabled: true, InboxIdentifier: "id"}
	msgRepo := newMockMsgRepo()
	msgRepo.chatJIDByConvID[7] = "11988887777@s.whatsapp.net"
	msgSvc := &stubMessageSvc{}
	svc := NewService(context.Background(), repo, msgRepo, msgSvc)

	p := makePayload(1, "olá do agente", "agent-src-1", 7, false)
	if err := svc.HandleIncomingWebhook(context.Background(), "sess", p); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(msgSvc.sentText) != 1 {
		t.Fatalf("expected 1 SendText call, got %d", len(msgSvc.sentText))
	}
	if got := msgSvc.sentText[0].Body; got != "olá do agente" {
		t.Errorf("body: got %q", got)
	}
	if got := msgSvc.sentText[0].Phone; got != "11988887777@s.whatsapp.net" {
		t.Errorf("phone: got %q", got)
	}
}

func TestHandleIncomingWebhook_PrivateNoteIgnored(t *testing.T) {
	repo := newInMemRepo()
	repo.configs["sess"] = &Config{SessionID: "sess", Enabled: true, InboxIdentifier: "id"}
	msgSvc := &stubMessageSvc{}
	svc := NewService(context.Background(), repo, newMockMsgRepo(), msgSvc)

	p := makePayload(1, "internal", "s", 7, true)
	if err := svc.HandleIncomingWebhook(context.Background(), "sess", p); err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(msgSvc.sentText) != 0 {
		t.Errorf("expected no send for private note, got %d", len(msgSvc.sentText))
	}
}

func TestHandleIncomingWebhook_EchoBlocked(t *testing.T) {
	repo := newInMemRepo()
	repo.configs["sess"] = &Config{SessionID: "sess", Enabled: true, InboxIdentifier: "id"}
	msgSvc := &stubMessageSvc{}
	svc := NewService(context.Background(), repo, newMockMsgRepo(), msgSvc)

	// source_id começando com "WAID:" identifica o payload como eco do wzap.
	p := makePayload(1, "echo", "WAID:xyz", 7, false)
	if err := svc.HandleIncomingWebhook(context.Background(), "sess", p); err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(msgSvc.sentText) != 0 {
		t.Errorf("expected echo blocked, got %d send calls", len(msgSvc.sentText))
	}
}

func TestHandleIncomingWebhook_OutgoingAudio(t *testing.T) {
	repo := newInMemRepo()
	repo.configs["sess"] = &Config{SessionID: "sess", Enabled: true, InboxIdentifier: "id"}
	msgRepo := newMockMsgRepo()
	msgRepo.chatJIDByConvID[7] = "11988887777@s.whatsapp.net"
	msgSvc := &stubMessageSvc{}
	svc := NewService(context.Background(), repo, msgRepo, msgSvc)

	ext := "webm"
	fileKey := "1/uploads/abc-voice.webm"
	p := makePayload(1, "", "agent-audio-1", 7, false)
	p.Message.Attachments = []dto.ElodeskWebhookAttachment{{
		ID:        42,
		FileType:  elodeskFileTypeAudio,
		FileKey:   &fileKey,
		Extension: &ext,
		DataURL:   "http://example/voice.webm?sig=x",
	}}

	if err := svc.HandleIncomingWebhook(context.Background(), "sess", p); err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(msgSvc.sentAudio) != 1 {
		t.Fatalf("expected 1 SendAudio, got %d", len(msgSvc.sentAudio))
	}
	got := msgSvc.sentAudio[0]
	if got.URL != "http://example/voice.webm?sig=x" {
		t.Errorf("url: got %q", got.URL)
	}
	if got.Phone != "11988887777@s.whatsapp.net" {
		t.Errorf("phone: got %q", got.Phone)
	}
	if got.MimeType != "audio/webm" {
		t.Errorf("mimeType: got %q", got.MimeType)
	}
}

func TestHandleIncomingWebhook_OutgoingMediaWithoutDataURL(t *testing.T) {
	repo := newInMemRepo()
	repo.configs["sess"] = &Config{SessionID: "sess", Enabled: true, InboxIdentifier: "id"}
	msgRepo := newMockMsgRepo()
	msgRepo.chatJIDByConvID[7] = "11988887777@s.whatsapp.net"
	msgSvc := &stubMessageSvc{}
	svc := NewService(context.Background(), repo, msgRepo, msgSvc)

	p := makePayload(1, "", "agent-img-1", 7, false)
	p.Message.Attachments = []dto.ElodeskWebhookAttachment{{
		ID:       9,
		FileType: elodeskFileTypeImage,
		// DataURL ausente — deve ser skipado em vez de explodir.
	}}

	if err := svc.HandleIncomingWebhook(context.Background(), "sess", p); err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(msgSvc.sentImage)+len(msgSvc.sentAudio)+len(msgSvc.sentDoc)+len(msgSvc.sentVideo) != 0 {
		t.Errorf("expected zero send calls when dataUrl is empty")
	}
}

func TestHandleIncomingWebhook_OutgoingDocument(t *testing.T) {
	repo := newInMemRepo()
	repo.configs["sess"] = &Config{SessionID: "sess", Enabled: true, InboxIdentifier: "id"}
	msgRepo := newMockMsgRepo()
	msgRepo.chatJIDByConvID[7] = "11988887777@s.whatsapp.net"
	msgSvc := &stubMessageSvc{}
	svc := NewService(context.Background(), repo, msgRepo, msgSvc)

	ext := "pdf"
	fileKey := "1/uploads/contrato.pdf"
	p := makePayload(1, "veja o anexo", "agent-doc-1", 7, false)
	p.Message.Attachments = []dto.ElodeskWebhookAttachment{{
		ID:        10,
		FileType:  elodeskFileTypeFile,
		FileKey:   &fileKey,
		Extension: &ext,
		DataURL:   "http://example/contrato.pdf?sig=x",
	}}

	if err := svc.HandleIncomingWebhook(context.Background(), "sess", p); err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(msgSvc.sentDoc) != 1 {
		t.Fatalf("expected 1 SendDocument, got %d", len(msgSvc.sentDoc))
	}
	got := msgSvc.sentDoc[0]
	if got.MimeType != "application/pdf" {
		t.Errorf("mimeType: got %q", got.MimeType)
	}
	if got.Caption != "veja o anexo" {
		t.Errorf("caption: got %q", got.Caption)
	}
	if got.FileName != "contrato.pdf" {
		t.Errorf("fileName: got %q", got.FileName)
	}
}

func TestHandleIncomingWebhook_DuplicateBySourceID(t *testing.T) {
	repo := newInMemRepo()
	repo.configs["sess"] = &Config{SessionID: "sess", Enabled: true, InboxIdentifier: "id"}
	msgRepo := newMockMsgRepo()
	msgRepo.existingElodesk["sess:agent-src-1"] = true
	msgSvc := &stubMessageSvc{}
	svc := NewService(context.Background(), repo, msgRepo, msgSvc)

	p := makePayload(1, "dup", "agent-src-1", 7, false)
	if err := svc.HandleIncomingWebhook(context.Background(), "sess", p); err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(msgSvc.sentText) != 0 {
		t.Errorf("expected dedup by elodesk_src_id, got %d send calls", len(msgSvc.sentText))
	}
}
