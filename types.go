package ring

import "fmt"

type CameraData struct {
	ID          int64                  `json:"id"`
	Description string                 `json:"description"`
	DeviceID    string                 `json:"device_id"`
	Kind        string                 `json:"kind"`
	LocationID  string                 `json:"location_id"`
	Address     string                 `json:"address"`
	Features    map[string]interface{} `json:"features"`
	Firmware    string                 `json:"firmware_version"`
	LEDStatus   string                 `json:"led_status"`
	BatteryLife interface{}            `json:"battery_life"`
	SirenStatus *SirenStatus           `json:"siren_status"`
	Settings    map[string]interface{} `json:"settings"`
	Alerts      *Alerts                `json:"alerts"`
}

type SirenStatus struct {
	SecondsRemaining int `json:"seconds_remaining"`
}

type Alerts struct {
	Connection string `json:"connection"`
	Battery    string `json:"battery"`
}

type ChimeData struct {
	ID          int64  `json:"id"`
	Description string `json:"description"`
	DeviceID    string `json:"device_id"`
	Kind        string `json:"kind"`
	LocationID  string `json:"location_id"`
	Firmware    string `json:"firmware_version"`
	Volume      int    `json:"volume"`
}

type BaseStationData struct {
	ID          int64  `json:"id"`
	Description string `json:"description"`
	DeviceID    string `json:"device_id"`
	Kind        string `json:"kind"`
	LocationID  string `json:"location_id"`
	Firmware    string `json:"firmware_version"`
}

type BeamBridgeData struct {
	ID          int64  `json:"id"`
	Description string `json:"description"`
	DeviceID    string `json:"device_id"`
	Kind        string `json:"kind"`
	LocationID  string `json:"location_id"`
	Firmware    string `json:"firmware_version"`
}

type DeviceHealth struct {
	Firmware          string `json:"firmware"`
	LatestFirmware    string `json:"latest_firmware"`
	RSSI              int    `json:"rssi"`
	SSID              string `json:"ssid"`
	NetworkConnection string `json:"network_connection"`
}

type RingDevice struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	DeviceType   string `json:"device_type"`
	Kind         string `json:"kind"`
	LocationID   string `json:"location_id"`
	BatteryLevel int    `json:"battery_level"`
	HasLight     bool   `json:"has_light"`
	HasSiren     bool   `json:"has_siren"`
	IsOnline     bool   `json:"is_online"`
	LEDStatus    string `json:"led_status"`
	Firmware     string `json:"firmware"`
}

type CommandRequest struct {
	Command   string                 `json:"command"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type CommandResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func CameraToDevice(cam CameraData) RingDevice {
	deviceType := "camera"
	if isDoorbell(cam.Kind) {
		deviceType = "doorbell"
	}

	hasLight := false
	hasSiren := false

	if isFloodlightOrSpotlight(cam.Kind) {
		hasLight = true
		hasSiren = true
	}

	batteryLevel := parseBatteryLevel(cam.BatteryLife)

	isOnline := true
	if cam.Alerts != nil && cam.Alerts.Connection == "offline" {
		isOnline = false
	}

	return RingDevice{
		ID:           cam.ID,
		Name:         cam.Description,
		DeviceType:   deviceType,
		Kind:         cam.Kind,
		BatteryLevel: batteryLevel,
		HasLight:     hasLight,
		HasSiren:     hasSiren,
		IsOnline:     isOnline,
		LEDStatus:    cam.LEDStatus,
		LocationID:   cam.LocationID,
		Firmware:     cam.Firmware,
	}
}

func ChimeToDevice(ch ChimeData) RingDevice {
	return RingDevice{
		ID:         ch.ID,
		Name:       ch.Description,
		DeviceType: "chime",
		Kind:       ch.Kind,
		IsOnline:   true,
		LocationID: ch.LocationID,
		Firmware:   ch.Firmware,
	}
}

func BaseStationToDevice(bs BaseStationData) RingDevice {
	return RingDevice{
		ID:         bs.ID,
		Name:       bs.Description,
		DeviceType: "base_station",
		Kind:       bs.Kind,
		IsOnline:   true,
		LocationID: bs.LocationID,
		Firmware:   bs.Firmware,
	}
}

func isDoorbell(kind string) bool {
	doorbellKinds := map[string]bool{
		"doorbell":              true,
		"doorbell_v3":           true,
		"doorbell_v4":           true,
		"doorbell_v5":           true,
		"doorbell_scallop":      true,
		"doorbell_scallop_lite": true,
		"doorbell_portal":       true,
		"doorbell_oyster":       true,
		"lpd_v1":                true,
		"lpd_v2":                true,
		"lpd_v4":                true,
		"jbox_v1":               true,
	}
	return doorbellKinds[kind]
}

func isFloodlightOrSpotlight(kind string) bool {
	kinds := map[string]bool{
		"floodlight_v1":     true,
		"floodlight_v2":     true,
		"floodlight_pro":    true,
		"spotlightw_v2":     true,
		"spotlight_pasture": true,
		"hp_cam_v1":         true,
		"hp_cam_v2":         true,
	}
	return kinds[kind]
}

func parseBatteryLevel(v interface{}) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case string:
		if val == "" {
			return -1
		}
		var n int
		fmt.Sscanf(val, "%d", &n)
		return n
	case nil:
		return -1
	default:
		return -1
	}
}
