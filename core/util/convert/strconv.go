package convert

import (
	"errors"
	"strconv"
)

func Strconv[T any](s ...string) ([]T, error) {
	if len(s) == 0 {
		return []T{}, nil
	}

	result := make([]T, 0, len(s))
	var parsedValue T
	switch any(parsedValue).(type) {
	case int:
		for _, v := range s {
			i, err := strconv.Atoi(v)
			if err != nil {
				return []T{}, err
			}
			result = append(result, any(i).(T))
		}
	case int8:
		for _, v := range s {
			i, err := strconv.ParseInt(v, 10, 8)
			if err != nil {
				return []T{}, err
			}
			result = append(result, any(int8(i)).(T))
		}
	case int16:
		for _, v := range s {
			i, err := strconv.ParseInt(v, 10, 16)
			if err != nil {
				return []T{}, err
			}
			result = append(result, any(int16(i)).(T))
		}
	case int32:
		for _, v := range s {
			i, err := strconv.ParseInt(v, 10, 32)
			if err != nil {
				return []T{}, err
			}
			result = append(result, any(int32(i)).(T))
		}
	case int64:
		for _, v := range s {
			i, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return []T{}, err
			}
			result = append(result, any(i).(T))
		}
	case uint:
		for _, v := range s {
			i, err := strconv.ParseUint(v, 10, 0)
			if err != nil {
				return []T{}, err
			}
			result = append(result, any(uint(i)).(T))
		}
	case uint8:
		for _, v := range s {
			i, err := strconv.ParseUint(v, 10, 8)
			if err != nil {
				return []T{}, err
			}
			result = append(result, any(uint8(i)).(T))
		}
	case uint16:
		for _, v := range s {
			i, err := strconv.ParseUint(v, 10, 16)
			if err != nil {
				return []T{}, err
			}
			result = append(result, any(uint16(i)).(T))
		}
	case uint32:
		for _, v := range s {
			i, err := strconv.ParseUint(v, 10, 32)
			if err != nil {
				return []T{}, err
			}
			result = append(result, any(uint32(i)).(T))
		}
	case uint64:
		for _, v := range s {
			i, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				return []T{}, err
			}
			result = append(result, any(i).(T))
		}
	case float32:
		for _, v := range s {
			f, err := strconv.ParseFloat(v, 32)
			if err != nil {
				return []T{}, err
			}
			result = append(result, any(float32(f)).(T))
		}
	case float64:
		for _, v := range s {
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return []T{}, err
			}
			result = append(result, any(f).(T))
		}
	default:
		return []T{}, errors.New("unsupported type")
	}

	return result, nil
}
