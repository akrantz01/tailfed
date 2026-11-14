package configloader

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

var durationType = reflect.TypeFor[time.Duration]()

// Choicer is implemented by enum-like config types
type Choicer interface {
	Choices() []string
}

// BindFlags create command line flags for each configuration option
//
// Uses the `short` and `help` struct tags
func BindFlags(fs *pflag.FlagSet, cfg any) {
	v := reflect.ValueOf(cfg)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		panic("configloader.Bind: cfg must be a pointer to a struct")
	}

	bindStruct(fs, "", v.Elem())
}

func bindStruct(fs *pflag.FlagSet, prefix string, v reflect.Value) {
	t := v.Type()
	for i := range t.NumField() {
		field := t.Field(i)
		fieldValue := v.Field(i)

		name, _, _ := strings.Cut(field.Tag.Get("koanf"), ",")

		if field.Anonymous || !field.IsExported() || name == "" || name == "-" {
			continue
		}

		key := prefix + name
		if fieldValue.Kind() == reflect.Struct {
			bindStruct(fs, key+".", fieldValue)
		} else {
			usage := field.Tag.Get("help")
			if c, ok := reflect.Zero(field.Type).Interface().(Choicer); ok {
				usage += fmt.Sprintf(" (options: %s)", strings.Join(c.Choices(), ", "))
			}

			bindLeaf(fs, key, usage, field.Tag.Get("short"), fieldValue)
		}
	}
}

func bindLeaf(fs *pflag.FlagSet, key, usage, short string, v reflect.Value) {
	switch {
	case v.Type() == durationType:
		fs.DurationP(key, short, time.Duration(v.Int()), usage)
	case v.Kind() == reflect.String:
		fs.StringP(key, short, v.String(), usage)
	case v.Kind() == reflect.Bool:
		fs.BoolP(key, short, v.Bool(), usage)
	case v.Kind() == reflect.Int:
		fs.IntP(key, short, int(v.Int()), usage)
	case v.Kind() == reflect.Slice:
		if v.Type().Elem().Kind() == reflect.String {
			fs.StringSliceP(key, short, stringSlice(v), usage)
		}
	case v.Kind() == reflect.Map:
	default:
		panic(fmt.Sprintf("configloader.Bind: unsupported type %s for %q", v.Type(), key))
	}
}

func stringSlice(v reflect.Value) []string {
	out := make([]string, v.Len())
	for i := range out {
		out[i] = v.Index(i).String()
	}
	return out
}
