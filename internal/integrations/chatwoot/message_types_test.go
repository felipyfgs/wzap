package chatwoot

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"testing"

	"wzap/internal/imgutil"
	"wzap/internal/model"
)

// ── Task 6.6: Media streaming / size check ────────────────────────────────────

func TestHandleMessage_FileTooBig(t *testing.T) {
	client := &mockClient{
		contacts:      []Contact{{ID: 1}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}
	svc := newTestService(client)
	cfg := &Config{SessionID: "sess", Enabled: true, InboxID: 1, TimeoutMediaSeconds: 60, TimeoutLargeSeconds: 300}

	msg := map[string]any{
		"imageMessage": map[string]any{
			"directPath": "/v/test",
			"mediaKey":   "AAAA",
			"fileLength": float64(maxMediaBytes + 1),
			"mimetype":   "image/jpeg",
		},
	}
	svc.processMediaMessage(context.Background(), cfg, 1, "msg1", false, msg, "", 0)

	if len(client.messages) == 0 {
		t.Error("expected warning message created for oversized file")
	}
	if len(client.attachments) > 0 {
		t.Error("expected no attachment created for oversized file")
	}
}

// ── Task 7.6: Sticker conversion ──────────────────────────────────────────────

func TestConvertWebPToPNG_ValidImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	img.Set(5, 5, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)

	out, err := imgutil.ConvertWebPToPNG(buf.Bytes())
	if err != nil {
		t.Fatalf("convertWebPToPNG failed on valid PNG: %v", err)
	}
	if len(out) == 0 {
		t.Error("expected non-empty PNG output")
	}
}

func TestConvertWebPToPNG_InvalidData(t *testing.T) {
	_, err := imgutil.ConvertWebPToPNG([]byte("not-an-image"))
	if err == nil {
		t.Error("expected error on invalid image data")
	}
}

func TestConvertWebPToGIF_ValidImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)

	out, err := imgutil.ConvertWebPToGIF(buf.Bytes())
	if err != nil {
		t.Fatalf("convertWebPToGIF failed: %v", err)
	}
	if len(out) == 0 {
		t.Error("expected non-empty GIF output")
	}
}

// ── Task 8.6: Polls e reactions ───────────────────────────────────────────────

