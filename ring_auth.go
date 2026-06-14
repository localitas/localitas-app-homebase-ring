package ring

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/localitas/localitas-go"
)

const (
	oauthURL  = "https://oauth.ring.com/oauth/token"
	clientID  = "ring_official_android"
	userAgent = "android:com.ringapp"
)

type authConfig struct {
	RT  string `json:"rt"`
	HID string `json:"hid,omitempty"`
}

type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

type RingAuth struct {
	mu             sync.RWMutex
	config         *authConfig
	accessToken    string
	expiresAt      time.Time
	hardwareID     string
	httpClient     *http.Client
	vaultClient    *client.Client
	vaultPublicID  string
	vaultSecretKey string
}

func NewRingAuth(vaultClient *client.Client, vaultPublicID, vaultSecretKey, hardwareID string) *RingAuth {
	if vaultSecretKey == "" {
		vaultSecretKey = "ring_refresh_token"
	}
	return &RingAuth{
		httpClient:     &http.Client{Timeout: 20 * time.Second},
		vaultClient:    vaultClient,
		vaultPublicID:  vaultPublicID,
		vaultSecretKey: vaultSecretKey,
		hardwareID:     hardwareID,
	}
}

func (a *RingAuth) LoadFromVault(ctx context.Context) error {
	secrets, err := a.vaultClient.VaultGetSecrets(ctx, a.vaultPublicID)
	if err != nil {
		return fmt.Errorf("vault get secrets: %w", err)
	}

	rawToken, ok := secrets[a.vaultSecretKey]
	if !ok || rawToken == "" {
		return fmt.Errorf("vault secret key %q not found in credential %s", a.vaultSecretKey, a.vaultPublicID)
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	a.config = parseAuthConfig(rawToken)
	if a.config == nil {
		return fmt.Errorf("invalid ring refresh token format")
	}

	if a.config.HID == "" && a.hardwareID != "" {
		a.config.HID = a.hardwareID
	}
	if a.config.HID != "" {
		a.hardwareID = a.config.HID
	}

	return nil
}

func (a *RingAuth) GetAccessToken(ctx context.Context) (string, error) {
	a.mu.RLock()
	if a.accessToken != "" && time.Now().UTC().Before(a.expiresAt) {
		token := a.accessToken
		a.mu.RUnlock()
		return token, nil
	}
	a.mu.RUnlock()

	return a.refreshAccessToken(ctx)
}

func (a *RingAuth) GetHardwareID() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.hardwareID
}

func (a *RingAuth) clearAccessToken() {
	a.mu.Lock()
	a.accessToken = ""
	a.mu.Unlock()
}

func (a *RingAuth) refreshAccessToken(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.config == nil || a.config.RT == "" {
		return "", fmt.Errorf("no refresh token available, load from vault first")
	}

	body := map[string]string{
		"client_id":     clientID,
		"scope":         "client",
		"grant_type":    "refresh_token",
		"refresh_token": a.config.RT,
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, oauthURL, jsonReader(bodyJSON))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("2fa-support", "true")
	req.Header.Set("2fa-code", "")
	if a.hardwareID != "" {
		req.Header.Set("hardware_id", a.hardwareID)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("oauth request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("oauth failed with status %d", resp.StatusCode)
	}

	var tokenResp oauthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decode oauth response: %w", err)
	}

	a.accessToken = tokenResp.AccessToken
	a.expiresAt = time.Now().UTC().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)

	a.config.RT = tokenResp.RefreshToken
	if a.hardwareID != "" {
		a.config.HID = a.hardwareID
	}

	newWrappedToken := encodeAuthConfig(a.config)
	go a.updateVaultToken(newWrappedToken)

	log.Printf("Ring auth refreshed, expires in %ds", tokenResp.ExpiresIn)
	return a.accessToken, nil
}

func (a *RingAuth) updateVaultToken(newToken string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	secrets, err := a.vaultClient.VaultGetSecrets(ctx, a.vaultPublicID)
	if err != nil {
		log.Printf("ring auth: failed to read vault for token update: %v", err)
		return
	}
	secrets[a.vaultSecretKey] = newToken
	log.Printf("ring auth: updated refresh token in vault")
}

func parseAuthConfig(raw string) *authConfig {
	if raw == "" {
		return nil
	}

	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return &authConfig{RT: raw}
	}

	var cfg authConfig
	if err := json.Unmarshal(decoded, &cfg); err != nil {
		return &authConfig{RT: raw}
	}
	if cfg.RT == "" {
		return &authConfig{RT: raw}
	}
	return &cfg
}

func encodeAuthConfig(cfg *authConfig) string {
	b, _ := json.Marshal(cfg)
	return base64.StdEncoding.EncodeToString(b)
}
