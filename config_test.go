package yellowdog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

func TestSecretIsRedacted(t *testing.T) {
	config := DefaultConfig(ServicesSchema{DefaultURL: "https://example.com"}, APIKey{ID: "key", Secret: "hunter2"})

	outputs := map[string]string{}
	for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%q", "%x", "%d"} {
		outputs[verb] = fmt.Sprintf(verb, config.APIKey) + fmt.Sprintf(verb, config.APIKey.Secret)
	}
	marshalled, err := json.Marshal(config.APIKey)
	if err != nil {
		t.Fatal(err)
	}
	outputs["json"] = string(marshalled)
	var text, jsonLog bytes.Buffer
	slog.New(slog.NewTextHandler(&text, nil)).Info("config", "config", config, "secret", config.APIKey.Secret)
	slog.New(slog.NewJSONHandler(&jsonLog, nil)).Info("config", "config", config, "secret", config.APIKey.Secret)
	outputs["slog text"] = text.String()
	outputs["slog json"] = jsonLog.String()

	for name, output := range outputs {
		if strings.Contains(output, "hunter2") || strings.Contains(output, fmt.Sprintf("%x", "hunter2")) {
			t.Errorf("%s output contains the secret: %s", name, output)
		}
		if !strings.Contains(output, "[REDACTED]") {
			t.Errorf("%s output doesn't say [REDACTED]: %s", name, output)
		}
	}
}
