package util

// PtrTo returns pointer to given value
func PtrTo[T any](v T) *T {
	return &v
}

// UnwrapOrZero returns value from pointer or zero value if pointer is nil
func UnwrapOrZero[T any](ptr *T) T {
	if ptr == nil {
		return *new(T)
	}

	return *ptr
}

// UnwrapOrDefault returns value from pointer or provided default value if pointer is nil
func UnwrapOrDefault[T any](ptr *T, def T) T {
	if ptr == nil {
		return def
	}

	return *ptr
}

// OkOrDefault returns pointer or provided default pointer if pointer is nil
func OkOrDefault[T any](ptr *T, def *T) *T {
	if ptr == nil {
		return def
	}
	return ptr
}
