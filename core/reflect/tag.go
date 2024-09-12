package reflect

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
)

type TagConfig struct {
	tag string
}

func WithTagTag(tag string) func(*TagConfig) {
	return func(c *TagConfig) {
		c.tag = tag
	}
}

func must(target any) (reflect.Type, reflect.Value, error) {
	valueOf := reflect.ValueOf(target)

	if valueOf.Kind() != reflect.Ptr {
		return nil, reflect.Value{}, errors.New("target must be a pointer")
	}

	if valueOf.IsNil() {
		return nil, reflect.Value{}, errors.New("target must not be nil")
	}

	return valueOf.Type().Elem(), valueOf.Elem(), nil
}

func SetDefaultTag(target any, opts ...func(*TagConfig)) error {
	t, v, err := must(target)
	if err != nil {
		return err
	}

	config := &TagConfig{
		tag: "default",
	}
	for _, opt := range opts {
		opt(config)
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get(config.tag)
		// Even if the default tag value is empty, as long as it is a struct type,
		// the default value needs to be set recursively
		if tag == "" && field.Type.Kind() != reflect.Struct {
			continue
		}

		value := v.Field(i)
		if !value.IsZero() {
			continue
		}

		if err := setValue(value, tag); err != nil {
			return err
		}
	}

	return nil
}

func setValue(value reflect.Value, tag string) error {
	switch value.Kind() {
	case reflect.String:
		value.SetString(tag)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if parse, err := strconv.ParseInt(tag, 10, 64); err == nil {
			value.SetInt(parse)
		} else {
			return err
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if parse, err := strconv.ParseUint(tag, 10, 64); err == nil {
			value.SetUint(parse)
		} else {
			return err
		}
	case reflect.Float32, reflect.Float64:
		if parse, err := strconv.ParseFloat(tag, 64); err == nil {
			value.SetFloat(parse)
		} else {
			return err
		}
	case reflect.Bool:
		if parse, err := strconv.ParseBool(tag); err == nil {
			value.SetBool(parse)
		} else {
			return err
		}
	case reflect.Ptr:
		value.Set(reflect.New(value.Type().Elem()))
	case reflect.Slice:
		if err := setSliceValue(value, tag); err != nil {
			return err
		}
	case reflect.Struct:
		if err := SetDefaultTag(value.Addr().Interface()); err != nil {
			return err
		}
	default:
		return errors.New("unsupported type")
	}
	return nil
}

func setSliceValue(value reflect.Value, tag string) error {
	elemKind := value.Type().Elem().Kind()
	tags := strings.Split(tag, ",")
	length := len(tags)
	slice := reflect.MakeSlice(value.Type(), length, length)

	for i, s := range tags {
		elem := slice.Index(i)
		switch elemKind {
		case reflect.String:
			elem.SetString(s)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if parse, err := strconv.ParseInt(s, 10, 64); err == nil {
				elem.SetInt(parse)
			} else {
				return err
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if parse, err := strconv.ParseUint(s, 10, 64); err == nil {
				elem.SetUint(parse)
			} else {
				return err
			}
		case reflect.Float32, reflect.Float64:
			if parse, err := strconv.ParseFloat(s, 64); err == nil {
				elem.SetFloat(parse)
			} else {
				return err
			}
		case reflect.Bool:
			if parse, err := strconv.ParseBool(s); err == nil {
				elem.SetBool(parse)
			} else {
				return err
			}
		case reflect.Struct:
			if err := setValue(elem, s); err != nil {
				return err
			}
		default:
			return errors.New("unsupported type")
		}
	}

	value.Set(slice)
	return nil
}
