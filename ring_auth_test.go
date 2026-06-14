package ring

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestParseAuthConfig_RawToken(t *testing.T) {
	cfg := parseAuthConfig("my-raw-refresh-token")
	if cfg.RT != "my-raw-refresh-token" {
		t.Errorf("expected RT=my-raw-refresh-token, got %q", cfg.RT)
	}
}

func TestParseAuthConfig_Base64Wrapped(t *testing.T) {
	inner := authConfig{RT: "actual-token", HID: "hw-123"}
	b, _ := json.Marshal(inner)
	encoded := base64.StdEncoding.EncodeToString(b)

	cfg := parseAuthConfig(encoded)
	if cfg.RT != "actual-token" {
		t.Errorf("expected RT=actual-token, got %q", cfg.RT)
	}
	if cfg.HID != "hw-123" {
		t.Errorf("expected HID=hw-123, got %q", cfg.HID)
	}
}

func TestParseAuthConfig_Empty(t *testing.T) {
	cfg := parseAuthConfig("")
	if cfg != nil {
		t.Error("expected nil for empty input")
	}
}

func TestEncodeAuthConfig_Roundtrip(t *testing.T) {
	original := &authConfig{RT: "token-abc", HID: "hw-456"}
	encoded := encodeAuthConfig(original)
	parsed := parseAuthConfig(encoded)

	if parsed.RT != original.RT {
		t.Errorf("RT roundtrip failed: got %q", parsed.RT)
	}
	if parsed.HID != original.HID {
		t.Errorf("HID roundtrip failed: got %q", parsed.HID)
	}
}
