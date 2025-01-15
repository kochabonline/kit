package jwt

import (
	"reflect"
)

func JwtMapClaimsReflect[T any](m map[string]any, key string) T {
	var parsed T
	value, ok := m[key]
	if !ok {
		return parsed
	}

	val := reflect.ValueOf(value)
	parsedVal := reflect.ValueOf(&parsed).Elem()

	if val.Type().ConvertibleTo(parsedVal.Type()) {
		parsedVal.Set(val.Convert(parsedVal.Type()))
	} else if val.Kind() == reflect.Slice && parsedVal.Kind() == reflect.Slice {
		elemType := parsedVal.Type().Elem()
		slice := reflect.MakeSlice(parsedVal.Type(), val.Len(), val.Cap())
		for i := 0; i < val.Len(); i++ {
			elem := val.Index(i)
			if elem.Kind() == reflect.Interface && elem.Elem().Type().ConvertibleTo(elemType) {
				slice.Index(i).Set(elem.Elem().Convert(elemType))
			} else if elem.Type().ConvertibleTo(elemType) {
				slice.Index(i).Set(elem.Convert(elemType))
			} else {
				return parsed
			}
		}
		parsedVal.Set(slice)
	}

	return parsed
}

func JwtMapClaimsParse[T any](m map[string]any, key string) T {
	var parsed T
	value, ok := m[key]
	if !ok {
		return parsed
	}

	switch any(parsed).(type) {
	case int:
		if v, ok := value.(float64); ok {
			return any(int(v)).(T)
		}
	case int64:
		if v, ok := value.(float64); ok {
			return any(int64(v)).(T)
		}
	case uint:
		if v, ok := value.(float64); ok {
			return any(uint(v)).(T)
		}
	case uint64:
		if v, ok := value.(float64); ok {
			return any(uint64(v)).(T)
		}
	case float64:
		if v, ok := value.(float64); ok {
			return any(v).(T)
		}
	case string:
		if v, ok := value.(string); ok {
			return any(v).(T)
		}
	case bool:
		if v, ok := value.(bool); ok {
			return any(v).(T)
		}
	case []string:
		if v, ok := value.([]any); ok {
			result := make([]string, len(v))
			for i, elem := range v {
				if str, ok := elem.(string); ok {
					result[i] = str
				} else {
					return parsed
				}
			}
			return any(result).(T)
		}
	case []int:
		if v, ok := value.([]any); ok {
			result := make([]int, len(v))
			for i, elem := range v {
				if num, ok := elem.(float64); ok {
					result[i] = int(num)
				} else {
					return parsed
				}
			}
			return any(result).(T)
		}
	case []int64:
		if v, ok := value.([]any); ok {
			result := make([]int64, len(v))
			for i, elem := range v {
				if num, ok := elem.(float64); ok {
					result[i] = int64(num)
				} else {
					return parsed
				}
			}
			return any(result).(T)
		}
	default:
		return JwtMapClaimsReflect[T](m, key)
	}

	return parsed
}
