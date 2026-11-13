package model

import (
	"encoding/json"
	"strings"
	"testing"
)

// A struct literal has to reach the wire with its discriminator, under the property name the
// platform reads, which is "action" for NodeAction and "type" everywhere else.
func TestStructLiteralMarshalsWithItsDiscriminator(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want string
	}{
		{"type", &AwsCredential{}, `"type":"co.yellowdog.platform.account.credentials.AwsCredential"`},
		{"action", &NodeCreateWorkersAction{}, `"action":"CREATE_WORKERS"`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b, err := json.Marshal(c.in)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if !strings.Contains(string(b), c.want) {
				t.Errorf("got %s, want it to contain %s", b, c.want)
			}
		})
	}
}
