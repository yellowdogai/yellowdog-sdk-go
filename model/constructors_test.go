package model

import (
	"encoding/json"
	"strings"
	"testing"
)

// A constructor's whole job is the discriminator, which is the one field a caller
// cannot know. Nothing else is set, including required fields.
func TestConstructorSetsOnlyTheDiscriminator(t *testing.T) {
	source := NewAwsInstancesComputeSource()

	if source.Type != "co.yellowdog.platform.model.AwsInstancesComputeSource" {
		t.Errorf("Type = %q, want the fully-qualified type name", source.Type)
	}
	if source.Name != "" {
		t.Errorf("Name = %q; the constructor takes no arguments and sets no other field", source.Name)
	}
}

// Validate catches a bare struct literal before the request is sent rather than
// leaving it to the platform to reject.
func TestValidate(t *testing.T) {
	if err := NewAwsCredential().Validate(); err != nil {
		t.Errorf("a constructed value should validate, got %v", err)
	}

	err := (&AwsCredential{Name: "x"}).Validate()
	if err == nil {
		t.Fatal("a bare struct literal has no discriminator and must not validate")
	}
	for _, want := range []string{"AwsCredential", "NewAwsCredential"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q, so it does not tell the caller what to do", err, want)
		}
	}
}

// The discriminator has to reach the wire under the property name the platform reads,
// which is "action" for NodeAction and "type" everywhere else.
func TestConstructedValueMarshalsWithItsDiscriminator(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want string
	}{
		{"type", NewAwsCredential(), `"type":"co.yellowdog.platform.account.credentials.AwsCredential"`},
		{"action", NewNodeCreateWorkersAction(), `"action":"CREATE_WORKERS"`},
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
