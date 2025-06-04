package reflect

import (
	"reflect"
)

type MapConfig struct {
	tag       string
	skipEmpty bool
}

func WithMapTag(tag string) func(*MapConfig) {
	return func(c *MapConfig) {
		c.tag = tag
	}
}

func WithMapSkipEmpty() func(*MapConfig) {
	return func(c *MapConfig) {
		c.skipEmpty = true
	}
}

func StructConvMap(target any, opts ...func(*MapConfig)) (map[string]any, error) {
	config := &MapConfig{
		tag: "json",
	}
	for _, opt := range opts {
		opt(config)
	}

	valueOf := reflect.ValueOf(target)
	if valueOf.Kind() == reflect.Ptr {
		valueOf = valueOf.Elem()
	}
	typeOf := valueOf.Type()

	result := make(map[string]any, typeOf.NumField())
	for i := range typeOf.NumField() {
		field := typeOf.Field(i)
		tag := field.Tag.Get(config.tag)
		if tag == "" {
			tag = field.Name
		}

		value := valueOf.Field(i)
		if value.IsZero() && config.skipEmpty {
			continue
		}

		switch value.Kind() {
		case reflect.Struct:
			m, err := StructConvMap(value.Interface())
			if err != nil {
				return nil, err
			}
			result[tag] = m
		default:
			result[tag] = value.Interface()
		}
	}

	return result, nil
}
