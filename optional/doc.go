// Package optional provides a generic Optional[T] type for fields that may
// or may not be present in API requests and responses.
//
// # Overview
//
// Optional[T] is a value type (not a pointer) carrying a value that may be
// absent. Its fields are unexported, so [Of] is the only way to produce a
// present Optional and the presence check cannot be skipped on read.
//
// Use [Of] to set a field, and [Optional.Get] to read one:
//
//	name := optional.Of("my-namespace")
//	if v, ok := name.Get(); ok {
//	    fmt.Println(v)
//	}
//
// Use [Optional.Or] where a default will do:
//
//	timeout := cfg.Timeout.Or(30 * time.Second)
//
// Two Optionals of a comparable T compare with ==, so a presence-and-equality
// test needs no unpacking:
//
//	if req.Status == optional.Of(model.WorkRequirementStatusCompleted) {
//
// The zero value is absent.
//
// # JSON serialisation
//
// Optional[T] fields use the "omitzero" struct tag directive. An absent field is
// omitted from JSON output; a present one is always included, even when it holds
// the zero value of T.
//
// Incoming JSON nulls and missing fields deserialise to an absent Optional.
package optional
