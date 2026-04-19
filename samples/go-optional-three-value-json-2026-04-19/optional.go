package optional

import (
	"bytes"
	"encoding/json"
)

// Optional represents a field that can be in three states:
//  1. Not provided (Set=false) - field was omitted from JSON
//  2. Explicitly set to null (Set=true, Null=true) - field should be cleared
//  3. Set to a value (Set=true, Null=false) - field should be updated
type Optional[T any] struct {
	Set   bool
	Null  bool
	Value T
}

func NewValue[T any](v T) Optional[T] {
	return Optional[T]{Set: true, Null: false, Value: v}
}

func NewNull[T any]() Optional[T] {
	return Optional[T]{Set: true, Null: true}
}

func NewUnset[T any]() Optional[T] {
	return Optional[T]{}
}

func (o Optional[T]) IsUnset() bool { return !o.Set }
func (o Optional[T]) IsNull() bool  { return o.Set && o.Null }
func (o Optional[T]) IsSet() bool   { return o.Set && !o.Null }

func (o Optional[T]) IsZero() bool {
	return !o.Set
}

func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if !o.Set || o.Null {
		return []byte("null"), nil
	}
	return json.Marshal(o.Value)
}

func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true
	if bytes.Equal(data, []byte("null")) {
		o.Null = true
		return nil
	}
	o.Null = false
	return json.Unmarshal(data, &o.Value)
}

func ApplyToPtr[T any](o Optional[T], current *T) *T {
	if o.IsUnset() {
		return current
	}
	if o.IsNull() {
		return nil
	}
	v := o.Value
	return &v
}
