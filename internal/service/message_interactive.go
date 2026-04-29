package service

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"

	"wzap/internal/dto"
	"wzap/internal/model"
)

func (s *MessageService) SendButton(ctx context.Context, sessionID string, req dto.SendButtonReq) (string, error) {
	runtime, err := s.runtimeResolver.ResolveMessage(ctx, sessionID, model.CapabilityMessageButton)
	if err != nil {
		return "", err
	}

	return runConnectedRuntime(ctx, runtime.SessionRuntime, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (string, error) {
		jid, err := parseJID(req.Phone)
		if err != nil {
			return "", err
		}

		buttons := make([]*waE2E.ButtonsMessage_Button, len(req.Buttons))
		for i, b := range req.Buttons {
			buttons[i] = &waE2E.ButtonsMessage_Button{
				ButtonID: proto.String(b.ID),
				ButtonText: &waE2E.ButtonsMessage_Button_ButtonText{
					DisplayText: proto.String(b.Text),
				},
				Type: waE2E.ButtonsMessage_Button_RESPONSE.Enum(),
			}
		}

		msg := &waE2E.Message{
			ViewOnceMessage: &waE2E.FutureProofMessage{
				Message: &waE2E.Message{
					ButtonsMessage: &waE2E.ButtonsMessage{
						ContentText: proto.String(req.Body),
						FooterText:  proto.String(req.Footer),
						Buttons:     buttons,
						HeaderType:  waE2E.ButtonsMessage_EMPTY.Enum(),
						ContextInfo: buildContextInfo(req.ReplyTo, req.MentionedJIDs, nil),
					},
				},
			},
		}

		opts := buildSendOpts(req.CustomID)
		resp, err := client.SendMessage(ctx, jid, msg, opts...)
		if err != nil {
			return "", fmt.Errorf("failed to send button message: %w", err)
		}

		s.persistSent(session.ID, resp.ID, jid.String(), "buttons", req.Body, "", client)

		return resp.ID, nil
	})
}

func (s *MessageService) SendList(ctx context.Context, sessionID string, req dto.SendListReq) (string, error) {
	runtime, err := s.runtimeResolver.ResolveMessage(ctx, sessionID, model.CapabilityMessageList)
	if err != nil {
		return "", err
	}

	return runConnectedRuntime(ctx, runtime.SessionRuntime, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (string, error) {
		jid, err := parseJID(req.Phone)
		if err != nil {
			return "", err
		}

		sections := make([]*waE2E.ListMessage_Section, len(req.Sections))
		for i, sec := range req.Sections {
			rows := make([]*waE2E.ListMessage_Row, len(sec.Rows))
			for j, r := range sec.Rows {
				rows[j] = &waE2E.ListMessage_Row{
					RowID:       proto.String(r.ID),
					Title:       proto.String(r.Title),
					Description: proto.String(r.Description),
				}
			}
			sections[i] = &waE2E.ListMessage_Section{
				Title: proto.String(sec.Title),
				Rows:  rows,
			}
		}

		msg := &waE2E.Message{
			ListMessage: &waE2E.ListMessage{
				Title:       proto.String(req.Title),
				Description: proto.String(req.Body),
				FooterText:  proto.String(req.Footer),
				ButtonText:  proto.String(req.ButtonText),
				ListType:    waE2E.ListMessage_SINGLE_SELECT.Enum(),
				Sections:    sections,
				ContextInfo: buildContextInfo(req.ReplyTo, req.MentionedJIDs, nil),
			},
		}

		opts := buildSendOpts(req.CustomID)
		resp, err := client.SendMessage(ctx, jid, msg, opts...)
		if err != nil {
			return "", fmt.Errorf("failed to send list message: %w", err)
		}

		s.persistSent(session.ID, resp.ID, jid.String(), "list", req.Title, "", client)

		return resp.ID, nil
	})
}

func (s *MessageService) SendPoll(ctx context.Context, sessionID string, req dto.SendPollReq) (string, error) {
	runtime, err := s.runtimeResolver.ResolveMessage(ctx, sessionID, model.CapabilityMessagePoll)
	if err != nil {
		return "", err
	}

	return runSessionRuntime(ctx, runtime.SessionRuntime, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (string, error) {
		jid, err := parseJID(req.Phone)
		if err != nil {
			return "", err
		}

		msg := client.BuildPollCreation(req.Name, req.Options, req.SelectableCount)

		resp, err := client.SendMessage(ctx, jid, msg)
		if err != nil {
			return "", fmt.Errorf("failed to send poll message: %w", err)
		}

		s.persistSent(session.ID, resp.ID, jid.String(), "poll", req.Name, "", client)

		return resp.ID, nil
	})
}
