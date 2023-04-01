package arrayx

func UniqArray[T comparable](arr []T) []T {
	uniMap := make(map[T]struct{}, len(arr))
	dst := make([]T, 0, len(arr))
	for _, v := range arr {
		if _, ok := uniMap[v]; !ok {
			uniMap[v] = struct{}{}
			dst = append(dst, v)
		}
	}
	return dst
}
