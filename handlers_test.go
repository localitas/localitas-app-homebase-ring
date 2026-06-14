package ring

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleCommand_MissingCommand(t *testing.T) {
	app := &App{}
	h := &handler{app: app}

	req := httptest.NewRequest("POST", "/api/devices/123/command", strings.NewReader(`{}`))
	req.SetPathValue("id", "123")
	w := httptest.NewRecorder()

	h.handleCommand(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleCommand_InvalidJSON(t *testing.T) {
	app := &App{}
	h := &handler{app: app}

	req := httptest.NewRequest("POST", "/api/devices/123/command", strings.NewReader(`not json`))
	req.SetPathValue("id", "123")
	w := httptest.NewRecorder()

	h.handleCommand(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleCommand_UnknownCommand(t *testing.T) {
	app := &App{}
	h := &handler{app: app}

	req := httptest.NewRequest("POST", "/api/devices/123/command", strings.NewReader(`{"command":"fly"}`))
	req.SetPathValue("id", "123")
	w := httptest.NewRecorder()

	h.handleCommand(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !strings.Contains(resp["error"], "unknown command") {
		t.Errorf("expected unknown command error, got %q", resp["error"])
	}
}

func TestHandleConfigure_MissingVaultID(t *testing.T) {
	app := &App{}
	h := &handler{app: app}

	req := httptest.NewRequest("POST", "/api/configure", strings.NewReader(`{}`))
	w := httptest.NewRecorder()

	h.handleConfigure(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleConfigure_InvalidJSON(t *testing.T) {
	app := &App{}
	h := &handler{app: app}

	req := httptest.NewRequest("POST", "/api/configure", strings.NewReader(`not json`))
	w := httptest.NewRecorder()

	h.handleConfigure(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleCommand_InvalidDeviceID(t *testing.T) {
	app := &App{}
	h := &handler{app: app}

	req := httptest.NewRequest("POST", "/api/devices/abc/command", strings.NewReader(`{"command":"light_on"}`))
	req.SetPathValue("id", "abc")
	w := httptest.NewRecorder()

	h.handleCommand(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
