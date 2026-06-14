package ring

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	clientAPIBase = "https://api.ring.com/clients_api/"
	deviceAPIBase = "https://api.ring.com/devices/v1/"
	appAPIBase    = "https://prd-api-us.prd.rings.solutions/api/v1/"
	snapshotBase  = "https://app-snaps.ring.com/"
	apiVersion    = 11
)

type RingClient struct {
	auth       *RingAuth
	httpClient *http.Client
}

func NewRingClient(auth *RingAuth) *RingClient {
	return &RingClient{
		auth:       auth,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *RingClient) CreateSession(ctx context.Context) error {
	body := map[string]interface{}{
		"device": map[string]interface{}{
			"hardware_id": c.auth.GetHardwareID(),
			"metadata": map[string]interface{}{
				"api_version":  apiVersion,
				"device_model": "homebase-ring",
			},
			"os": "android",
		},
	}

	_, err := c.doRequest(ctx, http.MethodPost, clientAPIBase+"session", body)
	return err
}

type RingDevicesResponse struct {
	Doorbots           []CameraData      `json:"doorbots"`
	Chimes             []ChimeData       `json:"chimes"`
	AuthorizedDoorbots []CameraData      `json:"authorized_doorbots"`
	StickupCams        []CameraData      `json:"stickup_cams"`
	BaseStations       []BaseStationData `json:"base_stations"`
	BeamBridges        []BeamBridgeData  `json:"beams_bridges"`
	Other              []json.RawMessage `json:"other"`
}

func (c *RingClient) FetchDevices(ctx context.Context) (*RingDevicesResponse, error) {
	data, err := c.doRequest(ctx, http.MethodGet, clientAPIBase+"ring_devices", nil)
	if err != nil {
		return nil, err
	}

	var resp RingDevicesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode ring_devices: %w", err)
	}
	return &resp, nil
}

func (c *RingClient) SetLight(ctx context.Context, deviceID int64, on bool) error {
	state := "off"
	if on {
		state = "on"
	}
	_, err := c.doRequest(ctx, http.MethodPut, fmt.Sprintf("%sdoorbots/%d/floodlight_light_%s", clientAPIBase, deviceID, state), nil)
	return err
}

func (c *RingClient) SetSiren(ctx context.Context, deviceID int64, on bool) error {
	state := "off"
	if on {
		state = "on"
	}
	_, err := c.doRequest(ctx, http.MethodPut, fmt.Sprintf("%sdoorbots/%d/siren_%s", clientAPIBase, deviceID, state), nil)
	return err
}

func (c *RingClient) GetHealth(ctx context.Context, deviceID int64) (*DeviceHealth, error) {
	data, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("%sdoorbots/%d/health", clientAPIBase, deviceID), nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		DeviceHealth DeviceHealth `json:"device_health"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode health: %w", err)
	}
	return &resp.DeviceHealth, nil
}

func (c *RingClient) GetSnapshot(ctx context.Context, deviceID int64) ([]byte, error) {
	url := fmt.Sprintf("%ssnapshots/next/%d?extras=force", snapshotBase, deviceID)

	token, err := c.auth.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "image/jpeg")
	if hid := c.auth.GetHardwareID(); hid != "" {
		req.Header.Set("hardware_id", hid)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("snapshot request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("snapshot failed with status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (c *RingClient) GetChimeHealth(ctx context.Context, deviceID int64) (*DeviceHealth, error) {
	data, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("%schimes/%d/health", clientAPIBase, deviceID), nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		DeviceHealth DeviceHealth `json:"device_health"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode chime health: %w", err)
	}
	return &resp.DeviceHealth, nil
}

func (c *RingClient) SetChimeVolume(ctx context.Context, deviceID int64, volume int) error {
	body := map[string]interface{}{
		"chime": map[string]interface{}{
			"settings": map[string]interface{}{
				"volume": volume,
			},
		},
	}
	_, err := c.doRequest(ctx, http.MethodPut, fmt.Sprintf("%schimes/%d", clientAPIBase, deviceID), body)
	return err
}

func (c *RingClient) PlayChimeSound(ctx context.Context, deviceID int64, kind string) error {
	if kind == "" {
		kind = "ding"
	}
	body := map[string]interface{}{
		"kind": kind,
	}
	_, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("%schimes/%d/play_sound", clientAPIBase, deviceID), body)
	return err
}

func (c *RingClient) doRequest(ctx context.Context, method, url string, body interface{}) ([]byte, error) {
	token, err := c.auth.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	if hid := c.auth.GetHardwareID(); hid != "" {
		req.Header.Set("hardware_id", hid)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ring request %s %s: %w", method, url, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		c.auth.clearAccessToken()
		return c.doRequest(ctx, method, url, body)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ring API %s %s returned %d: %s", method, url, resp.StatusCode, string(data))
	}

	return data, nil
}

func jsonReader(b []byte) io.Reader {
	return bytes.NewReader(b)
}
