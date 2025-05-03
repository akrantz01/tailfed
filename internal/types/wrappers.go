package types

import (
	"encoding/json"
	"math"
	"reflect"
	"time"
)

const (
	epsilon    = 1e-9
	epsilonInv = 1 - epsilon
)

// Duration wraps a [time.Duration] to support serializing as nanoseconds or a [time.ParseDuration] string.
type Duration time.Duration

var _ json.Marshaler = (*Duration)(nil)
var _ json.Unmarshaler = (*Duration)(nil)

func (d *Duration) String() string {
	return time.Duration(*d).String()
}

func (d *Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(raw []byte) error {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}

	switch v := value.(type) {
	case float64:
		i, frac := math.Modf(math.Abs(v))
		if frac < epsilon || frac > epsilonInv {
			*d = Duration(time.Duration(i))
		} else {
			return &json.UnmarshalTypeError{
				Value: "int64",
				Type:  reflect.TypeFor[int64](),
			}
		}

	case string:
		if parsed, err := time.ParseDuration(v); err != nil {
			return err
		} else {
			*d = Duration(parsed)
		}

	default:
		return &json.UnmarshalTypeError{
			Value: "duration",
			Type:  reflect.TypeFor[Duration](),
		}
	}

	return nil
}
