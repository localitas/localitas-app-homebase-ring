package ring

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	client "github.com/localitas/localitas-go"
)

const testAPIToken = "lt_test_cluster_token"

func TestMain(m *testing.M) {
	os.Setenv("LOCALITAS_API_TOKEN", testAPIToken)
	client.SetIdentityKeyFromAPIToken(testAPIToken)
	os.Exit(m.Run())
}

func identity(userID, permission string) string {
	raw, _ := json.Marshal(client.TokenInfo{UserID: userID, Email: userID + "@test.local", Permission: permission})
	payload := base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte("localitas-internal-token-v1:" + testAPIToken))
	mac := hmac.New(sha256.New, []byte(base64.StdEncoding.EncodeToString(sum[:])))
	mac.Write([]byte(payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// The Ring account's cameras and configuration answer only homebase (core's
// token) and users core vouches for with the route's scope.
func TestRoutes_AdmitCoreOrScopedUsers(t *testing.T) {
	probe := func(scope client.Scope, token string) int {
		h := coreOrUser(scope, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
		req := httptest.NewRequest("POST", "/api/configure", strings.NewReader(`{}`))
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	forged := base64.StdEncoding.EncodeToString([]byte(`{"user_id":"x","permission":"admin"}`))
	for _, tc := range []struct {
		name  string
		scope client.Scope
		token string
		want  int
	}{
		{"core's token on an admin route", client.ScopeAdmin, testAPIToken, http.StatusOK},
		{"an admin user", client.ScopeAdmin, identity("root", "admin"), http.StatusOK},
		{"a write user on an admin route", client.ScopeAdmin, identity("parent", "write"), http.StatusForbidden},
		{"a read user on a read route", client.ScopeRead, identity("kid", "read"), http.StatusOK},
		{"no token on a read route", client.ScopeRead, "", http.StatusUnauthorized},
		{"a forged token", client.ScopeRead, forged, http.StatusUnauthorized},
	} {
		if got := probe(tc.scope, tc.token); got != tc.want {
			t.Errorf("%s: %d, want %d", tc.name, got, tc.want)
		}
	}
}
