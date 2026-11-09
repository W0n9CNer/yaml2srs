package tools

import "cmp"

func UnorderedDeduplication[T cmp.Ordered](slice []T) []T {
	m := make(map[T]struct{})

	for _, v := range slice {
		m[v] = struct{}{}
	}

	set := make([]T, len(m))
	count := len(m)
	for k := range m {
		count--
		set[count] = k
	}
	return set
}
