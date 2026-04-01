package service

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"wzap/internal/dto"
	"wzap/internal/model"
	"wzap/internal/wa"
)

type GroupService struct {
	engine *wa.Manager
}

func NewGroupService(engine *wa.Manager) *GroupService {
	return &GroupService{engine: engine}
}

func (s *GroupService) List(ctx context.Context, sessionID string) ([]model.Group, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}
	if !client.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	groups, err := client.GetJoinedGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %w", err)
	}

	var result []model.Group
	ownJID := client.Store.ID

	for _, gw := range groups {
		isAdmin := false
		for _, part := range gw.Participants {
			if part.JID.User == ownJID.User && (part.IsAdmin || part.IsSuperAdmin) {
				isAdmin = true
				break
			}
		}

		result = append(result, model.Group{
			JID:          gw.JID.String(),
			Name:         gw.Name,
			Participants: len(gw.Participants),
			IsAdmin:      isAdmin,
		})
	}

	return result, nil
}

func (s *GroupService) CreateGroup(ctx context.Context, sessionID string, req dto.CreateGroupReq) (*model.Group, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}
	if !client.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	var jids []types.JID
	for _, p := range req.Participants {
		if p != "" {
			jid, err := types.ParseJID(wa.EnsureJIDSuffix(p))
			if err == nil {
				jids = append(jids, jid)
			}
		}
	}

	groupReq := whatsmeow.ReqCreateGroup{
		Name:         req.Name,
		Participants: jids,
	}

	info, err := client.CreateGroup(ctx, groupReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	return &model.Group{
		JID:          info.JID.String(),
		Name:         info.Name,
		Participants: len(info.Participants),
		IsAdmin:      true, // creator is admin
	}, nil
}

func (s *GroupService) GetInfo(ctx context.Context, sessionID string, groupJID string) (*model.Group, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}
	if !client.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	jid, err := types.ParseJID(groupJID)
	if err != nil {
		return nil, fmt.Errorf("invalid group JID: %w", err)
	}

	info, err := client.GetGroupInfo(ctx, jid)
	if err != nil {
		return nil, fmt.Errorf("failed to get group info: %w", err)
	}

	isAdmin := false
	ownJID := client.Store.ID
	for _, part := range info.Participants {
		if part.JID.User == ownJID.User && (part.IsAdmin || part.IsSuperAdmin) {
			isAdmin = true
			break
		}
	}

	return &model.Group{
		JID:          info.JID.String(),
		Name:         info.Name,
		Participants: len(info.Participants),
		IsAdmin:      isAdmin,
	}, nil
}

func (s *GroupService) GetInviteLink(ctx context.Context, sessionID string, groupJID string, reset bool) (string, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return "", err
	}
	if !client.IsConnected() {
		return "", fmt.Errorf("client not connected")
	}

	jid, err := types.ParseJID(groupJID)
	if err != nil {
		return "", fmt.Errorf("invalid group JID: %w", err)
	}

	link, err := client.GetGroupInviteLink(ctx, jid, reset)
	if err != nil {
		return "", fmt.Errorf("failed to get group invite link: %w", err)
	}

	return link, nil
}

func (s *GroupService) LeaveGroup(ctx context.Context, sessionID string, groupJID string) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}
	if !client.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	jid, err := types.ParseJID(groupJID)
	if err != nil {
		return fmt.Errorf("invalid group JID: %w", err)
	}

	if err := client.LeaveGroup(ctx, jid); err != nil {
		return fmt.Errorf("failed to leave group: %w", err)
	}
	return nil
}

func (s *GroupService) GetInfoFromLink(ctx context.Context, sessionID string, inviteCode string) (*model.Group, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}
	if !client.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	info, err := client.GetGroupInfoFromLink(ctx, inviteCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get group info from link: %w", err)
	}

	return &model.Group{
		JID:          info.JID.String(),
		Name:         info.Name,
		Participants: len(info.Participants),
		IsAdmin:      false, // Not in group yet, or at least unknown from link
	}, nil
}

func (s *GroupService) JoinWithLink(ctx context.Context, sessionID string, inviteCode string) (string, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return "", err
	}
	if !client.IsConnected() {
		return "", fmt.Errorf("client not connected")
	}

	jid, err := client.JoinGroupWithLink(ctx, inviteCode)
	if err != nil {
		return "", fmt.Errorf("failed to join group: %w", err)
	}

	return jid.String(), nil
}

