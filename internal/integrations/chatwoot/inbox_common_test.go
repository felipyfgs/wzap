package chatwoot

import (
	"context"
	"encoding/json"
	"testing"

	"wzap/internal/model"
)

// TestInboxPrologueParity valida que as etapas compartilhadas do prólogo
// (parse → LID → filtro → idempotência em cache) produzem a mesma decisão
// skip/continue para ambos os modos (API e Cloud) quando o mesmo payload
// é processado. A divergência controlada ocorre apenas quando
// checkDBIdempotency=true é passado para o modo API.
func TestInboxPrologueParity(t *testing.T) {
	tests := []struct {
		name           string
		chatJID        string
		msgID          string
		ignoreGroups   bool
		primeCache     bool
		wantSkip       bool
		senderAlt      string
		recipientAlt   string
	}{
		{
			name:     "mensagem v\u00e1lida (s.whatsapp.net) prossegue",
			chatJID:  "5511999999999@s.whatsapp.net",
			msgID:    "msg-ok",
			wantSkip: false,
		},
		{
			name:     "chatJID vazio pula",
			chatJID:  "",
			msgID:    "msg-empty",
			wantSkip: true,
		},
		{
			name:     "newsletter sempre pula",
			chatJID:  "123@newsletter",
			msgID:    "msg-news",
			wantSkip: true,
		},
		{
			name:         "grupo com ignoreGroups=true pula",
			chatJID:      "123@g.us",
			msgID:        "msg-group",
			ignoreGroups: true,
			wantSkip:     true,
		},
		{
			name:       "duplicata no cache pula",
			chatJID:    "5511@s.whatsapp.net",
			msgID:      "msg-dup",
			primeCache: true,
			wantSkip:   true,
		},
		{
			name:         "LID irresolv\u00edvel pula",
			chatJID:      "abc@lid",
			msgID:        "msg-lid",
			senderAlt:    "",
			recipientAlt: "",
			wantSkip:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			payload := makeProloguePayload(t, tc.chatJID, tc.msgID, tc.senderAlt, tc.recipientAlt)

			for _, mode := range []struct {
				name  string
				check bool
			}{
				{"api", true},
				{"cloud", false},
			} {
				t.Run(mode.name, func(t *testing.T) {
					svc := newTestService(&mockClient{})
					cfg := &Config{SessionID: "sess", IgnoreGroups: tc.ignoreGroups}
					if tc.primeCache && tc.msgID != "" {
						svc.cache.SetIdempotent(ctx, cfg.SessionID, "WAID:"+tc.msgID)
					}

					res, skip, err := svc.inboxPrologue(ctx, cfg, payload, inboxPrologueOpts{checkDBIdempotency: mode.check})
					if err != nil {
						t.Fatalf("erro inesperado: %v", err)
					}
					if skip != tc.wantSkip {
						t.Fatalf("modo %s: skip=%v, esperado %v", mode.name, skip, tc.wantSkip)
					}
					if !skip && res == nil {
						t.Fatalf("modo %s: result nil com skip=false", mode.name)
					}
					if !skip && res.chatJID == "" {
						t.Fatalf("modo %s: chatJID vazio no result", mode.name)
					}
				})
			}
		})
	}
}

func makeProloguePayload(t *testing.T, chatJID, msgID, senderAlt, recipientAlt string) []byte {
	t.Helper()
	envelope := model.EventEnvelope{
		Event:     "Message",
		Session:   model.SessionInfo{ID: "sess"},
		Timestamp: "2024-01-01T00:00:00Z",
	}
	p := waMessagePayload{
		Info: waMessageInfo{
			Chat:         chatJID,
			ID:           msgID,
			SenderAlt:    senderAlt,
			RecipientAlt: recipientAlt,
		},
		Message: map[string]any{"conversation": "oi"},
	}
	envelope.Data, _ = json.Marshal(p)
	b, _ := json.Marshal(envelope)
	return b
}
