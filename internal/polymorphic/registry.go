package polymorphic

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// Registry provides type-safe polymorphic JSON unmarshaling using a discriminator field.
// It maps discriminator values to concrete types and unmarshals JSON into the appropriate type.
type Registry[T any] struct {
	field string                  // discriminator field name (e.g., "type")
	types map[string]reflect.Type // maps discriminator value to concrete type
}

// NewRegistry creates a new polymorphic registry with the specified discriminator field name.
// The discriminator field is the JSON field used to determine which concrete type to unmarshal.
func NewRegistry[T any](discriminatorField string) *Registry[T] {
	return &Registry[T]{
		field: discriminatorField,
		types: make(map[string]reflect.Type),
	}
}

// Register associates a discriminator value with a concrete type.
// The concrete parameter must be a nil pointer to the type (e.g., (*ConcreteType)(nil)).
// This method is chainable and panics if the discriminator is already registered.
//
// Example:
//
//	registry.Register("aws", (*AWSCredential)(nil)).
//	         Register("gcp", (*GCPCredential)(nil))
func (r *Registry[T]) Register(discriminator string, concrete T) *Registry[T] {
	typ := reflect.TypeOf(concrete)
	if typ == nil {
		panic("concrete type cannot be nil")
	}

	if typ.Kind() != reflect.Pointer {
		panic("concrete must be a pointer type")
	}
	typ = typ.Elem()

	if _, exists := r.types[discriminator]; exists {
		panic("duplicate discriminator: " + discriminator)
	}

	r.types[discriminator] = typ
	return r
}

// Unmarshal unmarshals JSON data into the appropriate concrete type based on the discriminator field.
// It returns an error if the discriminator value is unknown or if unmarshaling fails.
func (r *Registry[T]) Unmarshal(data []byte) (T, error) {
	var zero T

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return zero, fmt.Errorf("extracting discriminator: %w", err)
	}

	discriminatorRaw, ok := raw[r.field]
	if !ok {
		return zero, fmt.Errorf("missing discriminator field: %q", r.field)
	}

	var discriminatorValue string
	if err := json.Unmarshal(discriminatorRaw, &discriminatorValue); err != nil {
		return zero, fmt.Errorf("parsing discriminator field %q: %w", r.field, err)
	}

	if discriminatorValue == "" {
		return zero, fmt.Errorf("unknown %s: %q", r.field, discriminatorValue)
	}

	typ, ok := r.types[discriminatorValue]
	if !ok {
		return zero, fmt.Errorf("unknown %s: %q", r.field, discriminatorValue)
	}

	ptr := reflect.New(typ)

	if err := json.Unmarshal(data, ptr.Interface()); err != nil {
		return zero, fmt.Errorf("unmarshaling %s: %w", discriminatorValue, err)
	}

	result, ok := ptr.Interface().(T)
	if !ok {
		return zero, fmt.Errorf("type %s does not implement interface", typ.Name())
	}

	return result, nil
}

// UnmarshalList unmarshals a JSON array into a slice of polymorphic types.
// Each element is unmarshaled using Unmarshal, and any error stops processing.
func (r *Registry[T]) UnmarshalList(data []byte) ([]T, error) {
	var rawList []json.RawMessage
	if err := json.Unmarshal(data, &rawList); err != nil {
		return nil, fmt.Errorf("unmarshaling array: %w", err)
	}

	result := make([]T, 0, len(rawList))
	for i, raw := range rawList {
		if string(raw) == "null" {
			var zero T
			result = append(result, zero)
			continue
		}

		item, err := r.Unmarshal(raw)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling element %d: %w", i, err)
		}
		result = append(result, item)
	}

	return result, nil
}
