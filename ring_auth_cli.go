package ring

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type authTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

type auth2FAResponse struct {
	Error    string `json:"error"`
	TSVState string `json:"tsv_state"`
	Phone    string `json:"phone"`
}

func AcquireRefreshToken(email, password, twoFactorCode string) (string, string, error) {
	httpClient := &http.Client{Timeout: 20 * time.Second}

	grantData := map[string]string{
		"client_id":  clientID,
		"scope":      "client",
		"grant_type": "password",
		"username":   email,
		"password":   password,
	}

	body, _ := json.Marshal(grantData)
	req, err := http.NewRequest(http.MethodPost, oauthURL, bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("2fa-support", "true")
	req.Header.Set("2fa-code", twoFactorCode)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK {
		var tokenResp authTokenResponse
		if err := json.Unmarshal(respBody, &tokenResp); err != nil {
			return "", "", fmt.Errorf("decode response: %w", err)
		}
		cfg := &authConfig{RT: tokenResp.RefreshToken}
		return encodeAuthConfig(cfg), "", nil
	}

	if resp.StatusCode == 412 {
		var twoFA auth2FAResponse
		json.Unmarshal(respBody, &twoFA)

		prompt := "Please enter the 2FA code sent to your phone/email"
		if twoFA.TSVState == "totp" {
			prompt = "Please enter the code from your authenticator app"
		} else if twoFA.Phone != "" {
			prompt = fmt.Sprintf("Please enter the code sent to %s via %s", twoFA.Phone, twoFA.TSVState)
		}
		return "", prompt, nil
	}

	if resp.StatusCode == 400 {
		var errResp auth2FAResponse
		json.Unmarshal(respBody, &errResp)
		if errResp.Error != "" {
			return "", "", fmt.Errorf("%s", errResp.Error)
		}
	}

	return "", "", fmt.Errorf("authentication failed (status %d): %s", resp.StatusCode, string(respBody))
}
