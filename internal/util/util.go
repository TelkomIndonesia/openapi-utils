package util

func MapFirstEntry[K comparable, V any](m map[K]V) (e struct {
	Key   K
	Value V
}) {
	for k, v := range m {
		return struct {
			Key   K
			Value V
		}{
			Key:   k,
			Value: v,
		}
	}
	return
}

func getFromMap[K comparable, V any](m map[K]V, key K) (v V) {
	if m == nil {
		return
	}
	return m[key]
}
