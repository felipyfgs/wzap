package service

import (
	"context"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"wzap/internal/dto"
	"wzap/internal/whatsapp"
)

type CommunityService struct {
	engine *whatsapp.Engine
}

func NewCommunityService(engine *whatsapp.Engine) *CommunityService {
	return &CommunityService{engine: engine}
}

func (s *CommunityService) Create(ctx context.Context, sessionID string, req dto.CreateCommunityReq) (*types.GroupInfo, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}

	reqCreate := whatsmeow.ReqCreateGroup{
		Name: req.Name,
	}

	return client.CreateGroup(ctx, reqCreate)
}

func (s *CommunityService) AddParticipant(ctx context.Context, sessionID string, req dto.CommunityParticipantReq) ([]types.GroupParticipant, error) {
	return s.updateParticipant(ctx, sessionID, req, whatsmeow.ParticipantChangeAdd)
}

func (s *CommunityService) RemoveParticipant(ctx context.Context, sessionID string, req dto.CommunityParticipantReq) ([]types.GroupParticipant, error) {
	return s.updateParticipant(ctx, sessionID, req, whatsmeow.ParticipantChangeRemove)
}

func (s *CommunityService) updateParticipant(ctx context.Context, sessionID string, req dto.CommunityParticipantReq, action whatsmeow.ParticipantChange) ([]types.GroupParticipant, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}

	jid, err := types.ParseJID(req.JID)
	if err != nil {
		return nil, err
	}

	var jids []types.JID
	for _, p := range req.Participants {
		if p != "" {
			if !strings.Contains(p, "@") {
				p += "@s.whatsapp.net"
			}
			pj, err := types.ParseJID(p)
			if err == nil {
				jids = append(jids, pj)
			}
		}
	}

	return client.UpdateGroupParticipants(ctx, jid, jids, action)
}
