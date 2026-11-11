package optional

import "encoding/json"

// Optional[T] holds a value that may or may not be present.
// The zero value is absent.
//
// The fields are unexported so that [Of] is the only way to produce a present
// Optional, and [Optional.Get] or [Optional.Or] the only ways to read one. An
// exported Value field would let Optional[string]{Value: "x"} compile and mean
// absent, and would let an absent value be read as "" without the reader ever
// learning it was absent.
type Optional[T any] struct {
	value T
	valid bool
}

// Of returns an Optional containing v.
func Of[T any](v T) Optional[T] { return Optional[T]{value: v, valid: true} }

// Get returns the contained value and whether it is present. An absent Optional
// returns the zero value of T and false.
func (o Optional[T]) Get() (T, bool) { return o.value, o.valid }

// Or returns the contained value if present, otherwise def.
func (o Optional[T]) Or(def T) T {
	if o.valid {
		return o.value
	}
	return def
}

// IsZero reports whether the Optional is absent.
// encoding/json (Go 1.24+) uses this to honour the omitzero struct tag, which
// is what the generated model fields carry. omitempty never consults IsZero.
func (o Optional[T]) IsZero() bool { return !o.valid }

// MarshalJSON encodes the value as JSON, or null when absent.
func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if !o.valid {
		return []byte("null"), nil
	}
	return json.Marshal(o.value)
}

// UnmarshalJSON decodes a JSON value; a JSON null sets the Optional to absent.
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		o.valid = false
		var zero T
		o.value = zero
		return nil
	}
	o.valid = true
	return json.Unmarshal(data, &o.value)
}
