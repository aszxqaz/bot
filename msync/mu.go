package msync

import "sync"

type Mu[T any] struct {
	mu   sync.RWMutex
	data T
}

func NewMu[T any](value T) *Mu[T] {
	return &Mu[T]{data: value}
}

func (m *Mu[T]) Get() T {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.data
}

func (m *Mu[T]) Set(value T) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = value
}

func (m *Mu[T]) Update(updateFn func(value T) T) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = updateFn(m.data)
}
