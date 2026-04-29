package service

import (
	"context"
	"fmt"
	"io"
	net_http "net/http"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"

	"wzap/internal/dto"
)

func parseJID(target string) (types.JID, error) {
	if !strings.Contains(target, "@") {
		return types.NewJID(target, types.DefaultUserServer), nil
	}
	jid, err := types.ParseJID(target)
	if err != nil {
		return types.JID{}, fmt.Errorf("invalid JID: %w", err)
	}
	return jid, nil
}

func buildContextInfo(reply *dto.ReplyContext, mentionedJIDs []string, forwarding *dto.ForwardingContext) *waE2E.ContextInfo {
	ci := &waE2E.ContextInfo{}

	if reply != nil && reply.MessageID != "" {
		ci.StanzaID = proto.String(reply.MessageID)
		if reply.Participant != "" {
			ci.Participant = proto.String(reply.Participant)
		}
		if len(reply.MentionedJID) > 0 {
			ci.MentionedJID = reply.MentionedJID
		}
	}

	if len(mentionedJIDs) > 0 {
		ci.MentionedJID = mentionedJIDs
	}

	if forwarding != nil {
		score := forwarding.Score
		if score == 0 {
			score = 1
		}
		ci.IsForwarded = proto.Bool(true)
		ci.ForwardingScore = proto.Uint32(score)
	}

	if ci.StanzaID == nil && ci.Participant == nil && len(ci.MentionedJID) == 0 && ci.IsForwarded == nil {
		return nil
	}

	return ci
}

func buildSendOpts(customID string) []whatsmeow.SendRequestExtra {
	if customID == "" {
		return nil
	}
	return []whatsmeow.SendRequestExtra{{ID: customID}}
}

var defaultHTTPClient = &net_http.Client{Timeout: 60 * time.Second}

func downloadURL(ctx context.Context, url string) ([]byte, error) {
	req, err := net_http.NewRequestWithContext(ctx, net_http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build download request: %w", err)
	}
	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download from url: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download returned status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read download body: %w", err)
	}
	return data, nil
}
