package ring

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/localitas/localitas-go/httputil"
)

type handler struct {
	app *App
}

func (h *handler) handleConfigure(w http.ResponseWriter, r *http.Request) {
	var req struct {
		VaultPublicID string `json:"vault_public_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, r, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}
	if req.VaultPublicID == "" {
		writeErr(w, r, http.StatusBadRequest, "vault_public_id is required")
		return
	}

	if err := h.app.Configure(r.Context(), req.VaultPublicID); err != nil {
		writeErr(w, r, http.StatusBadGateway, "configure failed: %v", err)
		return
	}

	writeJSON(w, r, http.StatusOK, map[string]string{"status": "configured"})
}

func (h *handler) handleListDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := h.app.GetAllDevices(r.Context())
	if err != nil {
		writeDevicesErr(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, devices)
}

// writeDevicesErr answers a failed device read: 503 while the plugin has no
// Ring account, 502 when Ring's cloud failed.
func writeDevicesErr(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, errNotConfigured) {
		writeErr(w, r, http.StatusServiceUnavailable, "%v", err)
		return
	}
	writeErr(w, r, http.StatusBadGateway, "failed to fetch devices: %v", err)
}

// configuredClient is the Ring account's client, or answers 503 itself.
func (h *handler) configuredClient(w http.ResponseWriter, r *http.Request) *RingClient {
	rc := h.app.ringClient()
	if rc == nil {
		writeErr(w, r, http.StatusServiceUnavailable, "%v", errNotConfigured)
	}
	return rc
}

func (h *handler) handleGetDevice(w http.ResponseWriter, r *http.Request) {
	id, err := parseDeviceID(r)
	if err != nil {
		writeErr(w, r, http.StatusBadRequest, "invalid device id")
		return
	}

	devices, err := h.app.GetAllDevices(r.Context())
	if err != nil {
		writeDevicesErr(w, r, err)
		return
	}

	for _, d := range devices {
		if d.ID == id {
			writeJSON(w, r, http.StatusOK, d)
			return
		}
	}
	writeErr(w, r, http.StatusNotFound, "device %d not found", id)
}

func (h *handler) handleGetHealth(w http.ResponseWriter, r *http.Request) {
	id, err := parseDeviceID(r)
	if err != nil {
		writeErr(w, r, http.StatusBadRequest, "invalid device id")
		return
	}

	rc := h.configuredClient(w, r)
	if rc == nil {
		return
	}
	health, err := rc.GetHealth(r.Context(), id)
	if err != nil {
		writeErr(w, r, http.StatusBadGateway, "health check failed: %v", err)
		return
	}
	writeJSON(w, r, http.StatusOK, health)
}

func (h *handler) handleGetSnapshot(w http.ResponseWriter, r *http.Request) {
	id, err := parseDeviceID(r)
	if err != nil {
		writeErr(w, r, http.StatusBadRequest, "invalid device id")
		return
	}

	rc := h.configuredClient(w, r)
	if rc == nil {
		return
	}
	data, err := rc.GetSnapshot(r.Context(), id)
	if err != nil {
		writeErr(w, r, http.StatusBadGateway, "snapshot failed: %v", err)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Write(data)
}

func (h *handler) handleCommand(w http.ResponseWriter, r *http.Request) {
	id, err := parseDeviceID(r)
	if err != nil {
		writeErr(w, r, http.StatusBadRequest, "invalid device id")
		return
	}

	var cmd CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		writeErr(w, r, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	if cmd.Command == "" {
		writeErr(w, r, http.StatusBadRequest, "command is required")
		return
	}
	if !knownCommands[cmd.Command] {
		writeErr(w, r, http.StatusBadRequest, "unknown command: %s", cmd.Command)
		return
	}
	rc := h.configuredClient(w, r)
	if rc == nil {
		return
	}

	ctx := r.Context()
	var cmdErr error

	switch cmd.Command {
	case "light_on":
		cmdErr = rc.SetLight(ctx, id, true)
	case "light_off":
		cmdErr = rc.SetLight(ctx, id, false)
	case "siren_on":
		cmdErr = rc.SetSiren(ctx, id, true)
	case "siren_off":
		cmdErr = rc.SetSiren(ctx, id, false)
	case "chime_play":
		kind := "ding"
		if k, ok := cmd.Arguments["kind"].(string); ok {
			kind = k
		}
		cmdErr = rc.PlayChimeSound(ctx, id, kind)
	case "chime_volume":
		vol, ok := cmd.Arguments["volume"].(float64)
		if !ok {
			writeErr(w, r, http.StatusBadRequest, "volume argument required")
			return
		}
		cmdErr = rc.SetChimeVolume(ctx, id, int(vol))
	}

	if cmdErr != nil {
		writeJSON(w, r, http.StatusOK, CommandResponse{Success: false, Error: cmdErr.Error()})
		return
	}
	writeJSON(w, r, http.StatusOK, CommandResponse{Success: true})
}

var knownCommands = map[string]bool{
	"light_on": true, "light_off": true, "siren_on": true, "siren_off": true, "chime_play": true, "chime_volume": true,
}

func parseDeviceID(r *http.Request) (int64, error) {
	idStr := r.PathValue("id")
	return strconv.ParseInt(idStr, 10, 64)
}

func writeJSON(w http.ResponseWriter, r *http.Request, status int, v interface{}) {
	httputil.WriteResponse(w, r, status, v)
}

func writeErr(w http.ResponseWriter, r *http.Request, status int, format string, args ...interface{}) {
	httputil.WriteError(w, r, status, format, args...)
}
