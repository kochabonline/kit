package reflect

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
)

var (
	ErrTagTargetMustBePointer = errors.New("target must be a pointer")
	ErrTagTargetMustNotBeNil  = errors.New("target must not be nil")
	ErrTagUnsupportedType     = errors.New("unsupported type")
)

// TagOption configuration options
type TagOption struct {
	tag string // tag name for default values
}

// WithTag sets the tag name
func WithTag(tag string) func(*TagOption) {
	return func(c *TagOption) {
		c.tag = tag
	}
}

// validateTarget validates whether the target object is valid
func validateTarget(target any) (reflect.Type, reflect.Value, error) {
	valueOf := reflect.ValueOf(target)

	if valueOf.Kind() != reflect.Ptr {
		return nil, reflect.Value{}, ErrTagTargetMustBePointer
	}

	if valueOf.IsNil() {
		return nil, reflect.Value{}, ErrTagTargetMustNotBeNil
	}

	return valueOf.Type().Elem(), valueOf.Elem(), nil
}

// SetDefaultTag sets default values for struct fields
func SetDefaultTag(target any, opts ...func(*TagOption)) error {
	t, v, err := validateTarget(target)
	if err != nil {
		return err
	}

	option := &TagOption{
		tag: "default",
	}

	for _, opt := range opts {
		opt(option)
	}

	return setStructDefaults(t, v, option.tag)
}

// setStructDefaults sets default values for struct fields
func setStructDefaults(t reflect.Type, v reflect.Value, tagName string) error {
	numField := t.NumField()
	for i := 0; i < numField; i++ {
		field := t.Field(i)
		tagValue := field.Tag.Get(tagName)
		fieldValue := v.Field(i)

		// Skip if field value is not zero value
		if !fieldValue.IsZero() {
			continue
		}

		// Even if tag value is empty, recursively set default values for struct types
		if tagValue == "" && field.Type.Kind() != reflect.Struct {
			continue
		}

		if err := setFieldValue(fieldValue, tagValue); err != nil {
			return err
		}
	}
	return nil
}

// parseSetValue parses string value and sets it to reflect value
func parseSetValue(value reflect.Value, str string) error {
	switch value.Kind() {
	case reflect.String:
		value.SetString(str)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return err
		}
		value.SetInt(parsed)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		parsed, err := strconv.ParseUint(str, 10, 64)
		if err != nil {
			return err
		}
		value.SetUint(parsed)
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return err
		}
		value.SetFloat(parsed)
	case reflect.Bool:
		parsed, err := strconv.ParseBool(str)
		if err != nil {
			return err
		}
		value.SetBool(parsed)
	default:
		return ErrTagUnsupportedType
	}
	return nil
}

// setFieldValue sets field value
func setFieldValue(value reflect.Value, tagValue string) error {
	switch value.Kind() {
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Bool:
		return parseSetValue(value, tagValue)
	case reflect.Ptr:
		value.Set(reflect.New(value.Type().Elem()))
	case reflect.Slice:
		return setSliceValue(value, tagValue)
	case reflect.Struct:
		return SetDefaultTag(value.Addr().Interface())
	default:
		return ErrTagUnsupportedType
	}
	return nil
}

// setSliceValue sets slice value
func setSliceValue(value reflect.Value, tagValue string) error {
	if tagValue == "" {
		return nil
	}

	tagValues := strings.Split(tagValue, ",")

	slice := reflect.MakeSlice(value.Type(), len(tagValues), len(tagValues))

	for i, val := range tagValues {
		elem := slice.Index(i)

		// For basic types, use unified parsing function
		if isBasicType(elem.Kind()) {
			if err := parseSetValue(elem, val); err != nil {
				return err
			}
		} else if elem.Kind() == reflect.Struct {
			// For struct types, recursively set default values
			if err := setFieldValue(elem, val); err != nil {
				return err
			}
		} else {
			return ErrTagUnsupportedType
		}
	}

	value.Set(slice)
	return nil
}

// isBasicType checks if it's a basic type
func isBasicType(kind reflect.Kind) bool {
	switch kind {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}
