package pool

import "testing"

type testObj struct {
	value int
}

func (t *testObj) Reset() {
	t.value = 0
}

func TestPool_GetPut(t *testing.T) {
	p := New(func() *testObj {
		return &testObj{value: 42}
	})

	obj := p.Get()
	if obj == nil {
		t.Fatalf("expected non-nil object from pool")
	}
	if obj.value != 42 {
		t.Fatalf("unexpected initial value: %d", obj.value)
	}

	// Меняем состояние и возвращаем объект в пул.
	obj.value = 100
	p.Put(obj)

	// Забираем объект снова и проверяем, что Reset был вызван.
	obj2 := p.Get()
	if obj2 == nil {
		t.Fatalf("expected non-nil object from pool on second Get")
	}
	if obj2.value != 0 {
		t.Fatalf("expected value to be reset to 0, got %d", obj2.value)
	}
}
