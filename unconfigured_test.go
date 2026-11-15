package ring

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Until homebase hands it a credential, the plugin has no Ring account: its
// device routes answer 503, not an empty list (which homebase would take
// for "every device is gone") and not a crash on the missing client.
func TestUnconfigured_DeviceRoutesAnswerUnavailable(t *testing.T) {
	mux := http.NewServeMux()
	New(nil, "/", "").RegisterRoutes(mux)
	for _, tc := range []struct{ method, path, body string }{
		{"GET", "/api/devices", ""},
		{"GET", "/api/devices/1", ""},
		{"GET", "/api/devices/1/health", ""},
		{"GET", "/api/devices/1/snapshot", ""},
		{"POST", "/api/devices/1/command", `{"command":"light_on"}`},
	} {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		req.Header.Set("Authorization", "Bearer "+testAPIToken)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("%s %s before configuration: %d %s, want 503", tc.method, tc.path, rec.Code, rec.Body)
		}
	}
}

// Homebase is hinted only when the devices changed: the same devices in
// another order are no change; a device going offline is one.
func TestDevicesSignature_ChangesOnlyWithTheDevices(t *testing.T) {
	porch := RingDevice{ID: 1, Name: "Porch", IsOnline: true}
	chime := RingDevice{ID: 2, Name: "Chime", IsOnline: true}
	if devicesSignature([]RingDevice{porch, chime}) != devicesSignature([]RingDevice{chime, porch}) {
		t.Error("the same devices in another order counted as a change")
	}
	offline := porch
	offline.IsOnline = false
	if devicesSignature([]RingDevice{porch, chime}) == devicesSignature([]RingDevice{offline, chime}) {
		t.Error("a device going offline was not a change")
	}
}
