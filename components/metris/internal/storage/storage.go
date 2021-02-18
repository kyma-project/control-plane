package storage

type Storage interface {
	// Put sets the value for a key.
	Put(key string, obj interface{})

	// Get returns the value stored in the map for a key, or nil if no value is present.
	// The ok result indicates whether value was found in the map.
	Get(key string) (obj interface{}, exists bool)

	// Update sets the value for a key.
	Update(key string, obj interface{})

	// Delete deletes the value for a key.
	Delete(key string)

	// List returns a list of all the objects.
	List() []interface{}

	// ListKeys returns a list of all the keys associated with objects.
	ListKeys() []string
}
