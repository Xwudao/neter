package stack

import (
	"fmt"
)

type Stack[T any] struct {
	items []T
}

func New[T any]() *Stack[T] {
	return new(Stack[T])
}

func (s *Stack[T]) String() string {
	return fmt.Sprint(s.items)
}

func (s *Stack[T]) IsEmpty() bool {
	return len(s.items) == 0
}

func (s *Stack[T]) Size() int {
	return len(s.items)
}

func (s *Stack[T]) Top() T {
	return s.items[len(s.items)-1]
}

func (s *Stack[T]) Push(item T) {
	s.items = append(s.items, item)
}

func (s *Stack[T]) Pop() T {
	item := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return item
}
