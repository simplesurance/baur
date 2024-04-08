package set

type Set[T comparable] map[T]struct{}

func From[T comparable](slice []T) Set[T] {
	set := make(Set[T], len(slice))

	for _, v := range slice {
		set[v] = struct{}{}
	}
	return set
}

func (s Set[T]) Slice() []T {
	res := make([]T, 0, len(s))

	for k := range s {
		res = append(res, k)
	}

	return res
}

func (s Set[T]) Add(val T) {
	s[val] = struct{}{}
}

func (s Set[T]) Contains(v T) bool {
	_, exists := s[v]
	return exists
}
