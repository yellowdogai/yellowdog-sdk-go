package yellowdog

import (
	_ "embed"
	"runtime"
	"strings"
)

//go:embed VERSION
var version string

func DefaultUserAgent() string {
	return "yd/1.0.0 yellowdog-sdk-go/" + Version() + " go/" + strings.TrimPrefix(runtime.Version(), "go")
}

func Version() string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}
