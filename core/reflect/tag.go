package reflect

import (
	"errors"
	"reflect"
	"strconv"
)

func must(obj any) (reflect.Type, reflect.Value, error) {
	valueOf := reflect.ValueOf(obj)

	if valueOf.Kind() != reflect.Ptr {
		return nil, reflect.Value{}, errors.New("object must be a pointer")
	}

	if valueOf.IsNil() {
		return nil, reflect.Value{}, errors.New("object must not be nil")
	}

	return valueOf.Type().Elem(), valueOf.Elem(), nil
}

func SetDefaultTag(obj any) error {
	t, v, err := must(obj)
	if err != nil {
		return err
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("default")
		if tag == "" && field.Type.Kind() != reflect.Struct {
			continue
		}

		value := v.Field(i)
		if value.IsZero() {
			switch value.Kind() {
			case reflect.String:
				value.SetString(tag)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				parse, err := strconv.ParseInt(tag, 10, 64)
				if err != nil {
					return err
				}
				value.SetInt(parse)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				parse, err := strconv.ParseUint(tag, 10, 64)
				if err != nil {
					return err
				}
				value.SetUint(parse)
			case reflect.Float32, reflect.Float64:
				parse, err := strconv.ParseFloat(tag, 64)
				if err != nil {
					return err
				}
				value.SetFloat(parse)
			case reflect.Bool:
				parse, err := strconv.ParseBool(tag)
				if err != nil {
					return err
				}
				value.SetBool(parse)
			case reflect.Ptr:
				value.Set(reflect.New(value.Type().Elem()))
			case reflect.Struct:
				err := SetDefaultTag(value.Addr().Interface())
				if err != nil {
					return err
				}
			default:
				return errors.New("unsupported type")
			}
		}
	}

	return nil
}
