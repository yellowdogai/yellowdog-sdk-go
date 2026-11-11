package model

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/sosodev/duration"
)

// Duration is a time.Duration that is encoded in JSON as an ISO 8601 duration
// string (e.g., "PT30S", "PT1H30M"), the format the YellowDog API uses.
type Duration time.Duration

// UnmarshalJSON implements json.Unmarshaler for ISO 8601 duration strings.
func (d *Duration) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("unmarshaling duration: %w", err)
	}

	parsed, err := parseISO8601Duration(s)
	if err != nil {
		return err
	}

	*d = parsed
	return nil
}

// MarshalJSON implements json.Marshaler for ISO 8601 duration strings.
// Sub-second precision is dropped.
func (d Duration) MarshalJSON() ([]byte, error) {
	td := time.Duration(d)

	seconds := int64(td.Seconds())

	if seconds == 0 {
		return json.Marshal("PT0S")
	}

	var result string

	hours := seconds / 3600
	seconds %= 3600
	minutes := seconds / 60
	seconds %= 60

	result = "PT"
	if hours > 0 {
		result += fmt.Sprintf("%dH", hours)
	}
	if minutes > 0 {
		result += fmt.Sprintf("%dM", minutes)
	}
	if seconds > 0 || (hours == 0 && minutes == 0) {
		result += fmt.Sprintf("%dS", seconds)
	}

	return json.Marshal(result)
}

// ToDuration converts Duration to time.Duration.
func (d Duration) ToDuration() time.Duration {
	return time.Duration(d)
}

// String returns the duration as a string (e.g., "1h30m0s").
func (d Duration) String() string {
	return time.Duration(d).String()
}

// NewDuration creates a Duration from a time.Duration.
func NewDuration(td time.Duration) Duration {
	return Duration(td)
}

// ParseDuration parses a Go duration string (e.g., "30s", "1h30m") into a Duration.
func ParseDuration(s string) (Duration, error) {
	td, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("parsing Go duration %q: %w", s, err)
	}
	return Duration(td), nil
}

// MustParseDuration is like ParseDuration but panics on error.
func MustParseDuration(s string) Duration {
	d, err := ParseDuration(s)
	if err != nil {
		panic(err)
	}
	return d
}

func parseISO8601Duration(s string) (Duration, error) {
	parsed, err := duration.Parse(s)
	if err != nil {
		return 0, fmt.Errorf("parsing ISO 8601 duration %q: %w", s, err)
	}
	return Duration(parsed.ToTimeDuration()), nil
}
