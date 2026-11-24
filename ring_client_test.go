package ring

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	client "github.com/localitas/localitas-go"
)

// ringServer answers Ring's OAuth endpoint (a fresh access token each time),
// the vault credential holding the refresh token, and an API endpoint that
// answers apiStatus to every token.
type ringServer struct {
	*httptest.Server
	refreshes, calls atomic.Int32
}

func newRingServer(t *testing.T, apiStatus func(token string) int) *ringServer {
	t.Helper()
	s := &ringServer{}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /oauth/token", func(w http.ResponseWriter, r *http.Request) {
		n := s.refreshes.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"access_token":"access-%d","refresh_token":"refresh-%d","expires_in":3600,"token_type":"Bearer"}`, n, n)
	})
	mux.HandleFunc("GET /apps/vault/api/credentials/ring-cred/secrets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ring_refresh_token":"refresh-0"}`)
	})
	mux.HandleFunc("GET /clients_api/ring_devices", func(w http.ResponseWriter, r *http.Request) {
		s.calls.Add(1)
		status := apiStatus(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		w.WriteHeader(status)
		if status == http.StatusOK {
			fmt.Fprint(w, `{"doorbots":[]}`)
			return
		}
		fmt.Fprint(w, `{"error":"Unauthorized"}`)
	})
	s.Server = httptest.NewServer(mux)
	t.Cleanup(s.Close)
	return s
}

func newTestRingClient(t *testing.T, s *ringServer) *RingClient {
	t.Helper()
	auth := NewRingAuth(client.New(s.URL), "ring-cred", "", "hid-1")
	auth.tokenURL = s.URL + "/oauth/token"
	if err := auth.LoadFromVault(t.Context()); err != nil {
		t.Fatal(err)
	}
	return NewRingClient(auth)
}

// A token Ring keeps refusing is re-authenticated once, and the request then
// fails with Ring's 401 instead of asking again forever.
func TestDoRequest_RefusedTokenReauthenticatesOnceThenFails(t *testing.T) {
	s := newRingServer(t, func(string) int { return http.StatusUnauthorized })
	c := newTestRingClient(t, s)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	_, err := c.doRequest(ctx, http.MethodGet, s.URL+"/clients_api/ring_devices", nil)

	calls, refreshes := s.calls.Load(), s.refreshes.Load()
	if err == nil || !strings.Contains(err.Error(), "returned 401") || calls != 2 || refreshes != 2 {
		t.Fatalf("Ring was asked %d times with %d token refreshes, ending with %v; want the request and one retry after one re-authentication, then Ring's 401", calls, refreshes, err)
	}
}

// An expired token Ring refuses once is replaced, and the request succeeds
// with the new one.
func TestDoRequest_ExpiredTokenIsReplacedOnce(t *testing.T) {
	s := newRingServer(t, func(token string) int {
		if token == "access-1" {
			return http.StatusUnauthorized
		}
		return http.StatusOK
	})
	c := newTestRingClient(t, s)

	data, err := c.doRequest(t.Context(), http.MethodGet, s.URL+"/clients_api/ring_devices", nil)

	if err != nil || string(data) != `{"doorbots":[]}` {
		t.Fatalf("doRequest = %q, %v; want the devices after one re-authentication", data, err)
	}
	if calls, refreshes := s.calls.Load(), s.refreshes.Load(); calls != 2 || refreshes != 2 {
		t.Fatalf("Ring was asked %d times with %d token refreshes, want 2 and 2", calls, refreshes)
	}
}
