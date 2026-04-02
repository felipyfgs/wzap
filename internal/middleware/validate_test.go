package middleware_test

import (
	"testing"

	"wzap/internal/middleware"

	"github.com/go-playground/validator/v10"
)

type testReq struct {
	Name  string `validate:"required"`
	Email string `validate:"required,email"`
}

func TestValidate_MissingRequired(t *testing.T) {
	req := testReq{}
	err := middleware.Validate.Struct(req)
	if err == nil {
		t.Error("expected validation error")
	}
	vErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}
	if len(vErrs) < 2 {
		t.Errorf("expected at least 2 errors, got %d", len(vErrs))
	}
}

func TestValidate_InvalidEmail(t *testing.T) {
	req := testReq{Name: "John", Email: "not-an-email"}
	err := middleware.Validate.Struct(req)
	if err == nil {
		t.Error("expected validation error for invalid email")
	}
}

func TestValidate_Valid(t *testing.T) {
	req := testReq{Name: "John", Email: "john@example.com"}
	if err := middleware.Validate.Struct(req); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
