package utils

import (
	"log"

	"github.com/utkarsh5026/Orchestra/store"
)

// UpdateStore attempts to store a value in a generic key-value store and logs any errors
//
// Parameters:
//   - store: The key-value store to update
//   - key: The key to store the value under
//   - data: The value to store
func UpdateStore[K comparable, V any](store store.Store[K, V], key K, data V) {
	if err := store.Put(key, data); err != nil {
		log.Printf("Error updating store: for key %v: %v\n", key, err)
	}
}
