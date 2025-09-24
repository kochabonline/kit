package reflect

import (
	"reflect"
)

// MapOption configuration options for struct to map conversion
type MapOption struct {
	tag       string // tag used to get field names
	skipEmpty bool   // whether to skip zero value fields
}

// WithMapTag sets the tag used to get field names
func WithMapTag(tag string) func(*MapOption) {
	return func(opt *MapOption) {
		opt.tag = tag
	}
}

// WithMapSkipEmpty sets to skip zero value fields
func WithMapSkipEmpty() func(*MapOption) {
	return func(opt *MapOption) {
		opt.skipEmpty = true
	}
}

// StructConvMap converts struct to map[string]any
// Supports recursive conversion of nested structs
func StructConvMap(target any, opts ...func(*MapOption)) (map[string]any, error) {
	// Initialize configuration, default to use json tag
	option := &MapOption{
		tag: "json",
	}

	// Apply option configurations
	for _, opt := range opts {
		opt(option)
	}

	return structToMap(target, option)
}

// structToMap performs the actual struct to map conversion
func structToMap(target any, config *MapOption) (map[string]any, error) {
	// Get reflection object of the value
	valueOf := reflect.ValueOf(target)

	// Handle pointer type
	if valueOf.Kind() == reflect.Pointer {
		if valueOf.IsNil() {
			return nil, nil
		}
		valueOf = valueOf.Elem()
	}

	// Ensure it's a struct type
	if valueOf.Kind() != reflect.Struct {
		return nil, nil
	}

	typeOf := valueOf.Type()
	numField := typeOf.NumField()

	// Pre-allocate map capacity to improve performance
	result := make(map[string]any, numField)

	// Iterate through struct fields
	for i := 0; i < numField; i++ {
		field := typeOf.Field(i)
		fieldValue := valueOf.Field(i)

		// Skip inaccessible fields
		if !fieldValue.CanInterface() {
			continue
		}

		// Get field name, prioritize using tag
		fieldName := getFieldName(field, config.tag)

		// If configured to skip zero values and current field is zero value, skip it
		if config.skipEmpty && fieldValue.IsZero() {
			continue
		}

		// Process field value
		value, err := processFieldValue(fieldValue, config)
		if err != nil {
			return nil, err
		}

		result[fieldName] = value
	}

	return result, nil
}

// getFieldName gets field name, prioritize using specified tag
func getFieldName(field reflect.StructField, tag string) string {
	if tagValue := field.Tag.Get(tag); tagValue != "" {
		return tagValue
	}
	return field.Name
}

// processFieldValue processes field value, supports nested structs
func processFieldValue(fieldValue reflect.Value, config *MapOption) (any, error) {
	switch fieldValue.Kind() {
	case reflect.Struct:
		// Recursively process nested structs
		return structToMap(fieldValue.Interface(), config)
	case reflect.Ptr:
		// Handle pointer type
		if fieldValue.IsNil() {
			return nil, nil
		}
		if fieldValue.Elem().Kind() == reflect.Struct {
			return structToMap(fieldValue.Interface(), config)
		}
		return fieldValue.Interface(), nil
	default:
		return fieldValue.Interface(), nil
	}
}
