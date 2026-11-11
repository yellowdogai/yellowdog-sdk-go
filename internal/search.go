package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/yellowdogai/yellowdog-sdk-go/optional"
	"github.com/yellowdogai/yellowdog-sdk-go/pagination"
)

// SearchClient retrieves the results of a search, one slice at a time: Slice retrieves a
// single slice, and ListAll and Iterate walk every slice.
type SearchClient[T any] struct {
	httpClient  *HTTPClient
	path        string
	search      interface{}
	unmarshaler func([]byte) (T, error)
	params      []Param
}

type Param struct {
	Name  string
	Value string
}

func NewSearchClient[T any](httpClient *HTTPClient, path string, search interface{}, params ...Param) *SearchClient[T] {
	return &SearchClient[T]{
		httpClient: httpClient,
		path:       path,
		search:     search,
		params:     params,
	}
}

func NewPolymorphicSearchClient[T any](httpClient *HTTPClient, path string, search interface{}, unmarshaler func([]byte) (T, error), params ...Param) *SearchClient[T] {
	return &SearchClient[T]{
		httpClient:  httpClient,
		path:        path,
		search:      search,
		unmarshaler: unmarshaler,
		params:      params,
	}
}

// Slice retrieves a slice of items from the point and with the size given by
// sliceReference. The zero SliceReference retrieves the first slice at the platform's
// default size; pass the returned NextSliceId as the next SliceId to continue.
func (s *SearchClient[T]) Slice(ctx context.Context, sliceReference pagination.SliceReference) (pagination.Slice[T], error) {
	return s.fetchSlice(ctx, s.buildPath(sliceReference))
}

// ListAll lists all the items.
//
// Warning: ListAll retrieves every slice before returning. See Slice for more control.
func (s *SearchClient[T]) ListAll(ctx context.Context) ([]T, error) {
	var items []T
	for item, err := range s.Iterate(ctx) {
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// Iterate iterates over all the items, retrieving each slice when the iteration reaches
// it. An error ends the iteration and is yielded with the zero value of T.
//
// Warning: Iterate retrieves every slice unless the loop stops early. See Slice for more
// control.
func (s *SearchClient[T]) Iterate(ctx context.Context) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var sliceReference pagination.SliceReference
		for {
			slice, err := s.Slice(ctx, sliceReference)
			if err != nil {
				var zero T
				yield(zero, err)
				return
			}
			for _, item := range slice.Items {
				if !yield(item, nil) {
					return
				}
			}
			// An empty nextSliceId ends the search as an absent one does: it
			// cannot be sent back as a sliceId.
			next, ok := slice.NextSliceId.Get()
			if !ok || next == "" || len(slice.Items) == 0 {
				return
			}
			sliceReference = pagination.SliceReference{SliceId: optional.Of(next)}
		}
	}
}

func (s *SearchClient[T]) fetchSlice(ctx context.Context, path string) (pagination.Slice[T], error) {
	if s.unmarshaler != nil {
		var response pagination.Slice[json.RawMessage]
		if err := s.httpClient.Get(ctx, path, &response); err != nil {
			return pagination.Slice[T]{}, fmt.Errorf("searching: %w", err)
		}
		items := make([]T, 0, len(response.Items))
		for _, raw := range response.Items {
			item, err := s.unmarshaler(raw)
			if err != nil {
				return pagination.Slice[T]{}, fmt.Errorf("searching: decoding item: %w", err)
			}
			items = append(items, item)
		}
		return pagination.Slice[T]{Items: items, NextSliceId: response.NextSliceId}, nil
	}

	var response pagination.Slice[T]
	if err := s.httpClient.Get(ctx, path, &response); err != nil {
		return pagination.Slice[T]{}, fmt.Errorf("searching: %w", err)
	}
	return response, nil
}

func (s *SearchClient[T]) buildPath(sliceReference pagination.SliceReference) string {
	params := url.Values{}

	for _, param := range s.params {
		params.Add(param.Name, param.Value)
	}

	if sliceId, ok := sliceReference.SliceId.Get(); ok {
		params.Add("sliceId", sliceId)
	}

	if size, ok := sliceReference.Size.Get(); ok {
		params.Add("size", strconv.Itoa(size))
	}

	if s.search != nil {
		s.addSearchParams(params, s.search)
	}

	return s.path + "?" + params.Encode()
}

// addSearchParams flattens a search struct into query parameters.
func (s *SearchClient[T]) addSearchParams(params url.Values, search interface{}) {
	v := reflect.ValueOf(search)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return
	}

	addStructParams(params, "", v)
}

// addStructParams adds one query parameter per set field of v, each named after
// its json tag. prefix is empty for the search itself and the parameter name of
// the parent for a nested struct, whose fields are flattened as "parent.child".
func addStructParams(params url.Values, prefix string, v reflect.Value) {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		fieldType := t.Field(i)
		if !fieldType.IsExported() {
			continue
		}

		jsonTag := fieldType.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		field := v.Field(i)
		if field.IsZero() {
			continue
		}

		fieldName, _, _ := strings.Cut(jsonTag, ",")
		if prefix != "" {
			fieldName = prefix + "." + fieldName
		}
		addParam(params, fieldName, field)
	}
}

// addParam adds name=value, unwrapping optionals and pointers, flattening a
// nested struct and repeating the parameter once per element of a slice.
func addParam(params url.Values, name string, v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			addParam(params, name, v.Elem())
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			addParam(params, name, v.Index(i))
		}
	case reflect.Struct:
		if isOptionalValue(v) {
			if value, present := optionalValue(v); present {
				addParam(params, name, value)
			}
			return
		}
		if t, ok := v.Interface().(time.Time); ok {
			params.Add(name, t.Format(time.RFC3339))
			return
		}
		addStructParams(params, name, v)
	case reflect.String:
		params.Add(name, v.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		params.Add(name, strconv.FormatInt(v.Int(), 10))
	case reflect.Bool:
		params.Add(name, strconv.FormatBool(v.Bool()))
	case reflect.Float32, reflect.Float64:
		params.Add(name, strconv.FormatFloat(v.Float(), 'f', -1, 64))
	default:
		params.Add(name, fmt.Sprintf("%v", v.Interface()))
	}
}

// isOptionalValue reports whether v is an optional.Optional[T] value.
func isOptionalValue(v reflect.Value) bool {
	if v.Kind() != reflect.Struct {
		return false
	}
	return v.Type().PkgPath() == "github.com/yellowdogai/yellowdog-sdk-go/optional"
}

// optionalValue returns the value held by an optional.Optional[T], and whether
// one is present. Optional keeps its value and validity unexported, so calling
// Get is the only way to read it: reflecting on field names finds nothing and
// silently drops the parameter.
func optionalValue(v reflect.Value) (reflect.Value, bool) {
	get := v.MethodByName("Get")
	if !get.IsValid() || get.Type().NumIn() != 0 || get.Type().NumOut() != 2 {
		return reflect.Value{}, false
	}
	out := get.Call(nil)
	if out[1].Kind() != reflect.Bool || !out[1].Bool() {
		return reflect.Value{}, false
	}
	return out[0], true
}
