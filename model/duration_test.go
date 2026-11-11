package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDurationJSONMarshaling(t *testing.T) {
	tests := []struct {
		name     string
		input    string        // Go format input
		wantJSON string        // Expected ISO 8601 JSON
		wantDur  time.Duration // Expected duration value
	}{
		{
			name:     "30 seconds",
			input:    "30s",
			wantJSON: `"PT30S"`,
			wantDur:  30 * time.Second,
		},
		{
			name:     "1 minute",
			input:    "1m",
			wantJSON: `"PT1M"`,
			wantDur:  1 * time.Minute,
		},
		{
			name:     "1 hour 30 minutes",
			input:    "1h30m",
			wantJSON: `"PT1H30M"`,
			wantDur:  90 * time.Minute,
		},
		{
			name:     "2 hours",
			input:    "2h",
			wantJSON: `"PT2H"`,
			wantDur:  2 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := ParseDuration(tt.input)
			if err != nil {
				t.Fatalf("ParseDuration(%q) error: %v", tt.input, err)
			}

			if time.Duration(d) != tt.wantDur {
				t.Errorf("ParseDuration(%q) = %v, want %v", tt.input, time.Duration(d), tt.wantDur)
			}

			jsonBytes, err := json.Marshal(d)
			if err != nil {
				t.Fatalf("json.Marshal error: %v", err)
			}

			if string(jsonBytes) != tt.wantJSON {
				t.Errorf("json.Marshal = %s, want %s", string(jsonBytes), tt.wantJSON)
			}

			var unmarshaled Duration
			if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
				t.Fatalf("json.Unmarshal error: %v", err)
			}

			if time.Duration(unmarshaled) != tt.wantDur {
				t.Errorf("After unmarshal: got %v, want %v", time.Duration(unmarshaled), tt.wantDur)
			}
		})
	}
}

func TestDurationUnmarshalISO8601(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    time.Duration
		wantErr bool
	}{
		{
			name: "PT30S",
			json: `"PT30S"`,
			want: 30 * time.Second,
		},
		{
			name: "PT1M",
			json: `"PT1M"`,
			want: 1 * time.Minute,
		},
		{
			name: "PT1H30M",
			json: `"PT1H30M"`,
			want: 90 * time.Minute,
		},
		{
			name: "null",
			json: `null`,
			want: 0,
		},
		{
			name:    "invalid",
			json:    `"invalid"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Duration
			err := json.Unmarshal([]byte(tt.json), &d)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if time.Duration(d) != tt.want {
				t.Errorf("got %v, want %v", time.Duration(d), tt.want)
			}
		})
	}
}

func TestDurationHelpers(t *testing.T) {
	d, err := ParseDuration("1h30m")
	if err != nil {
		t.Fatalf("ParseDuration error: %v", err)
	}
	if d.ToDuration() != 90*time.Minute {
		t.Errorf("ParseDuration: got %v, want 90m", d.ToDuration())
	}

	d2 := MustParseDuration("45s")
	if d2.ToDuration() != 45*time.Second {
		t.Errorf("MustParseDuration: got %v, want 45s", d2.ToDuration())
	}

	d3 := NewDuration(2 * time.Hour)
	if d3.ToDuration() != 2*time.Hour {
		t.Errorf("NewDuration: got %v, want 2h", d3.ToDuration())
	}

	d4 := NewDuration(5 * time.Minute)
	if d4.String() != "5m0s" {
		t.Errorf("String: got %q, want %q", d4.String(), "5m0s")
	}
}

func TestDurationPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParseDuration should panic on invalid input")
		}
	}()
	MustParseDuration("invalid")
}
