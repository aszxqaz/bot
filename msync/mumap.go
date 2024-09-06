package msync

import "sync"

type MuMap[K comparable, T any] struct {
	mu   sync.Mutex
	data map[K]T
}

func NewMuMap[K comparable, T any]() *MuMap[K, T] {
	return &MuMap[K, T]{data: make(map[K]T)}
}

func (mm *MuMap[K, T]) Get(key K) (T, bool) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	value, ok := mm.data[key]
	return value, ok
}

func (mm *MuMap[K, T]) Set(key K, value T) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.data[key] = value
}

func (mm *MuMap[K, T]) Delete(key K) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	delete(mm.data, key)
}

func (mm *MuMap[K, T]) Len() int {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	return len(mm.data)
}

func (mm *MuMap[K, T]) Range(f func(key K, value T) bool) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	for key, value := range mm.data {
		if !f(key, value) {
			break
		}
	}
}

func (mm *MuMap[K, T]) Keys() []K {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	keys := make([]K, 0, len(mm.data))
	for key := range mm.data {
		keys = append(keys, key)
	}
	return keys
}
