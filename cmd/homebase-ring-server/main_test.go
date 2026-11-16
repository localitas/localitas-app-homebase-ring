package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
)

type vaultCredential struct {
	Name string `json:"name"`
	Data struct {
		RingRefreshToken string `json:"ring_refresh_token"`
	} `json:"data"`
}

type storedRequest struct {
	method        string
	path          string
	authorization string
	contentType   string
	body          []byte
}

func printedCurl(t *testing.T, instructions string) string {
	t.Helper()
	var lines []string
	for _, line := range strings.Split(instructions, "\n") {
		if strings.HasPrefix(line, "  curl ") || (len(lines) > 0 && strings.HasPrefix(line, "    -")) {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		t.Fatalf("no curl command in:\n%s", instructions)
	}
	return strings.Join(lines, "\n")
}

func runPrinted(t *testing.T, refreshToken string) storedRequest {
	t.Helper()
	got := make(chan storedRequest, 1)
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got <- storedRequest{r.Method, r.URL.Path, r.Header.Get("Authorization"), r.Header.Get("Content-Type"), body}
		w.WriteHeader(http.StatusCreated)
	}))
	defer core.Close()

	cmd := exec.Command("sh", "-c", printedCurl(t, vaultStoreInstructions(core.URL, refreshToken))+" --fail --silent --show-error")
	cmd.Env = []string{"PATH=/usr/bin:/bin:/usr/local/bin:/opt/homebrew/bin", "HOME=" + t.TempDir(), "LOCALITAS_API_TOKEN=lt_core_api_token"}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("printed command failed: %v\n%s", err, out)
	}
	return <-got
}

func TestPrintedVaultCommandStoresTheRefreshTokenWithCoresAPIToken(t *testing.T) {
	req := runPrinted(t, "ring-refresh-token")

	if req.method != http.MethodPost || req.path != "/apps/vault/api/credentials" {
		t.Fatalf("request = %s %s, want POST /apps/vault/api/credentials", req.method, req.path)
	}
	if req.authorization != "Bearer lt_core_api_token" {
		t.Fatalf("Authorization = %q, want the token from $LOCALITAS_API_TOKEN", req.authorization)
	}
	if req.contentType != "application/json" {
		t.Fatalf("Content-Type = %q", req.contentType)
	}
	var cred vaultCredential
	if err := json.Unmarshal(req.body, &cred); err != nil {
		t.Fatalf("body %q: %v", req.body, err)
	}
	if cred.Name != "Ring" || cred.Data.RingRefreshToken != "ring-refresh-token" {
		t.Fatalf("credential = %+v", cred)
	}
}

func TestPrintedVaultCommandSendsValidJSONForAnyToken(t *testing.T) {
	token := `a"b\c/d+e='f`
	var cred vaultCredential
	if err := json.Unmarshal(runPrinted(t, token).body, &cred); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if cred.Data.RingRefreshToken != token {
		t.Fatalf("stored %q, want %q", cred.Data.RingRefreshToken, token)
	}
}

func TestInstructionsNameWhereCoresAPITokenComesFrom(t *testing.T) {
	text := vaultStoreInstructions("http://localhost:8090", "tok")
	for _, want := range []string{"LOCALITAS_API_TOKEN", "core.auth.api_token", "~/.localitas/config-core.yaml"} {
		if !strings.Contains(text, want) {
			t.Errorf("instructions lack %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "api-token)") || strings.Contains(text, ":8080") {
		t.Errorf("instructions still name the api-token file or port 8080:\n%s", text)
	}
}

func TestAuthCoreURLDefaultsToCoresPort(t *testing.T) {
	t.Setenv("CORE_URL", "")
	for _, f := range authCommand().Flags {
		if sf, ok := f.(*cli.StringFlag); ok && sf.Name == "core-url" {
			if sf.Value != "http://localhost:8090" {
				t.Fatalf("core-url default = %q, want http://localhost:8090", sf.Value)
			}
			return
		}
	}
	t.Fatal("auth has no --core-url flag")
}
