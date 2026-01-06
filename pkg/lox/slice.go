package lox

func MapErr[T any, R comparable](collection []T, iteratee func(item T) (R, error)) ([]R, error) {
	var err error

	result := make([]R, len(collection))

	for i, item := range collection {
		result[i], err = iteratee(item)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func Map[T, R any](collection []T, iteratee func(item T) R) []R {
	result := make([]R, len(collection))

	for i, item := range collection {
		result[i] = iteratee(item)
	}

	return result
}

func ReverseMap[T, T1 any, R comparable](collection map[R]T, iteratee func(key R, value T) T1) []T1 {
	result := make([]T1, 0, len(collection))

	for k, v := range collection {
		result = append(result, iteratee(k, v))
	}

	return result
}

func FilterAssociate[T any, R comparable](collection []T, callback func(item T) (R, bool)) map[R]T {
	result := make(map[R]T, len(collection))

	for _, item := range collection {
		if r, ok := callback(item); ok {
			result[r] = item
		}
	}

	return result
}
