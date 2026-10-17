package ring

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/localitas/localitas-go"
)

type AppHealth struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Icon        string `json:"icon"`
	Version     string `json:"version"`
	Status      string `json:"status"`
	PluginType  string `json:"plugin_type"`
	PluginFor   string `json:"plugin_for"`
}

var logger = slog.Default().With("component", "homebase-ring")

var DefaultHealth = AppHealth{
	Name:        "homebase-ring",
	DisplayName: "Ring",
	Icon:        "bell",
	Version:     "0.1.0",
	Status:      "healthy",
	PluginType:  "homebase-plugin",
	PluginFor:   "homebase",
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

	return nil
}

func (a *App) IsConfigured() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.configured
}

func (a *App) GetAllDevices(ctx context.Context) ([]RingDevice, error) {
	a.mu.RLock()
	rc := a.RingClient
	a.mu.RUnlock()

	if rc == nil {
		return nil, nil
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
