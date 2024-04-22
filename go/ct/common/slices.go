package common

func RightPadSlice[T any](source []T, size int) []T {
	res := make([]T, size)
	copy(res, source)
	return res
}

func LeftPadSlice[T any](source []T, size int) []T {
	res := make([]T, size)
	if size < len(source) {
		copy(res, source)
	} else {
		copy(res[size-len(source):], source)
	}
	return res
}
