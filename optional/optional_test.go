package optional

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestOf(t *testing.T) {
	v, ok := Of("hello").Get()
	if !ok {
		t.Fatal("expected Of to produce a present Optional")
	}
	if v != "hello" {
		t.Fatalf("got %q, want %q", v, "hello")
	}
}

func TestGet_Absent(t *testing.T) {
	var absent Optional[string]
	v, ok := absent.Get()
	if ok {
		t.Error("the zero value must report itself as absent")
	}
	if v != "" {
		t.Errorf("got %q, want the zero value of the type", v)
	}
}

// Two Optionals of a comparable type compare with ==, which is what call sites
// use in place of a presence check followed by an equality check.
func TestEquality(t *testing.T) {
	var absent Optional[string]
	if Of("a") != Of("a") { //nolint:staticcheck // SA4000: deliberately compares two separately constructed values
		t.Error("equal present values should compare equal")
	}
	if Of("a") == Of("b") {
		t.Error("different present values should not compare equal")
	}
	if absent == Of("") {
		t.Error("absent should not compare equal to a present zero value")
	}
}

func TestOr(t *testing.T) {
	present := Of(5)
	if got := present.Or(10); got != 5 {
		t.Errorf("present.Or(10) = %d, want 5", got)
	}

	var absent Optional[int]
	if got := absent.Or(10); got != 10 {
		t.Errorf("absent.Or(10) = %d, want 10", got)
	}
}

func TestOr_ZeroValuePresent(t *testing.T) {
	// A present zero value must still be distinguishable from absent: Or must
	// return the stored zero, not fall through to the default.
	present := Of(0)
	if got := present.Or(99); got != 0 {
		t.Errorf("present.Or(99) = %d, want 0", got)
	}
}

func TestIsZero(t *testing.T) {
	var absent Optional[string]
	if !absent.IsZero() {
		t.Error("zero-value Optional should report IsZero() == true")
	}

	present := Of("")
	if present.IsZero() {
		t.Error("Of(\"\") should report IsZero() == false even though the value itself is the zero string")
	}
}

func TestMarshalJSON_Absent(t *testing.T) {
	var absent Optional[string]
	b, err := json.Marshal(absent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(b) != "null" {
		t.Errorf("got %s, want null", b)
	}
}

func TestMarshalJSON_PresentZeroValues(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want string
	}{
		{"empty string", Of(""), `""`},
		{"zero int", Of(0), "0"},
		{"false bool", Of(false), "false"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b, err := json.Marshal(c.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(b) != c.want {
				t.Errorf("got %s, want %s", b, c.want)
			}
		})
	}
}

func TestMarshalJSON_PresentStruct(t *testing.T) {
	type point struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	o := Of(point{X: 1, Y: 2})
	b, err := json.Marshal(o)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(b) != `{"x":1,"y":2}` {
		t.Errorf("got %s, want {\"x\":1,\"y\":2}", b)
	}
}

func TestUnmarshalJSON_Null(t *testing.T) {
	o := Of("leftover")

	if err := json.Unmarshal([]byte("null"), &o); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v, ok := o.Get()
	if ok {
		t.Error("expected the Optional to be absent after unmarshalling null")
	}
	if v != "" {
		t.Errorf("expected the value to be reset to the zero value, got %q", v)
	}
}

func TestUnmarshalJSON_Present(t *testing.T) {
	var o Optional[int]
	if err := json.Unmarshal([]byte("42"), &o); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if o != Of(42) {
		t.Errorf("got %v, want a present 42", o)
	}
}

func TestUnmarshalJSON_PresentZeroValue(t *testing.T) {
	var o Optional[int]
	if err := json.Unmarshal([]byte("0"), &o); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if o != Of(0) {
		t.Errorf("got %v; a present zero value must stay present", o)
	}
}

func TestUnmarshalJSON_InvalidPayload(t *testing.T) {
	var o Optional[int]
	err := json.Unmarshal([]byte(`"not a number"`), &o)
	if err == nil {
		t.Fatal("expected an error unmarshalling a string into Optional[int]")
	}
}

func TestJSONRoundTrip_OmitZero(t *testing.T) {
	type request struct {
		Name     string           `json:"name"`
		Priority Optional[int]    `json:"priority,omitzero"`
		Tag      Optional[string] `json:"tag,omitzero"`
	}

	r := request{
		Name:     "job",
		Priority: Of(5),
		// Tag left absent
	}

	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := `{"name":"job","priority":5}`
	if string(b) != want {
		t.Errorf("got %s, want %s", b, want)
	}

	var decoded request
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unexpected error decoding: %v", err)
	}
	if decoded.Priority != Of(5) {
		t.Errorf("expected Priority to round-trip as present(5), got %+v", decoded.Priority)
	}
	if !decoded.Tag.IsZero() {
		t.Errorf("expected Tag to round-trip as absent, got %+v", decoded.Tag)
	}
}

func TestJSONRoundTrip_PresentZeroValueSurvivesOmitZero(t *testing.T) {
	// This is the whole point of Optional[T] over a plain omitempty field:
	// an explicitly-set zero value must still be emitted, not dropped.
	type request struct {
		Enabled Optional[bool] `json:"enabled,omitzero"`
	}

	r := request{Enabled: Of(false)}

	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(b) != `{"enabled":false}` {
		t.Errorf("got %s, want {\"enabled\":false} — a present false must not be omitted", b)
	}
}

// ExampleOf renders at the top of the package's godoc page, where a caller
// wondering how to set an optional field will look first.
func ExampleOf() {
	type request struct {
		Namespace Optional[string] `json:"namespace,omitzero"`
		Priority  Optional[int]    `json:"priority,omitzero"`
	}

	// Priority is left absent, so omitzero drops it from the request body.
	req := request{Namespace: Of("my-namespace")}

	b, err := json.Marshal(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(b))

	if v, ok := req.Namespace.Get(); ok {
		fmt.Println(v)
	}
	fmt.Println(req.Priority.Or(5))
	// Output:
	// {"namespace":"my-namespace"}
	// my-namespace
	// 5
}
