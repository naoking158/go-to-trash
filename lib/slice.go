package lib

// UniqByKey is copied from samber/lo
// ref: https://github.com/samber/lo/blob/bc6d6e8b8f5515c7c58a376423923c08b6f723cd/slice.go#L159
func UniqByKey[T any, U comparable, Slice ~[]T](collection Slice, keyOf func(item T) U) Slice {
	result := make([]T, 0, len(collection))
	seen := make(map[U]struct{}, len(collection))

	for i := range collection {
		key := keyOf(collection[i])
		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}
		result = append(result, collection[i])
	}

	return result
}
