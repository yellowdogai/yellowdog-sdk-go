package yellowdog

import (
	"io"
	"os"
	"regexp"
	"testing"
)

func TestVersionFunc(t *testing.T) {

	// Given
	originalVersion := version
	version = "  \n v1.2.3 \n "
	want := "1.2.3"

	// When
	v := Version()

	// Then
	version = originalVersion
	if v != want {
		t.Errorf(`Version() = %s, want %s`, v, want)
	}
}

func TestVersionVar(t *testing.T) {

	// Given
	want := readVersionFileContents(t)

	// When
	v := version

	// Then
	if v != string(want) {
		t.Errorf(`Version() = %s, want %s`, v, want)
	}
}

func readVersionFileContents(t *testing.T) []byte {

	f, err := os.Open("VERSION")
	if err != nil {
		t.Error("Could not open VERSION file")
	}
	defer func(f *os.File) { _ = f.Close() }(f)

	want, err := io.ReadAll(f)
	if err != nil {
		t.Error("Could not read VERSION file")
	}
	return want
}

func TestBaseUserAgent(t *testing.T) {

	// Given
	originalVersion := version
	version = "1.2.3"
	want := regexp.MustCompile(`^yd/1\.1\.0 yellowdog-sdk-go/1\.2\.3 go/\d+\.\d+\.\d+$`)

	// When
	userAgent := baseUserAgent()

	// Then
	version = originalVersion
	if !want.MatchString(userAgent) {
		t.Errorf(`BaseUserAgent() = %s, want %s`, userAgent, want)
	}
}
