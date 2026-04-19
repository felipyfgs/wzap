package dto_test

import (
	"testing"

	"wzap/internal/dto"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func TestSendTextReq_MissingPhone(t *testing.T) {
	req := dto.SendTextReq{Body: "hello"}
	if err := validate.Struct(req); err == nil {
		t.Error("expected validation error for missing phone")
	}
}

func TestSendTextReq_MissingBody(t *testing.T) {
	req := dto.SendTextReq{Phone: "5511999999999"}
	if err := validate.Struct(req); err == nil {
		t.Error("expected validation error for missing body")
	}
}

func TestSendTextReq_Valid(t *testing.T) {
	req := dto.SendTextReq{Phone: "5511999999999", Body: "hello"}
	if err := validate.Struct(req); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestCreateWebhookReq_MissingURL(t *testing.T) {
	req := dto.CreateWebhookReq{Events: []string{"Message"}}
	if err := validate.Struct(req); err == nil {
		t.Error("expected validation error for missing URL")
	}
}

func TestCreateWebhookReq_MissingEvents(t *testing.T) {
	req := dto.CreateWebhookReq{URL: "https://example.com/hook"}
	if err := validate.Struct(req); err == nil {
		t.Error("expected validation error for missing events")
	}
}

func TestCreateWebhookReq_Valid(t *testing.T) {
	req := dto.CreateWebhookReq{URL: "https://example.com/hook", Events: []string{"Message"}}
	if err := validate.Struct(req); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestSessionCreateReq_MissingName(t *testing.T) {
	req := dto.SessionCreateReq{}
	if err := validate.Struct(req); err == nil {
		t.Error("expected validation error for missing name")
	}
}

func TestSendButtonReq_Valid(t *testing.T) {
	req := dto.SendButtonReq{
		Phone: "5511999999999",
		Body:  "Choose an option",
		Buttons: []dto.ButtonItem{
			{ID: "1", Text: "Yes"},
			{ID: "2", Text: "No"},
		},
	}
	if err := validate.Struct(req); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestSendButtonReq_EmptyButtons(t *testing.T) {
	req := dto.SendButtonReq{
		Phone:   "5511999999999",
		Body:    "Choose an option",
		Buttons: []dto.ButtonItem{},
	}
	if err := validate.Struct(req); err == nil {
		t.Error("expected validation error for empty buttons (min=1)")
	}
}

func TestSendListReq_MissingButtonText(t *testing.T) {
	req := dto.SendListReq{
		Phone: "5511999999999",
		Title: "Title",
		Body:  "Body",
		Sections: []dto.ListSection{
			{Rows: []dto.ListRow{{ID: "1", Title: "Row 1"}}},
		},
	}
	if err := validate.Struct(req); err == nil {
		t.Error("expected validation error for missing buttonText")
	}
}

func TestPairPhoneReq_MissingPhone(t *testing.T) {
	req := dto.PairPhoneReq{}
	if err := validate.Struct(req); err == nil {
		t.Error("expected validation error for missing phone")
	}
}