func TestHandlePollCreation(t *testing.T) {
	client := &mockClient{
		contacts:      []Contact{{ID: 1}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}
	svc := newTestService(client)
	cfg := &Config{SessionID: "sess", Enabled: true, InboxID: 1}

	poll := map[string]any{
		"name": "Qual sua cor favorita?",
		"options": []any{
			map[string]any{"optionName": "Azul"},
			map[string]any{"optionName": "Verde"},
		},
	}
	svc.processPollCreation(context.Background(), cfg, 1, "poll-msg", false, poll)

	if len(client.messages) == 0 {
		t.Fatal("expected poll message to be created")
	}
	if !containsStr(client.messages[0].Content, "Enquete") {
		t.Errorf("expected poll header in content, got: %s", client.messages[0].Content)
	}
	if !containsStr(client.messages[0].Content, "Azul") {
		t.Errorf("expected poll option 'Azul' in content, got: %s", client.messages[0].Content)
	}
}

func TestHandleReaction_Add(t *testing.T) {
	cwID := 99
	convID := 1
	client := &mockClient{
		contacts:      []Contact{{ID: 1}},
		conversations: []Conversation{{ID: convID, InboxID: 1, Status: "open"}},
	}
	svc := newTestService(client)
	svc.msgRepo = &mockMsgRepoFixed{cwMsgID: &cwID, cwConvID: &convID}
	cfg := &Config{SessionID: "sess", Enabled: true, InboxID: 1}

	reactMsg := map[string]any{
		"key":  map[string]any{"ID": "target-msg"},
		"text": "👍",
	}
	svc.processReaction(context.Background(), cfg, convID, "react-msg", false, reactMsg)

	if len(client.messages) == 0 {
		t.Error("expected reaction message to be created")
	}
	if client.messages[0].Content != "👍" {
		t.Errorf("expected reaction content '👍', got %s", client.messages[0].Content)
	}
}

func TestHandleReaction_Remove(t *testing.T) {
	cwID := 99
	convID := 1
	client := &mockClient{
		contacts:      []Contact{{ID: 1}},
		conversations: []Conversation{{ID: convID, InboxID: 1, Status: "open"}},
	}
	svc := newTestService(client)
	svc.msgRepo = &mockMsgRepoFixed{cwMsgID: &cwID, cwConvID: &convID}
	cfg := &Config{SessionID: "sess", Enabled: true, InboxID: 1}

	reactMsg := map[string]any{
		"key":  map[string]any{"ID": "target-msg"},
		"text": "",
	}
	svc.processReaction(context.Background(), cfg, convID, "react-msg", false, reactMsg)
	// Should call DeleteMessage (no panic)
}

// ── Task 9.7: Location, vCard, Edit ───────────────────────────────────────────

func TestExtractLocationFromText_GoogleMaps(t *testing.T) {
	text := "📍 Location\nhttps://www.google.com/maps?q=-23.5505,-46.6333"
	lat, lng, ok := extractLocationFromText(text)
	if !ok {
		t.Fatal("expected location extraction to succeed")
	}
	if lat < -23.6 || lat > -23.5 {
		t.Errorf("unexpected lat: %f", lat)
	}
	if lng < -46.7 || lng > -46.6 {
		t.Errorf("unexpected lng: %f", lng)
	}
}

func TestExtractLocationFromText_Plain(t *testing.T) {
	text := "-23.5505, -46.6333"
	_, _, ok := extractLocationFromText(text)
	if !ok {
		t.Error("expected plain coords to be extracted")
	}
}

func TestExtractLocationFromText_NoCoords(t *testing.T) {
	_, _, ok := extractLocationFromText("just a plain text message")
	if ok {
		t.Error("expected no location in plain text")
	}
}

func TestIsVCardContent(t *testing.T) {
	if !isVCardContent("BEGIN:VCARD\nFN:John\nEND:VCARD") {
		t.Error("expected vCard detection")
	}
	if isVCardContent("Hello World") {
		t.Error("expected no vCard in plain text")
	}
}

func TestSplitVCards_Multiple(t *testing.T) {
	content := "BEGIN:VCARD\nFN:Alice\nEND:VCARD\nBEGIN:VCARD\nFN:Bob\nEND:VCARD\n"
	cards := splitVCards(content)
	if len(cards) != 2 {
		t.Errorf("expected 2 vCards, got %d", len(cards))
	}
}

func TestFormatVCard(t *testing.T) {
	vcard := "BEGIN:VCARD\nFN:John Doe\nTEL:+5511999999999\nEND:VCARD"
	result := formatVCard(vcard)
	if !containsStr(result, "John Doe") {
		t.Errorf("expected name in formatted vCard, got: %s", result)
	}
	if !containsStr(result, "+5511999999999") {
		t.Errorf("expected phone in formatted vCard, got: %s", result)
	}
}

// ── Task 10.6: Indicadores especiais ─────────────────────────────────────────

func TestApplyMessagePrefixes_Forwarded(t *testing.T) {
	msg := map[string]any{
		"contextInfo": map[string]any{
			"isForwarded":     true,
			"forwardingScore": float64(1),
		},
	}
	result := applyMessagePrefixes(msg, "Olá")
	if !containsStr(result, "Encaminhada") {
		t.Errorf("expected '[Encaminhada]' prefix, got: %s", result)
	}
}

func TestApplyMessagePrefixes_ForwardedMany(t *testing.T) {
	msg := map[string]any{
		"contextInfo": map[string]any{
			"isForwarded":     true,
			"forwardingScore": float64(5),
		},
	}
	result := applyMessagePrefixes(msg, "Texto")
	if !containsStr(result, "várias vezes") {
		t.Errorf("expected 'várias vezes' prefix, got: %s", result)
	}
}

func TestApplyMessagePrefixes_Ephemeral(t *testing.T) {
	msg := map[string]any{
		"contextInfo": map[string]any{
			"ephemeralSettingTimestamp": float64(1234567890),
		},
	}
	result := applyMessagePrefixes(msg, "texto")
	if !containsStr(result, "temporária") {
		t.Errorf("expected '[mensagem temporária]' prefix, got: %s", result)
	}
}

func TestIsGIF(t *testing.T) {
	msg := map[string]any{
		"videoMessage": map[string]any{
			"gifPlayback": true,
		},
	}
	if !isGIF(msg) {
		t.Error("expected isGIF = true for gifPlayback video")
	}
	if isGIF(map[string]any{}) {
		t.Error("expected isGIF = false for empty message")
	}
}

// ── Task 11.6: Eventos de grupo ───────────────────────────────────────────────

func TestHandleGroupInfo_ParticipantAdd(t *testing.T) {
	client := &mockClient{
		contacts:      []Contact{{ID: 1}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}
	svc := newTestService(client)

	data := map[string]any{
		"JID":  "123456789-987654321@g.us",
		"Join": []string{"5511999999999@s.whatsapp.net"},
	}
	payload := buildPayload(t, "sess", model.EventGroupInfo, data)
	cfg := &Config{SessionID: "sess", Enabled: true, InboxID: 1}
	_ = svc.processGroupInfo(context.Background(), cfg, payload)

	if len(client.messages) == 0 {
		t.Error("expected group notification message")
	}
	if !containsStr(client.messages[0].Content, "entrou") {
		t.Errorf("expected 'entrou' in message, got: %s", client.messages[0].Content)
	}
}

func TestHandleGroupInfo_SubjectChange(t *testing.T) {
	client := &mockClient{
		contacts:      []Contact{{ID: 1}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}
	svc := newTestService(client)

	data := map[string]any{
		"JID": "123456789-987654321@g.us",
		"Name": map[string]any{
			"Name": "Novo Nome do Grupo",
		},
	}
	payload := buildPayload(t, "sess", model.EventGroupInfo, data)
	cfg := &Config{SessionID: "sess", Enabled: true, InboxID: 1}
	_ = svc.processGroupInfo(context.Background(), cfg, payload)

	if len(client.messages) == 0 {
		t.Error("expected subject change notification")
	}
	if !containsStr(client.messages[0].Content, "Novo Nome do Grupo") {
		t.Errorf("expected new name in message, got: %s", client.messages[0].Content)
	}
}

// ── Task 12.6: Interactive messages ──────────────────────────────────────────

func TestHandleButtonResponse(t *testing.T) {
	client := &mockClient{
		contacts:      []Contact{{ID: 1}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}
	svc := newTestService(client)
	cfg := &Config{SessionID: "sess", Enabled: true, InboxID: 1}

	btnResp := map[string]any{
		"selectedDisplayText": "Confirmar",
		"selectedButtonId":    "btn_1",
	}
	svc.processButtonResponse(context.Background(), cfg, 1, "btn-msg", false, map[string]any{}, btnResp)

	if len(client.messages) == 0 {
		t.Fatal("expected button response message")
	}
	if !containsStr(client.messages[0].Content, "[Botão] Confirmar") {
		t.Errorf("expected '[Botão] Confirmar', got: %s", client.messages[0].Content)
	}
}

func TestHandleListResponse(t *testing.T) {
	client := &mockClient{
		contacts:      []Contact{{ID: 1}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}
	svc := newTestService(client)
	cfg := &Config{SessionID: "sess", Enabled: true, InboxID: 1}

	listResp := map[string]any{
		"singleSelectReply": map[string]any{
			"title":       "Opção A",
			"description": "Descrição da opção A",
		},
	}
	svc.processListResponse(context.Background(), cfg, 1, "list-msg", false, map[string]any{}, listResp)

	if len(client.messages) == 0 {
		t.Fatal("expected list response message")
	}
	if !containsStr(client.messages[0].Content, "[Lista] Opção A") {
		t.Errorf("expected '[Lista] Opção A', got: %s", client.messages[0].Content)
	}
}

func TestHandleTemplateButtonReply(t *testing.T) {
	client := &mockClient{
		contacts:      []Contact{{ID: 1}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}
	svc := newTestService(client)
	cfg := &Config{SessionID: "sess", Enabled: true, InboxID: 1}

	tmpl := map[string]any{"selectedDisplayText": "Sim, quero"}
	svc.processTemplateReply(context.Background(), cfg, 1, "tmpl-msg", false, map[string]any{}, tmpl)

	if len(client.messages) == 0 {
		t.Fatal("expected template reply message")
	}
	if !containsStr(client.messages[0].Content, "[Template] Sim, quero") {
		t.Errorf("expected '[Template] Sim, quero', got: %s", client.messages[0].Content)
	}
}

func TestHandleButtonResponse_WithStanzaID(t *testing.T) {
	cwID := 55
	client := &mockClient{
		contacts:      []Contact{{ID: 1}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}
	svc := newTestService(client)
	svc.msgRepo = &mockMsgRepoFixed{cwMsgID: &cwID}
	cfg := &Config{SessionID: "sess", Enabled: true, InboxID: 1}

	msg := map[string]any{
		"contextInfo": map[string]any{"stanzaId": "stanza-abc"},
	}
	btnResp := map[string]any{"selectedDisplayText": "OK"}
	svc.processButtonResponse(context.Background(), cfg, 1, "btn2", false, msg, btnResp)

	if len(client.messages) == 0 {
		t.Fatal("expected button response message")
	}
	if client.messages[0].SourceReplyID != cwID {
		t.Errorf("expected SourceReplyID=%d, got %d", cwID, client.messages[0].SourceReplyID)
	}
	ca := client.messages[0].ContentAttributes
	if ca == nil {
		t.Fatal("expected ContentAttributes")
	}
	inReplyTo, _ := ca["in_reply_to"].(int)
	if inReplyTo != cwID {
		t.Errorf("expected in_reply_to=%d, got %d", cwID, inReplyTo)
	}
	inReplyToExtID, _ := ca["reply_source_id"].(string)
	if inReplyToExtID != "WAID:stanza-abc" {
		t.Errorf("expected reply_source_id=WAID:stanza-abc, got %s", inReplyToExtID)
	}
}

// ── ViewOnce tests ───────────────────────────────────────────────────────────

type mockMediaDownloader struct {
	data []byte
	err  error
}

func (m *mockMediaDownloader) DownloadMediaByPath(_ context.Context, _, _ string, _, _, _ []byte, _ int, _ string) ([]byte, error) {
	return m.data, m.err
}

func TestHandleViewOnce_V2_DownloadSuccess(t *testing.T) {
	client := &mockClient{
		contacts:      []Contact{{ID: 1}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}
	svc := newTestService(client)
	svc.mediaDownloader = &mockMediaDownloader{data: []byte("fake-image-data")}
	cfg := &Config{SessionID: "sess", Enabled: true, InboxID: 1, TimeoutMediaSeconds: 60}

	vonce := map[string]any{
		"message": map[string]any{
			"imageMessage": map[string]any{
				"directPath": "/v/test",
				"mediaKey":   "AAAA",
				"mimetype":   "image/jpeg",
				"fileLength": float64(1024),
			},
		},
	}
	svc.processViewOnce(context.Background(), cfg, 1, "vo-msg", false, vonce, true, "", 0)

	if len(client.attachments) == 0 {
		t.Error("expected attachment to be uploaded for viewOnce v2 with successful download")
	}
	if len(client.messages) > 0 {
		t.Error("expected no text fallback when download succeeds")
	}
}

func TestHandleViewOnce_V2_DownloadFail_FallsBackToText(t *testing.T) {
	client := &mockClient{
		contacts:      []Contact{{ID: 1}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}
	svc := newTestService(client)
	svc.mediaDownloader = &mockMediaDownloader{err: fmt.Errorf("download failed")}
	cfg := &Config{SessionID: "sess", Enabled: true, InboxID: 1, TimeoutMediaSeconds: 60}

	vonce := map[string]any{
		"message": map[string]any{
			"imageMessage": map[string]any{
				"directPath": "/v/test",
				"mediaKey":   "AAAA",
				"mimetype":   "image/jpeg",
				"fileLength": float64(1024),
			},
		},
	}
	svc.processViewOnce(context.Background(), cfg, 1, "vo-msg", false, vonce, true, "", 0)

	if len(client.attachments) > 0 {
		t.Error("expected no attachment when download fails")
	}
	if len(client.messages) == 0 {
		t.Fatal("expected text fallback message")
	}
	if !containsStr(client.messages[0].Content, "mensagem vista uma vez") {
		t.Errorf("expected fallback text, got: %s", client.messages[0].Content)
	}
}

func TestHandleViewOnce_V1_AlwaysText(t *testing.T) {
	client := &mockClient{
		contacts:      []Contact{{ID: 1}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}
	svc := newTestService(client)
	svc.mediaDownloader = &mockMediaDownloader{data: []byte("should-not-be-used")}
	cfg := &Config{SessionID: "sess", Enabled: true, InboxID: 1}

	vonce := map[string]any{
		"message": map[string]any{
			"imageMessage": map[string]any{
				"directPath": "/v/test",
				"mediaKey":   "AAAA",
				"mimetype":   "image/jpeg",
			},
		},
	}
	svc.processViewOnce(context.Background(), cfg, 1, "vo-msg", false, vonce, false, "", 0)

	if len(client.attachments) > 0 {
		t.Error("expected no attachment for viewOnce v1")
	}
	if len(client.messages) == 0 {
		t.Fatal("expected text message for viewOnce v1")
	}
	if !containsStr(client.messages[0].Content, "mensagem vista uma vez") {
		t.Errorf("expected fallback text, got: %s", client.messages[0].Content)
	}
}
