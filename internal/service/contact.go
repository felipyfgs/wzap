package service

import (
	"context"
	"encoding/base64"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"wzap/internal/dto"
	"wzap/internal/model"
	"wzap/internal/wa"
)

type ContactService struct {
	engine *wa.Manager
}

func NewContactService(engine *wa.Manager) *ContactService {
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
			PhoneNumber: check.JID.User,
		})
	}

	return results, nil
}

func (s *ContactService) List(ctx context.Context, sessionID string, filter string) ([]model.Contact, error) {
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
		if jid.Server == types.GroupServer || jid.Server == types.HiddenUserServer {
			continue
		}
		if filter == "saved" && info.FullName == "" && info.FirstName == "" {
			continue
		}
		result = append(result, model.Contact{
			JID:          jid.String(),
			Name:         info.FullName,
			PushName:     info.PushName,
			BusinessName: info.BusinessName,
		})
	}

	return result, nil
}

func (s *ContactService) GetAvatar(ctx context.Context, sessionID string, req dto.GetAvatarReq) (*dto.GetAvatarResp, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}
	if !client.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	jid, err := types.ParseJID(req.Phone)
	if err != nil {
		return nil, err
	}

	info, err := client.GetProfilePictureInfo(ctx, jid, &whatsmeow.GetProfilePictureParams{})
	if err != nil {
		return &dto.GetAvatarResp{}, nil
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
	if !client.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	jid, err := types.ParseJID(req.Phone)
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
	if !client.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	jid, err := types.ParseJID(req.Phone)
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
	if !client.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	return client.GetBlocklist(ctx)
}

func (s *ContactService) GetUserInfo(ctx context.Context, sessionID string, req dto.GetUserInfoReq) (map[string]dto.UserInfoResp, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}

	var jids []types.JID
	for _, jidStr := range req.Phones {
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
	if !client.IsConnected() {
		return types.PrivacySettings{}, fmt.Errorf("client not connected")
	}

	settings := client.GetPrivacySettings(ctx)
	return settings, nil
}

func (s *ContactService) SetProfilePicture(ctx context.Context, sessionID string, req dto.SetProfilePictureReq) (string, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return "", err
	}
	if !client.IsConnected() || client.Store.ID == nil {
		return "", fmt.Errorf("client not connected")
	}

	data, err := base64.StdEncoding.DecodeString(req.Image)
	if err != nil {
		return "", fmt.Errorf("invalid base64: %w", err)
	}

	return client.SetGroupPhoto(ctx, client.Store.ID.ToNonAD(), data)
}

func (s *ContactService) SubscribePresence(ctx context.Context, sessionID string, req dto.SubscribePresenceReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}
	if !client.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	jid, err := types.ParseJID(req.Phone)
	if err != nil {
		return fmt.Errorf("invalid JID: %w", err)
	}

	return client.SubscribePresence(ctx, jid)
}

func (s *ContactService) SetPrivacy(ctx context.Context, sessionID string, req dto.SetPrivacyReq) (types.PrivacySettings, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return types.PrivacySettings{}, err
	}
	if !client.IsConnected() {
		return types.PrivacySettings{}, fmt.Errorf("client not connected")
	}

	return client.SetPrivacySetting(ctx, types.PrivacySettingType(req.Setting), types.PrivacySetting(req.Value))
}

func (s *ContactService) SetStatusMessage(ctx context.Context, sessionID string, req dto.SetStatusMessageReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}
	if !client.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	return client.SetStatusMessage(ctx, req.Status)
}

func (s *ContactService) UpdateProfileName(ctx context.Context, sessionID string, name string) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}
	if !client.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	patch := appstate.BuildSettingPushName(name)
	return client.SendAppState(ctx, patch)
}
