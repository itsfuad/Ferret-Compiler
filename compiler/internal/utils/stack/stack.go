package stack

type Stack[T any] struct {
	values []T
}

func New[T any]() *Stack[T] {
	return &Stack[T]{values: make([]T, 0)}
}

func (s *Stack[T]) Push(value T) {
	s.values = append(s.values, value)
}

func (s *Stack[T]) Pop() T {
	if len(s.values) == 0 {
		var zero T
		return zero
	}
	value := s.values[len(s.values)-1]
	s.values = s.values[:len(s.values)-1]
	return value
}

func (s *Stack[T]) Peek() T {
	if len(s.values) == 0 {
		var zero T
		return zero
	}
	return s.values[len(s.values)-1]
}

func (s *Stack[T]) IsEmpty() bool {
	return len(s.values) == 0
}

func (s *Stack[T]) Count() int {
	return len(s.values)
}
