package store

import (
	"fmt"
)

type InMemoryTaskStore[K comparable, V any] struct {
	Db map[K]V
}

func NewInMemoryTaskStore[K comparable, V any]() *InMemoryTaskStore[K, V] {
	return &InMemoryTaskStore[K, V]{
		Db: make(map[K]V),
	}
}

func (i *InMemoryTaskStore[K, V]) Put(key K, value V) error {
	i.Db[key] = value
	return nil
}

func (i *InMemoryTaskStore[K, V]) Get(key K) (V, error) {
	value, ok := i.Db[key]
	if !ok {
		var zero V
		return zero, fmt.Errorf("key %v does not exist", key)
	}
	return value, nil
}

func (i *InMemoryTaskStore[K, V]) List() ([]V, error) {
	items := make([]V, 0, len(i.Db))
	for _, v := range i.Db {
		items = append(items, v)
	}
	return items, nil
}

func (i *InMemoryTaskStore[K, V]) Count() (int, error) {
	return len(i.Db), nil
}
