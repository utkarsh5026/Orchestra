package store

// Store defines the interface for task storage implementations.
// It provides basic CRUD operations for storing and retrieving tasks.
type Store interface {
	// Put stores a value with the given key.
	// Returns an error if the operation fails.
	Put(key string, value any) error

	// Get retrieves the value associated with the given key.
	// Returns the value and nil error if found, or nil and error if not found.
	Get(key string) (any, error)

	// List returns all values in the store.
	// Returns a slice of values and nil error on success, or nil and error on failure.
	List() (any, error)

	// Count returns the total number of items in the store.
	// Returns the count and nil error on success, or 0 and error on failure.
	Count() (int, error)
}
