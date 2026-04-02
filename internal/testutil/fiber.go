package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/gofiber/fiber/v2"
)

func NewApp() *fiber.App {
	return fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
}

func DoRequest(app *fiber.App, method, path string, body interface{}) (*http.Response, []byte) {
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		return nil, nil
	}
	respBody, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	return resp, respBody
}

func ParseResp(body []byte) map[string]interface{} {
	var m map[string]interface{}
	_ = json.Unmarshal(body, &m)
	return m
}
