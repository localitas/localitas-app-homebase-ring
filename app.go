package ring

import (
	"context"
	"encoding/json"
	"log"
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
	log.Printf("Ring credentials loaded from vault (credential: %s)", vaultPublicID)

	ringClient := NewRingClient(auth)
	if err := ringClient.CreateSession(ctx); err != nil {
		log.Printf("Ring session creation warning (non-fatal): %v", err)
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
	mux.HandleFunc("GET /api/devices", h.handleListDevices)
	mux.HandleFunc("GET /api/devices/{id}", h.handleGetDevice)
	mux.HandleFunc("GET /api/devices/{id}/health", h.handleGetHealth)
	mux.HandleFunc("GET /api/devices/{id}/snapshot", h.handleGetSnapshot)

	// Write endpoints
	mux.HandleFunc("POST /api/devices/{id}/command", client.RequireScopeFunc(client.ScopeWrite, h.handleCommand))

	// Admin endpoints
	mux.HandleFunc("POST /api/configure", client.RequireScopeFunc(client.ScopeAdmin, h.handleConfigure))
}

func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DefaultHealth)
}
