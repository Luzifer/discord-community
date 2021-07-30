package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var (
	errValueNotSet   = errors.New("specified value not found")
	errValueMismatch = errors.New("specified value has different format")
)

type moduleAttributeStore map[string]interface{}

func (m moduleAttributeStore) Expect(keys ...string) error {
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

func (m moduleAttributeStore) MustDuration(name string, defVal *time.Duration) time.Duration {
	v, err := m.Duration(name)
	if err != nil {
		if defVal != nil {
			return *defVal
		}
		panic(err)
	}
	return v
}

func (m moduleAttributeStore) MustInt64(name string, defVal *int64) int64 {
	v, err := m.Int64(name)
	if err != nil {
		if defVal != nil {
			return *defVal
		}
		panic(err)
	}
	return v
}

func (m moduleAttributeStore) MustString(name string, defVal *string) string {
	v, err := m.String(name)
	if err != nil {
		if defVal != nil {
			return *defVal
		}
		panic(err)
	}
	return v
}

func (m moduleAttributeStore) Duration(name string) (time.Duration, error) {
	v, err := m.String(name)
	if err != nil {
		return 0, errors.Wrap(err, "getting string value")
	}

	d, err := time.ParseDuration(v)
	return d, errors.Wrap(err, "parsing value")
}

func (m moduleAttributeStore) Int64(name string) (int64, error) {
	v, ok := m[name]
	if !ok {
		return 0, errValueNotSet
	}

	switch v.(type) {
	case int:
		return int64(v.(int)), nil
	case int16:
		return int64(v.(int16)), nil
	case int32:
		return int64(v.(int32)), nil
	case int64:
		return v.(int64), nil
	}

	return 0, errValueMismatch
}

func (m moduleAttributeStore) String(name string) (string, error) {
	v, ok := m[name]
	if !ok {
		return "", errValueNotSet
	}

	if sv, ok := v.(string); ok {
		return sv, nil
	}

	if iv, ok := v.(fmt.Stringer); ok {
		return iv.String(), nil
	}

	return "", errValueMismatch
}

func (m moduleAttributeStore) StringSlice(name string) ([]string, error) {
	v, ok := m[name]
	if !ok {
		return nil, errValueNotSet
	}

	switch v.(type) {
	case []string:
		return v.([]string), nil

	case []interface{}:
		var out []string

		for _, iv := range v.([]interface{}) {
			sv, ok := iv.(string)
			if !ok {
				return nil, errors.New("value in slice was not string")
			}
			out = append(out, sv)
		}

		return out, nil
	}

	return nil, errValueMismatch
}
