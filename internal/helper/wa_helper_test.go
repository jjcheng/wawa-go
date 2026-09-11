package helper

import (
	"testing"
	"time"
)

func TestGetWAGMTOffset(t *testing.T) {
	tests := []struct {
		name       string
		timezoneID string
		at         time.Time
		want       string
		wantErr    bool
	}{
		{name: "Singapore", timezoneID: "128", at: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC), want: "GMT+08:00"},
		{name: "Sydney summer daylight saving", timezoneID: "15", at: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC), want: "GMT+11:00"},
		{name: "Sydney winter standard time", timezoneID: "15", at: time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC), want: "GMT+10:00"},
		{name: "Etc GMT uses POSIX-reversed sign", timezoneID: "370", at: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC), want: "GMT-01:00"},
		{name: "unknown", timezoneID: "0", wantErr: true},
		{name: "not numeric", timezoneID: "Singapore", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := getWAGMTOffsetAt(test.timezoneID, test.at)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Errorf("GetWAGMTOffset(%q) = %q, want %q", test.timezoneID, got, test.want)
			}
		})
	}
}

func TestWAMetaTimezoneLocationsAreLoadable(t *testing.T) {
	for id, locationName := range waMetaTimezoneLocations {
		if _, err := time.LoadLocation(locationName); err != nil {
			t.Errorf("Meta timezone ID %d has invalid location %q: %v", id, locationName, err)
		}
	}
}
