package ring

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/localitas/localitas-go"
)

type AppHealth struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Icon        string   `json:"icon"`
	Version     string   `json:"version"`
	Status      string   `json:"status"`
	Provides    []string `json:"provides"`
}

var logger = slog.Default().With("component", "homebase-ring")

// Homebase's plugin contract: the interface a plugin provides, and where it
// hints homebase that its devices changed.
const (
	homebasePluginInterface = "homebase-devices/v1"
	homebaseDevicesCache    = "homebase"
	homebaseDevicesChannel  = "devices"
	// ringPollInterval is how often the Ring account's devices are read:
	// Ring's cloud pushes nothing to the plugin.
	ringPollInterval = time.Minute
)

var DefaultHealth = AppHealth{
	Name:        "homebase-ring",
	DisplayName: "Ring",
	Icon:        "bell",
	Version:     "0.1.0",
	Status:      "healthy",
	Provides:    []string{homebasePluginInterface},
}

type App struct {
	mu         sync.RWMutex
	BasePath   string
	Auth       *RingAuth
	RingClient *RingClient
	client     *client.Client
	configured bool
	hardwareID string
}

func New(c *client.Client, basePath, hardwareID string) *App {
	if basePath == "" {
		basePath = "/"
	}
	return &App{
		BasePath:   basePath,
		client:     c,
		hardwareID: hardwareID,
	}
}

func (a *App) Configure(ctx context.Context, vaultPublicID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	auth := NewRingAuth(a.client, vaultPublicID, "ring_refresh_token", a.hardwareID)
	if err := auth.LoadFromVault(ctx); err != nil {
		return err
	}
	logger.Info("ring credentials loaded from vault", "credential", vaultPublicID)

	ringClient := NewRingClient(auth)
	if err := ringClient.CreateSession(ctx); err != nil {
		logger.Warn("ring session creation warning (non-fatal)", "error", err)
	}

	a.Auth = auth
	a.RingClient = ringClient
	a.configured = true

	go a.hintDevicesChanged(context.Background())
	return nil
}

// ringClient is the Ring account's client, nil until configured.
func (a *App) ringClient() *RingClient {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.RingClient
}

// WatchDevices reads the Ring account's devices every ringPollInterval
// (Ring's cloud pushes nothing) and hints homebase when they changed, until
// ctx ends. Homebase reads them only when hinted.
func (a *App) WatchDevices(ctx context.Context) {
	ticker := time.NewTicker(ringPollInterval)
	defer ticker.Stop()
	last := ""
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if a.ringClient() == nil {
			continue
		}
		devices, err := a.GetAllDevices(ctx)
		if err != nil {
			logger.Warn("ring devices not read", "error", err)
			continue
		}
		if sig := devicesSignature(devices); sig != last {
			last = sig
			a.hintDevicesChanged(ctx)
		}
	}
}

// devicesSignature identifies a set of devices and their state, whatever
// their order.
func devicesSignature(devices []RingDevice) string {
	sorted := append([]RingDevice(nil), devices...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	b, _ := json.Marshal(sorted)
	return string(b)
}

// hintDevicesChanged tells homebase to read this plugin's devices again.
func (a *App) hintDevicesChanged(ctx context.Context) {
	if a.client == nil {
		return
	}
	value, _ := json.Marshal(struct {
		Plugin string `json:"plugin"`
	}{DefaultHealth.Name})
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if _, err := a.client.Cache(homebaseDevicesCache).PubSub(homebaseDevicesChannel, client.PubSubOpts{MaxSize: 1000}).Publish(ctx, string(value)); err != nil {
		logger.Warn("homebase not hinted of a device change", "error", err)
	}
}

func (a *App) IsConfigured() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.configured
}

// errNotConfigured answers a device request before homebase has handed the
// plugin its credential.
var errNotConfigured = errors.New("not configured: no Ring account yet")

func (a *App) GetAllDevices(ctx context.Context) ([]RingDevice, error) {
	rc := a.ringClient()
	if rc == nil {
		return nil, errNotConfigured
	}

	resp, err := rc.FetchDevices(ctx)
	if err != nil {
		return nil, err
	}

	var devices []RingDevice

	for _, cam := range resp.Doorbots {
		devices = append(devices, CameraToDevice(cam))
	}
	for _, cam := range resp.AuthorizedDoorbots {
		devices = append(devices, CameraToDevice(cam))
	}
	for _, cam := range resp.StickupCams {
		devices = append(devices, CameraToDevice(cam))
	}
	for _, ch := range resp.Chimes {
		devices = append(devices, ChimeToDevice(ch))
	}
	for _, bs := range resp.BaseStations {
		devices = append(devices, BaseStationToDevice(bs))
	}

	return devices, nil
}

func (a *App) RegisterRoutes(mux *http.ServeMux) {
	h := &handler{app: a}

	mux.HandleFunc("GET /health.json", HandleHealth)

	// Read endpoints
	mux.Handle("GET /api/devices", coreOrUser(client.ScopeRead, h.handleListDevices))
	mux.Handle("GET /api/devices/{id}", coreOrUser(client.ScopeRead, h.handleGetDevice))
	mux.Handle("GET /api/devices/{id}/health", coreOrUser(client.ScopeRead, h.handleGetHealth))
	mux.Handle("GET /api/devices/{id}/snapshot", coreOrUser(client.ScopeRead, h.handleGetSnapshot))

	// Write endpoints
	mux.Handle("POST /api/devices/{id}/command", coreOrUser(client.ScopeWrite, h.handleCommand))

	// Admin endpoints
	mux.Handle("POST /api/configure", coreOrUser(client.ScopeAdmin, h.handleConfigure))
}

func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DefaultHealth)
}

// coreOrUser admits core itself (homebase calls the plugin with the API
// token core gave it) or a user core vouches for who holds scope.
func coreOrUser(scope client.Scope, handler http.HandlerFunc) http.Handler {
	asUser := client.AuthMiddleware(client.RequireScopeFunc(scope, handler))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if client.IsSystemRequest(r) {
			handler(w, r)
			return
		}
		asUser.ServeHTTP(w, r)
	})
}
