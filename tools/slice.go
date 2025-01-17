package tools

import "github.com/sagernet/sing/common/x/constraints"

func UnorderedDeduplication[T constraints.Ordered](slice []T) []T {
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
