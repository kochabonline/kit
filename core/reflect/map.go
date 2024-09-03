package reflect

import (
	"reflect"
)

func Map(target any) (map[string]any, error) {
	var result = make(map[string]any)

	valueOf := reflect.ValueOf(target)
	if valueOf.Kind() == reflect.Ptr {
		valueOf = valueOf.Elem()
	}
	typeOf := valueOf.Type()

	for i := 0; i < typeOf.NumField(); i++ {
		field := typeOf.Field(i)
		tag := field.Tag.Get("json")
		if tag == "" {
			tag = field.Name
		}

		value := valueOf.Field(i)
		if value.IsZero() {
			continue
		}

		switch value.Kind() {
		case reflect.Struct:
			m, err := Map(value.Interface())
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
