package yellowdog

import (
	"runtime"
	"strings"
)

const Version = "0.1.0"

func DefaultUserAgent() string {
	return "yd/1.0.0 yellowdog-sdk-go/" + Version + " go/" + strings.TrimPrefix(runtime.Version(), "go")
}
