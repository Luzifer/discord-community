package main

import (
	"errors"
	"fmt"
)

var (
	errValueNotSet   = errors.New("specified value not found")
	errValueMismatch = errors.New("specified value has different format")
)

type moduleAttributeStore map[string]interface{}

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
