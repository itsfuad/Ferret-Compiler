package stack

import (
	"testing"
)

func TestStackPushPop(t *testing.T) {
	stack := New[int]()
	if stack.Count() != 0 {
		t.Errorf("expected count 0, got %d", stack.Count())
	}

	stack.Push(10)
	if stack.Count() != 1 {
		t.Errorf("expected count 1, got %d", stack.Count())
	}
	stack.Push(20)
	if stack.Count() != 2 {
		t.Errorf("expected count 2, got %d", stack.Count())
	}

	val := stack.Pop()
	if val != 20 {
		t.Errorf("expected 20, got %v", val)
	}
	if stack.Count() != 1 {
		t.Errorf("expected count 1 after pop, got %d", stack.Count())
	}

	val = stack.Pop()
	if val != 10 {
		t.Errorf("expected 10, got %v", val)
	}
	if stack.Count() != 0 {
		t.Errorf("expected count 0 after pop, got %d", stack.Count())
	}

	_ = stack.Pop()
}

func TestStackPushPopString(t *testing.T) {
	stack := New[string]()
	stack.Push("foo")
	stack.Push("bar")

	val := stack.Pop()
	if val != "bar" {
		t.Errorf("expected 'bar', got %v", val)
	}
	val = stack.Pop()
	if val != "foo" {
		t.Errorf("expected 'foo', got %v", val)
	}
	_ = stack.Pop()
}

func TestStackEmptyPop(t *testing.T) {
	stack := New[float64]()
	val := stack.Pop()
	if val != 0 {
		t.Errorf("expected ok=false, got ok=true")
	}
	if val != 0 {
		t.Errorf("expected zero value, got %v", val)
	}
}

func TestStackPeek(t *testing.T) {
	stack := New[int]()
	stack.Push(10)
	stack.Push(20)

	val := stack.Peek()
	if val != 20 {
		t.Errorf("expected 20, got %v", val)
	}
	stack.Pop()
	val = stack.Peek()
	if val != 10 {
		t.Errorf("expected 10, got %v", val)
	}
	stack.Pop()
	val = stack.Peek()
	if val != 0 {
		t.Errorf("expected zero value, got %v", val)
	}
}

// benchmark tests
func BenchmarkStackPush(b *testing.B) {
	stack := New[int]()
	for i := 0; b.Loop(); i++ {
		stack.Push(i)
	}
}

func BenchmarkStackPop(b *testing.B) {
	stack := New[int]()
	for i := 0; b.Loop(); i++ {
		stack.Push(i)
	}

	for b.Loop() {
		stack.Pop()
	}
}

func BenchmarkStackPushPop(b *testing.B) {
	stack := New[int]()
	for i := 0; b.Loop(); i++ {
		stack.Push(i)
		stack.Pop()
	}
}
