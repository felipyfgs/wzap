package service

import (
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow/types"
	"wzap/internal/model"
)

type ContactService struct {
	engine *Engine
}

func NewContactService(engine *Engine) *ContactService {
	return &ContactService{engine: engine}
}

func (s *ContactService) CheckContacts(ctx context.Context, sessionID string, req model.CheckContactReq) ([]model.CheckContactResp, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}
	if !client.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	resp, err := client.IsOnWhatsApp(req.Phones)
	if err != nil {
		return nil, fmt.Errorf("failed to check contacts: %w", err)
	}

	var results []model.CheckContactResp
	for _, check := range resp {
		results = append(results, model.CheckContactResp{
			Exists:      check.IsIn,
			JID:         check.JID.String(),
			PhoneNumber: strings.TrimSuffix(check.JID.User, "@s.whatsapp.net"),
		})
	}

	return results, nil
}

func (s *ContactService) List(ctx context.Context, sessionID string) ([]model.Contact, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}

	contacts, err := client.Store.Contacts.GetAllContacts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts: %w", err)
	}

	var result []model.Contact
	for jid, info := range contacts {
		if jid.Server == types.GroupServer {
			continue
		}
		result = append(result, model.Contact{
			JID:      jid.String(),
			Name:     info.FullName,
			PushName: info.PushName,
		})
	}

	return result, nil
}
