package data

// ModuleState contains the state of a module. It is persisted to and loaded
// from a group's state.yaml.
//
// A type implementing this should typically hold a reference to its module.
type ModuleState interface {
	LoadFrom(yamlSubtree interface{}, store *Store) error

	ToYAML(store *Store) interface{}
	ToJSON() interface{}

	// defines the list of actions that should be registered for this module.
	Actions() []string
	// called in the server thread. updates the state. if no error is returned,
	// the OpenGL thread will call InitTransition on the module.
	// returns the content to return on success (nil if nothing should be returned).
	// if not nil, the content must be valid JSON.
	HandleAction(index int, payload []byte, store *Store) ([]byte, error)
}
