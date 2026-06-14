package ring

import "testing"

func TestCameraToDevice_Doorbell(t *testing.T) {
	cam := CameraData{
		ID:          12345,
		Description: "Front Door",
		Kind:        "doorbell_v3",
		LocationID:  "loc-1",
		Firmware:    "1.2.3",
	}

	dev := CameraToDevice(cam)

	if dev.ID != 12345 {
		t.Errorf("expected id 12345, got %d", dev.ID)
	}
	if dev.Name != "Front Door" {
		t.Errorf("expected name Front Door, got %q", dev.Name)
	}
	if dev.DeviceType != "doorbell" {
		t.Errorf("expected device_type doorbell, got %q", dev.DeviceType)
	}
	if !dev.IsOnline {
		t.Error("expected device to be online")
	}
}

func TestCameraToDevice_Floodlight(t *testing.T) {
	cam := CameraData{
		ID:          99999,
		Description: "Backyard Cam",
		Kind:        "floodlight_v2",
	}

	dev := CameraToDevice(cam)

	if dev.DeviceType != "camera" {
		t.Errorf("expected device_type camera, got %q", dev.DeviceType)
	}
	if !dev.HasLight {
		t.Error("expected has_light for floodlight")
	}
	if !dev.HasSiren {
		t.Error("expected has_siren for floodlight")
	}
}

func TestCameraToDevice_Offline(t *testing.T) {
	cam := CameraData{
		ID:          11111,
		Description: "Offline Cam",
		Kind:        "stickup_cam",
		Alerts:      &Alerts{Connection: "offline"},
	}

	dev := CameraToDevice(cam)

	if dev.IsOnline {
		t.Error("expected device to be offline")
	}
}

func TestChimeToDevice(t *testing.T) {
	ch := ChimeData{
		ID:          22222,
		Description: "Living Room Chime",
		Kind:        "chime",
		Volume:      5,
	}

	dev := ChimeToDevice(ch)

	if dev.DeviceType != "chime" {
		t.Errorf("expected device_type chime, got %q", dev.DeviceType)
	}
	if dev.Name != "Living Room Chime" {
		t.Errorf("expected name Living Room Chime, got %q", dev.Name)
	}
}

func TestBaseStationToDevice(t *testing.T) {
	bs := BaseStationData{
		ID:          33333,
		Description: "Alarm Base",
		Kind:        "hub.redsky",
	}

	dev := BaseStationToDevice(bs)

	if dev.DeviceType != "base_station" {
		t.Errorf("expected device_type base_station, got %q", dev.DeviceType)
	}
}

func TestParseBatteryLevel(t *testing.T) {
	cases := []struct {
		input    interface{}
		expected int
	}{
		{float64(85), 85},
		{"100", 100},
		{"", -1},
		{nil, -1},
	}
	for _, tc := range cases {
		got := parseBatteryLevel(tc.input)
		if got != tc.expected {
			t.Errorf("parseBatteryLevel(%v) = %d, want %d", tc.input, got, tc.expected)
		}
	}
}

func TestIsDoorbell(t *testing.T) {
	if !isDoorbell("doorbell") {
		t.Error("expected doorbell to be a doorbell")
	}
	if !isDoorbell("lpd_v2") {
		t.Error("expected lpd_v2 to be a doorbell")
	}
	if isDoorbell("stickup_cam") {
		t.Error("expected stickup_cam to not be a doorbell")
	}
}

func TestIsFloodlightOrSpotlight(t *testing.T) {
	if !isFloodlightOrSpotlight("floodlight_v2") {
		t.Error("expected floodlight_v2 to be floodlight/spotlight")
	}
	if isFloodlightOrSpotlight("doorbell") {
		t.Error("expected doorbell to not be floodlight/spotlight")
	}
}
