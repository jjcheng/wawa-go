package feature_ai_agent_profile

import (
	"net/http"
	"testing"
)

func TestUpdateOtherSettingsValidate(t *testing.T) {
	for _, id := range []int32{-1, 0, 1} {
		request := UpdateOtherSettings{Id: id}
		got := request.Validate()
		if id <= 0 && len(got) != 1 {
			t.Fatalf("id %d: want one validation error, got %d", id, len(got))
		}
		if id > 0 && len(got) != 0 {
			t.Fatalf("id %d: empty settings should be allowed, got %v", id, got)
		}
	}
}

func TestUpdateOtherSettingsAPISettings(t *testing.T) {
	settings := (UpdateOtherSettings{}).APISettings()
	if settings.Method != http.MethodPatch || settings.Path != "/v1/ai-agent/profiles/:id/other-settings" || !settings.Auth {
		t.Fatalf("unexpected API settings: %+v", settings)
	}
}
