package store

// Store defines the interface for task storage implementations.
// It provides basic CRUD operations for storing and retrieving tasks.
type Store[k comparable, V any] interface {
	// Put stores a value with the given key.
	// Returns an error if the operation fails.
	Put(key k, value V) error

	// Get retrieves the value associated with the given key.
	// Returns the value and nil error if found, or nil and error if not found.
	Get(key k) (V, error)

	// List returns all values in the store.
	// Returns a slice of values and nil error on success, or nil and error on failure.
	List() ([]V, error)

	// Count returns the total number of items in the store.
	// Returns the count and nil error on success, or 0 and error on failure.
	Count() (int, error)
}

type Type uint

const (
	InMemoryStoreType Type = iota
)

func NewStore[k comparable, V any](t Type) Store[k, V] {
	return NewInMemoryTaskStore[k, V]()
}
