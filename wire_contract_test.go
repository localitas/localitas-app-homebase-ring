package ring

import (
	"encoding/json"
	"testing"
)

// The bodies sent to Ring's API are byte for byte what the untyped maps
// they replaced marshaled: keys sorted, nothing omitted.
func TestRingRequestBodies_Bytes(t *testing.T) {
	cases := []struct {
		name string
		body any
		want string
	}{
		{"session", newSessionRequest("hw-1"), `{"device":{"hardware_id":"hw-1","metadata":{"api_version":11,"device_model":"homebase-ring"},"os":"android"}}`},
		{"session without hardware id", newSessionRequest(""), `{"device":{"hardware_id":"","metadata":{"api_version":11,"device_model":"homebase-ring"},"os":"android"}}`},
		{"chime volume", newChimeVolumeRequest(7), `{"chime":{"settings":{"volume":7}}}`},
		{"chime sound", chimeSoundRequest{Kind: "ding"}, `{"kind":"ding"}`},
	}
	for _, c := range cases {
		b, err := json.Marshal(c.body)
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != c.want {
			t.Errorf("%s:\n got %s\nwant %s", c.name, b, c.want)
		}
	}
}

func TestCommandRequest_ReadsArguments(t *testing.T) {
	var play CommandRequest
	if err := json.Unmarshal([]byte(`{"command":"chime_play","arguments":{"kind":"motion"}}`), &play); err != nil {
		t.Fatal(err)
	}
	if play.Command != "chime_play" || play.Arguments.Kind != "motion" || play.Arguments.Volume != nil {
		t.Errorf("chime_play read as %+v", play)
	}

	var vol CommandRequest
	if err := json.Unmarshal([]byte(`{"command":"chime_volume","arguments":{"volume":5}}`), &vol); err != nil {
		t.Fatal(err)
	}
	if vol.Arguments.Volume == nil || *vol.Arguments.Volume != 5 {
		t.Errorf("chime_volume read as %+v", vol)
	}

	var bare CommandRequest
	if err := json.Unmarshal([]byte(`{"command":"light_on"}`), &bare); err != nil {
		t.Fatal(err)
	}
	if bare.Arguments.Kind != "" || bare.Arguments.Volume != nil {
		t.Errorf("light_on read as %+v", bare)
	}
}

func TestCameraData_KeepsFeaturesAndSettings(t *testing.T) {
	var cam CameraData
	if err := json.Unmarshal([]byte(`{"id":1,"kind":"doorbell_v3","features":{"motions_enabled":true},"settings":{"chime_settings":{"type":1}}}`), &cam); err != nil {
		t.Fatal(err)
	}
	if string(cam.Features) != `{"motions_enabled":true}` || string(cam.Settings) != `{"chime_settings":{"type":1}}` {
		t.Errorf("features %s settings %s", cam.Features, cam.Settings)
	}
}
