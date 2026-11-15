package ring

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleHealth_ReturnsManifest(t *testing.T) {
	req := httptest.NewRequest("GET", "/health.json", nil)
	w := httptest.NewRecorder()

	HandleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var health AppHealth
	if err := json.Unmarshal(w.Body.Bytes(), &health); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if health.Name != "homebase-ring" {
		t.Errorf("expected name=homebase-ring, got %q", health.Name)
	}
	if len(health.Provides) != 1 || health.Provides[0] != "homebase-devices/v1" {
		t.Errorf("health.json provides %v, want homebase's plugin interface", health.Provides)
	}
	if health.Icon != "bell" {
		t.Errorf("expected icon=bell, got %q", health.Icon)
	}
}