func (s *GroupService) UpdateParticipants(ctx context.Context, sessionID, groupJID string, participants []string, action string) ([]types.GroupParticipant, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}
	if !client.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	jid, err := types.ParseJID(groupJID)
	if err != nil {
		return nil, fmt.Errorf("invalid group JID: %w", err)
	}

	var act whatsmeow.ParticipantChange
	switch action {
	case "add":
		act = whatsmeow.ParticipantChangeAdd
	case "remove":
		act = whatsmeow.ParticipantChangeRemove
	case "promote":
		act = whatsmeow.ParticipantChangePromote
	case "demote":
		act = whatsmeow.ParticipantChangeDemote
	default:
		return nil, fmt.Errorf("invalid action, must be add, remove, promote or demote")
	}

	var jids []types.JID
	for _, p := range participants {
		if p != "" {
			pj, err := types.ParseJID(wa.EnsureJIDSuffix(p))
			if err == nil {
				jids = append(jids, pj)
			}
		}
	}

	return client.UpdateGroupParticipants(ctx, jid, jids, act)
}

func (s *GroupService) GetRequestParticipants(ctx context.Context, sessionID, groupJID string) ([]types.GroupParticipantRequest, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}
	if !client.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	jid, err := types.ParseJID(groupJID)
	if err != nil {
		return nil, fmt.Errorf("invalid group JID: %w", err)
	}

	return client.GetGroupRequestParticipants(ctx, jid)
}

func (s *GroupService) UpdateRequestParticipants(ctx context.Context, sessionID, groupJID string, participants []string, action string) ([]types.GroupParticipant, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}
	if !client.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	jid, err := types.ParseJID(groupJID)
	if err != nil {
		return nil, fmt.Errorf("invalid group JID: %w", err)
	}

	var act whatsmeow.ParticipantRequestChange
	switch action {
	case "approve":
		act = whatsmeow.ParticipantChangeApprove
	case "reject":
		act = whatsmeow.ParticipantChangeReject
	default:
		return nil, fmt.Errorf("invalid action, must be approve or reject")
	}

	var jids []types.JID
	for _, p := range participants {
		if p != "" {
			pj, err := types.ParseJID(wa.EnsureJIDSuffix(p))
			if err == nil {
				jids = append(jids, pj)
			}
		}
	}

	return client.UpdateGroupRequestParticipants(ctx, jid, jids, act)
}

func (s *GroupService) UpdateName(ctx context.Context, sessionID, groupJID, name string) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}
	if !client.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	jid, err := types.ParseJID(groupJID)
	if err != nil {
		return fmt.Errorf("invalid group JID: %w", err)
	}
	return client.SetGroupName(ctx, jid, name)
}

func (s *GroupService) UpdateDescription(ctx context.Context, sessionID, groupJID, description string) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}
	if !client.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	jid, err := types.ParseJID(groupJID)
	if err != nil {
		return fmt.Errorf("invalid group JID: %w", err)
	}
	return client.SetGroupDescription(ctx, jid, description)
}

func (s *GroupService) UpdatePhoto(ctx context.Context, sessionID, groupJID string, photoBytes []byte) (string, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return "", err
	}
	if !client.IsConnected() {
		return "", fmt.Errorf("client not connected")
	}

	jid, err := types.ParseJID(groupJID)
	if err != nil {
		return "", fmt.Errorf("invalid group JID: %w", err)
	}
	return client.SetGroupPhoto(ctx, jid, photoBytes)
}

func (s *GroupService) SetAnnounce(ctx context.Context, sessionID, groupJID string, announce bool) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}
	if !client.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	jid, err := types.ParseJID(groupJID)
	if err != nil {
		return fmt.Errorf("invalid group JID: %w", err)
	}
	return client.SetGroupAnnounce(ctx, jid, announce)
}

func (s *GroupService) SetLocked(ctx context.Context, sessionID, groupJID string, locked bool) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}
	if !client.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	jid, err := types.ParseJID(groupJID)
	if err != nil {
		return fmt.Errorf("invalid group JID: %w", err)
	}
	return client.SetGroupLocked(ctx, jid, locked)
}

func (s *GroupService) SetJoinApproval(ctx context.Context, sessionID, groupJID string, approval bool) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}
	if !client.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	jid, err := types.ParseJID(groupJID)
	if err != nil {
		return fmt.Errorf("invalid group JID: %w", err)
	}
	return client.SetGroupJoinApprovalMode(ctx, jid, approval)
}
