package elodesk

import (
	"context"
	"errors"
	"fmt"
	"time"

	"wzap/internal/model"
	"wzap/internal/repo"
)

// mockClient implementa elodesk.Client para testes.
type mockClient struct {
	contacts      []Contact
	conversations []Conversation
	messages      []MessageReq

	nextMessageID int64
	nextConvID    int
	createMsgErr  error
}

func (m *mockClient) UpsertContact(_ context.Context, _ string, req UpsertContactReq) (*Contact, error) {
	c := Contact{ID: 1, Name: req.Name, Identifier: req.Identifier, PhoneNumber: req.PhoneNumber, SourceID: req.Identifier}
	m.contacts = append(m.contacts, c)
	return &c, nil
}

func (m *mockClient) GetOrCreateConversation(_ context.Context, _, _ string, _ GetOrCreateConvReq) (*Conversation, error) {
	m.nextConvID++
	c := Conversation{ID: m.nextConvID, ContactID: 1, InboxID: 1, Status: ConversationStatusOpen}
	m.conversations = append(m.conversations, c)
	return &c, nil
}

func (m *mockClient) CreateMessage(_ context.Context, _, _ string, convID int64, req MessageReq) (*Message, error) {
	if m.createMsgErr != nil {
		return nil, m.createMsgErr
	}
	m.messages = append(m.messages, req)
	m.nextMessageID++
	return &Message{
		ID:             m.nextMessageID,
		Content:        req.Content,
		SourceID:       req.SourceID,
		ConversationID: convID,
	}, nil
}

func (m *mockClient) CreateAttachment(_ context.Context, _, _ string, convID int64, _, _ string, _ []byte, _, _, sourceID string, _ map[string]any) (*Message, error) {
	m.nextMessageID++
	return &Message{ID: m.nextMessageID, ConversationID: convID, SourceID: sourceID}, nil
}

func (m *mockClient) UpdateConversationStatus(_ context.Context, _, _ string, _ int64, _ string) error {
	return nil
}

func (m *mockClient) CreateInbox(_ context.Context, _ int, _, _, _ string) (*CreateInboxResp, error) {
	return &CreateInboxResp{Identifier: "mock", ApiToken: "mock-token", ChannelID: 1}, nil
}

func (m *mockClient) UpdateInboxWebhook(_ context.Context, _, _ int, _, _ string) error {
	return nil
}

type mockMsgRepo struct {
	existingSourceIDs map[string]bool
	existingElodesk   map[string]bool
	chatJIDByConvID   map[int64]string
	saved             []*model.Message
	updatedElodesk    []elodeskUpdate
}

type elodeskUpdate struct {
	SessionID string
	MsgID     string
	ElMsgID   int64
	ElConvID  int64
	SrcID     string
}

func newMockMsgRepo() *mockMsgRepo {
	return &mockMsgRepo{
		existingSourceIDs: make(map[string]bool),
		existingElodesk:   make(map[string]bool),
		chatJIDByConvID:   make(map[int64]string),
	}
}

func (m *mockMsgRepo) Save(_ context.Context, msg *model.Message) error {
	m.saved = append(m.saved, msg)
	return nil
}
func (m *mockMsgRepo) FindByChat(_ context.Context, _, _ string, _, _ int) ([]model.Message, error) {
	return nil, nil
}
func (m *mockMsgRepo) FindBySession(_ context.Context, _ string, _, _ int) ([]model.Message, error) {
	return nil, nil
}
func (m *mockMsgRepo) FindByID(_ context.Context, sessionID, msgID string) (*model.Message, error) {
	return &model.Message{ID: msgID, SessionID: sessionID}, nil
}
func (m *mockMsgRepo) FindSessionByMessageID(_ context.Context, _ string) (string, error) {
	return "", errors.New("not implemented")
}
func (m *mockMsgRepo) FindByCWMessageID(_ context.Context, _ string, _ int) (*model.Message, error) {
	return nil, nil
}
func (m *mockMsgRepo) FindAllByCWMessageID(_ context.Context, _ string, _ int) ([]model.Message, error) {
	return nil, nil
}
func (m *mockMsgRepo) FindBySourceID(_ context.Context, _, _ string) (*model.Message, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockMsgRepo) FindBySourceIDPrefix(_ context.Context, _, _ string) (*model.Message, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockMsgRepo) FindByBody(_ context.Context, _, _ string, _ bool) (*model.Message, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockMsgRepo) FindByBodyAndChat(_ context.Context, _, _, _ string, _ bool) (*model.Message, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockMsgRepo) FindByBodyAndChatAny(_ context.Context, _, _, _ string, _ bool) (*model.Message, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockMsgRepo) FindByTimestamp(_ context.Context, _, _ string, _, _ int64) (*model.Message, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockMsgRepo) FindLastReceived(_ context.Context, _, _ string) (*model.Message, error) {
	return nil, fmt.Errorf("not found")
}
func (m *mockMsgRepo) UpdateChatwootRef(_ context.Context, _, _ string, _, _ int, _ string) error {
	return nil
}
func (m *mockMsgRepo) ListMissingChatwootRefs(_ context.Context, _ string, _ int, _ string) ([]string, error) {
	return nil, nil
}
func (m *mockMsgRepo) ExistsBySourceID(_ context.Context, sessionID, sourceID string) (bool, error) {
	return m.existingSourceIDs[sessionID+":"+sourceID], nil
}
func (m *mockMsgRepo) UpdateElodeskRef(_ context.Context, sessionID, msgID string, elMsgID, elConvID int64, srcID string) error {
	m.updatedElodesk = append(m.updatedElodesk, elodeskUpdate{
		SessionID: sessionID, MsgID: msgID, ElMsgID: elMsgID, ElConvID: elConvID, SrcID: srcID,
	})
	m.existingElodesk[sessionID+":"+srcID] = true
	return nil
}
func (m *mockMsgRepo) ExistsByElodeskSrcID(_ context.Context, sessionID, srcID string) (bool, error) {
	return m.existingElodesk[sessionID+":"+srcID], nil
}
func (m *mockMsgRepo) FindChatJIDByElodeskConvID(_ context.Context, _ string, elConvID int64) (string, error) {
	return m.chatJIDByConvID[elConvID], nil
}
func (m *mockMsgRepo) FindUnimportedHistory(_ context.Context, _ string, _ time.Time, _, _ int) ([]model.Message, error) {
	return []model.Message{}, nil
}
func (m *mockMsgRepo) MarkImported(_ context.Context, _, _ string) error      { return nil }
func (m *mockMsgRepo) UpdateMediaURL(_ context.Context, _, _, _ string) error { return nil }
func (m *mockMsgRepo) FindMedia(_ context.Context, _ string, _ repo.MediaFilter) ([]model.Message, int, error) {
	return nil, 0, nil
}

// inMemRepo é um Repo em memória para testes sem tocar no DB.
type inMemRepo struct {
	configs map[string]*Config
}

func newInMemRepo() *inMemRepo {
	return &inMemRepo{configs: make(map[string]*Config)}
}

func (r *inMemRepo) FindBySessionID(_ context.Context, sessionID string) (*Config, error) {
	cfg, ok := r.configs[sessionID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return cfg, nil
}
func (r *inMemRepo) Upsert(_ context.Context, cfg *Config) error {
	r.configs[cfg.SessionID] = cfg
	return nil
}
func (r *inMemRepo) Delete(_ context.Context, sessionID string) error {
	delete(r.configs, sessionID)
	return nil
}
