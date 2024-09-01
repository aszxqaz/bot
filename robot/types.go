package robot

import "sync"

type MuMap[T any] struct {
	mu   sync.Mutex
	data map[string]T
}

func NewMuMap[T any]() *MuMap[T] {
	return &MuMap[T]{data: make(map[string]T)}
}

func (mm *MuMap[T]) Get(key string) (T, bool) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	value, ok := mm.data[key]
	return value, ok
}

func (mm *MuMap[T]) Set(key string, value T) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.data[key] = value
}

func (mm *MuMap[T]) Delete(key string) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	delete(mm.data, key)
}

func (mm *MuMap[T]) Len() int {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	return len(mm.data)
}

func (mm *MuMap[T]) Range(f func(key string, value T) bool) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	for key, value := range mm.data {
		if !f(key, value) {
			break
		}
	}
}
