package data

import "reflect"

// Value is a writable value.
type Value interface {
	// Kind returns value's Kind.
	Kind() reflect.Kind

	// Len returns value's length.
	// It panics if value's Kind is not Array, Chan, Map, Slice, or String.
	Len() int

	// Index returns value's i'th element.
	// It panics if value's Kind is not Array, Slice, or String or i is out of range.
	Index(i int) Value

	// Interface returns value's current value as an interface{}.
	Interface() interface{}
}

// value is a wrapper that wraps reflect.Value to comply with Value interface.
type value struct {
	reflect.Value
}

func newValue(v reflect.Value) Value {
	return value{Value: v}
}

func (v value) Index(i int) Value {
	return newValue(v.Value.Index(i))
}
