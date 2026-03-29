package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"wzap/internal/dto"
	"wzap/internal/model"
	"wzap/internal/whatsapp"
)

type ContactService struct {
	engine *whatsapp.Engine
}

func NewContactService(engine *whatsapp.Engine) *ContactService {
	return &ContactService{engine: engine}
}

func (s *ContactService) CheckContacts(ctx context.Context, sessionID string, req dto.CheckContactReq) ([]dto.CheckContactResp, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}
	if !client.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	resp, err := client.IsOnWhatsApp(ctx, req.Phones)
	if err != nil {
		return nil, fmt.Errorf("failed to check contacts: %w", err)
	}

	var results []dto.CheckContactResp
	for _, check := range resp {
		results = append(results, dto.CheckContactResp{
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

func (s *ContactService) GetAvatar(ctx context.Context, sessionID string, req dto.GetAvatarReq) (*dto.GetAvatarResp, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}

	jid, err := types.ParseJID(req.JID)
	if err != nil {
		return nil, err
	}

	info, err := client.GetProfilePictureInfo(ctx, jid, &whatsmeow.GetProfilePictureParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to get avatar: %w", err)
	}

	if info == nil {
		return &dto.GetAvatarResp{}, nil
	}

	return &dto.GetAvatarResp{
		URL: info.URL,
		ID:  info.ID,
	}, nil
}

func (s *ContactService) Block(ctx context.Context, sessionID string, req dto.BlockContactReq) (*types.Blocklist, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}

	jid, err := types.ParseJID(req.JID)
	if err != nil {
		return nil, err
	}

	return client.UpdateBlocklist(ctx, jid, events.BlocklistChangeActionBlock)
}

func (s *ContactService) Unblock(ctx context.Context, sessionID string, req dto.BlockContactReq) (*types.Blocklist, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}

	jid, err := types.ParseJID(req.JID)
	if err != nil {
		return nil, err
	}

	return client.UpdateBlocklist(ctx, jid, events.BlocklistChangeActionUnblock)
}

func (s *ContactService) GetBlocklist(ctx context.Context, sessionID string) (*types.Blocklist, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}

	return client.GetBlocklist(ctx)
}

func (s *ContactService) GetUserInfo(ctx context.Context, sessionID string, req dto.GetUserInfoReq) (map[string]dto.UserInfoResp, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}

	var jids []types.JID
	for _, jidStr := range req.JIDs {
		jid, err := types.ParseJID(jidStr)
		if err == nil {
			jids = append(jids, jid)
		}
	}

	info, err := client.GetUserInfo(ctx, jids)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	resp := make(map[string]dto.UserInfoResp)
	for jid, infoData := range info {
		var devices []string
		for _, dev := range infoData.Devices {
			devices = append(devices, fmt.Sprintf("%d", dev.Device))
		}

		resp[jid.String()] = dto.UserInfoResp{
			JID:     jid.String(),
			Status:  infoData.Status,
			Picture: infoData.PictureID,
			Devices: devices,
		}
	}

	return resp, nil
}

func (s *ContactService) GetPrivacySettings(ctx context.Context, sessionID string) (types.PrivacySettings, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return types.PrivacySettings{}, err
	}

	// client.GetPrivacySettings only returns types.PrivacySettings
	settings := client.GetPrivacySettings(ctx)
	return settings, nil
}

func (s *ContactService) SetProfilePicture(ctx context.Context, sessionID string, req dto.SetProfilePictureReq) (string, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return "", err
	}

	data, err := base64.StdEncoding.DecodeString(req.Base64)
	if err != nil {
		return "", fmt.Errorf("invalid base64: %w", err)
	}

	return client.SetGroupPhoto(ctx, *client.Store.ID, data)
}
