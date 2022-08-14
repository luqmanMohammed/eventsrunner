package utils

// GetPointerFor returns a pointer of the given value.
func GetPointerFor[T any](v T) *T {
	return &v
}
