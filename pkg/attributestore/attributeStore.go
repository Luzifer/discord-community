package attributestore

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var (
	// ErrValueNotSet signals the value with the given key is not present
	// in the ModuleAttributeStore
	ErrValueNotSet = errors.New("specified value not found")

	// ErrValueMismatch signals the value with the given key has another
	// type than the getter function expected
	ErrValueMismatch = errors.New("specified value has different format")
)

// ModuleAttributeStore is a key-value store with ability to store
// arbitrary data having conversion methods
type ModuleAttributeStore map[string]any

// Expect checks whether all given keys are present in the store
func (m ModuleAttributeStore) Expect(keys ...string) error {
	var missing []string

	for _, k := range keys {
		if _, ok := m[k]; !ok {
			missing = append(missing, k)
		}
	}

	if len(missing) > 0 {
		return errors.Errorf("missing key(s) %s", strings.Join(missing, ", "))
	}

	return nil
}

// MustBool converts the stored value to bool or panics
func (m ModuleAttributeStore) MustBool(name string, defVal *bool) bool {
	v, err := m.Bool(name)
	if err != nil {
		if defVal != nil {
			return *defVal
		}
		panic(err)
	}
	return v
}

// MustDuration converts the stored value to time.Duration or panics
func (m ModuleAttributeStore) MustDuration(name string, defVal *time.Duration) time.Duration {
	v, err := m.Duration(name)
	if err != nil {
		if defVal != nil {
			return *defVal
		}
		panic(err)
	}
	return v
}

// MustInt64 converts the stored value to int64 or panics
func (m ModuleAttributeStore) MustInt64(name string, defVal *int64) int64 {
	v, err := m.Int64(name)
	if err != nil {
		if defVal != nil {
			return *defVal
		}
		panic(err)
	}
	return v
}

// MustString converts the stored value to string or panics
func (m ModuleAttributeStore) MustString(name string, defVal *string) string {
	v, err := m.String(name)
	if err != nil {
		if defVal != nil {
			return *defVal
		}
		panic(err)
	}
	return v
}

// Bool reads the stored value as bool
func (m ModuleAttributeStore) Bool(name string) (bool, error) {
	v, ok := m[name]
	if !ok {
		return false, ErrValueNotSet
	}

	switch v := v.(type) {
	case bool:
		return v, nil
	case string:
		bv, err := strconv.ParseBool(v)
		return bv, errors.Wrap(err, "parsing string to bool")
	}

	return false, ErrValueMismatch
}

// Duration reads the stored value as time.Duration
func (m ModuleAttributeStore) Duration(name string) (time.Duration, error) {
	v, err := m.String(name)
	if err != nil {
		return 0, errors.Wrap(err, "getting string value")
	}

	d, err := time.ParseDuration(v)
	return d, errors.Wrap(err, "parsing value")
}

// Int64 reads the stored value as int64
func (m ModuleAttributeStore) Int64(name string) (int64, error) {
	v, ok := m[name]
	if !ok {
		return 0, ErrValueNotSet
	}

	switch v := v.(type) {
	case int:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	}

	return 0, ErrValueMismatch
}

// String reads the stored value as string
func (m ModuleAttributeStore) String(name string) (string, error) {
	v, ok := m[name]
	if !ok {
		return "", ErrValueNotSet
	}

	if sv, ok := v.(string); ok {
		return sv, nil
	}

	if iv, ok := v.(fmt.Stringer); ok {
		return iv.String(), nil
	}

	return "", ErrValueMismatch
}

// StringSlice reads the stored value as []string
func (m ModuleAttributeStore) StringSlice(name string) ([]string, error) {
	v, ok := m[name]
	if !ok {
		return nil, ErrValueNotSet
	}

	switch v := v.(type) {
	case []string:
		return v, nil

	case []interface{}:
		var out []string

		for _, iv := range v {
			sv, ok := iv.(string)
			if !ok {
				return nil, errors.New("value in slice was not string")
			}
			out = append(out, sv)
		}

		return out, nil
	}

	return nil, ErrValueMismatch
}
