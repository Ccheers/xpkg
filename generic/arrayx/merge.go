package arrayx

func Merge[T any](slices ...[]T) []T {
	result := make([]T, 0, len(slices))
	for _, slice := range slices {
		result = append(result, slice...)
	}
	return result
}
