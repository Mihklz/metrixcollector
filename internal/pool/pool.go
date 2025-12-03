package pool

import "sync"

// Resetter описывает типы, которые умеют сбрасывать своё состояние.
type Resetter interface {
	Reset()
}

// Pool — типобезопасная обёртка над sync.Pool для объектов, поддерживающих метод Reset.
//
// T должен реализовывать интерфейс Resetter (то есть иметь метод Reset()).
type Pool[T Resetter] struct {
	p *sync.Pool
}

// New создаёт новый пул для объектов типа T.
//
// Аргумент newFn должен возвращать новый экземпляр типа T, который
// будет использоваться при исчерпании объектов в пуле.
func New[T Resetter](newFn func() T) *Pool[T] {
	if newFn == nil {
		panic("pool: newFn must not be nil")
	}

	return &Pool[T]{
		p: &sync.Pool{
			New: func() any {
				return newFn()
			},
		},
	}
}

// Get возвращает объект из пула.
// Если пул пуст, используется функция newFn, переданная в New.
func (p *Pool[T]) Get() T {
	v := p.p.Get()
	if v == nil {
		var zero T
		return zero
	}
	return v.(T)
}

// Put возвращает объект в пул.
// Перед возвратом всегда вызывается метод Reset(), чтобы очистить состояние.
func (p *Pool[T]) Put(v T) {
	v.Reset()
	p.p.Put(v)
}
