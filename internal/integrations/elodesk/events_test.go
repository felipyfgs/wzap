package elodesk

import (
	"context"
	"testing"

	"wzap/internal/model"
)

// TestEventDispatcher_AllEventTypesAreSafe garante que o dispatcher cobre
// todos os EventTypes publicados pelo engine sem panicar. Campos não
// implementados no MVP devem cair em log debug + return.
func TestEventDispatcher_AllEventTypesAreSafe(t *testing.T) {
	svc := NewService(context.Background(), newInMemRepo(), newMockMsgRepo(), nil)
	cfg := &Config{SessionID: "sess", Enabled: true, InboxIdentifier: "id-xyz"}
	disp := &eventDispatcher{svc: svc}
	payload := []byte(`{"EventName":"Dummy","Data":{}}`)

	for et := range model.ValidEventTypes {
		if et == model.EventAll {
			continue
		}
		t.Run(string(et), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("dispatcher panicked on %s: %v", et, r)
				}
			}()
			_ = disp.Handle(context.Background(), cfg, et, payload)
		})
	}
}
