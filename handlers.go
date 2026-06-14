package ring

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type handler struct {
	app *App
}

func (h *handler) handleConfigure(w http.ResponseWriter, r *http.Request) {
	var req struct {
		VaultPublicID string `json:"vault_public_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}
	if req.VaultPublicID == "" {
		writeErr(w, http.StatusBadRequest, "vault_public_id is required")
		return
	}

	if err := h.app.Configure(r.Context(), req.VaultPublicID); err != nil {
		writeErr(w, http.StatusBadGateway, "configure failed: %v", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "configured"})
}

func (h *handler) handleListDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := h.app.GetAllDevices(r.Context())
	if err != nil {
		writeErr(w, http.StatusBadGateway, "failed to fetch devices: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, devices)
}

func (h *handler) handleGetDevice(w http.ResponseWriter, r *http.Request) {
	id, err := parseDeviceID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid device id")
		return
	}

	devices, err := h.app.GetAllDevices(r.Context())
	if err != nil {
		writeErr(w, http.StatusBadGateway, "failed to fetch devices: %v", err)
		return
	}

	for _, d := range devices {
		if d.ID == id {
			writeJSON(w, http.StatusOK, d)
			return
		}
	}
	writeErr(w, http.StatusNotFound, "device %d not found", id)
}

func (h *handler) handleGetHealth(w http.ResponseWriter, r *http.Request) {
	id, err := parseDeviceID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid device id")
		return
	}

	health, err := h.app.RingClient.GetHealth(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "health check failed: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, health)
}

func (h *handler) handleGetSnapshot(w http.ResponseWriter, r *http.Request) {
	id, err := parseDeviceID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid device id")
		return
	}

	data, err := h.app.RingClient.GetSnapshot(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "snapshot failed: %v", err)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Write(data)
}

func (h *handler) handleCommand(w http.ResponseWriter, r *http.Request) {
	id, err := parseDeviceID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid device id")
		return
	}

	var cmd CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	if cmd.Command == "" {
		writeErr(w, http.StatusBadRequest, "command is required")
		return
	}

	ctx := r.Context()
	var cmdErr error

	switch cmd.Command {
	case "light_on":
		cmdErr = h.app.RingClient.SetLight(ctx, id, true)
	case "light_off":
		cmdErr = h.app.RingClient.SetLight(ctx, id, false)
	case "siren_on":
		cmdErr = h.app.RingClient.SetSiren(ctx, id, true)
	case "siren_off":
		cmdErr = h.app.RingClient.SetSiren(ctx, id, false)
	case "chime_play":
		kind := "ding"
		if k, ok := cmd.Arguments["kind"].(string); ok {
			kind = k
		}
		cmdErr = h.app.RingClient.PlayChimeSound(ctx, id, kind)
	case "chime_volume":
		vol, ok := cmd.Arguments["volume"].(float64)
		if !ok {
			writeErr(w, http.StatusBadRequest, "volume argument required")
			return
		}
		cmdErr = h.app.RingClient.SetChimeVolume(ctx, id, int(vol))
	default:
		writeErr(w, http.StatusBadRequest, "unknown command: %s", cmd.Command)
		return
	}

	if cmdErr != nil {
		writeJSON(w, http.StatusOK, CommandResponse{Success: false, Error: cmdErr.Error()})
		return
	}
	writeJSON(w, http.StatusOK, CommandResponse{Success: true})
}

func parseDeviceID(r *http.Request) (int64, error) {
	idStr := r.PathValue("id")
	return strconv.ParseInt(idStr, 10, 64)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
