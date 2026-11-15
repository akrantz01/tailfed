package configloader

import (
	"reflect"
)

// Defaulter is implemented by configuration types which have default values
type Defaulter interface {
	SetDefaults()
}

// applyDefaults recursively calls SetDefaults on a struct and all of its nested structs, parents first
func applyDefaults(v reflect.Value) {
	if v.Kind() != reflect.Struct || !v.CanAddr() {
		return
	}

	if d, ok := v.Addr().Interface().(Defaulter); ok {
		d.SetDefaults()
	}

	t := v.Type()
	for i := range t.NumField() {
		if t.Field(i).IsExported() {
			applyDefaults(v.Field(i))
		}
	}
}

// defaultsHook prepares the decoding target so that configured values are layered over the defaults.
func defaultsHook(from, to reflect.Value) (any, error) {
	switch {
	case to.Kind() == reflect.Map && from.Kind() == reflect.Map && to.CanSet() && !to.IsNil():
		to.SetZero()
	case to.Kind() == reflect.Struct && to.CanAddr() && to.IsZero():
		applyDefaults(to)
	}

	return from.Interface(), nil
}
