// Package collection provides small, domain-independent helpers for typed Go
// collections. It intentionally contains no persistence or business rules.
package collection

// Get returns the value and whether key exists.
func Get[K comparable, V any](items map[K]V, key K) (V, bool) {
	value, ok := items[key]
	return value, ok
}

// MapValues returns map values that satisfy include. A nil include keeps all.
func MapValues[K comparable, V any](items map[K]V, include func(V) bool) []V {
	result := make([]V, 0, len(items))
	for _, item := range items {
		if include == nil || include(item) {
			result = append(result, item)
		}
	}
	return result
}

// Filter returns slice elements that satisfy include.
func Filter[T any](items []T, include func(T) bool) []T {
	result := make([]T, 0, len(items))
	for _, item := range items {
		if include(item) {
			result = append(result, item)
		}
	}
	return result
}
