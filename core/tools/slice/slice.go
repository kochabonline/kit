package slice

// Contains compares a value with a list of values and returns true if the value is found in the list.
func Contains[T comparable](a T, slices []T) bool {
	const threshold = 50

	if len(slices) < threshold {
		for _, b := range slices {
			if b == a {
				return true
			}
		}
		return false
	}

	set := make(map[T]struct{}, len(slices))
	for _, b := range slices {
		set[b] = struct{}{}
	}
	_, found := set[a]
	return found
}

// MergeSlices merges multiple slices into a single slice.
func Merge[T comparable](slices ...[]T) []T {
	totalLength := 0
	for _, slice := range slices {
		totalLength += len(slice)
	}

	merged := make([]T, 0, totalLength)
	for _, slice := range slices {
		merged = append(merged, slice...)
	}
	return merged
}

// RemoveDuplicate removes duplicate values from a list.
func RemoveDuplicate[T comparable](slices []T) []T {
	if len(slices) == 0 {
		return slices
	}

	set := make(map[T]struct{}, len(slices))
	j := 0
	for _, item := range slices {
		if _, found := set[item]; !found {
			set[item] = struct{}{}
			slices[j] = item
			j++
		}
	}
	return slices[:j]
}

// MergeRemoveDuplicate merges multiple slices into a single slice and removes duplicate values.
func MergeRemoveDuplicate[T comparable](slices ...[]T) []T {
	return RemoveDuplicate(Merge(slices...))
}
