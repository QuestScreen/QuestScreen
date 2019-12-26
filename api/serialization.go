package api

// DataLayout describes how input / output data is structured.
type DataLayout int

const (
	// Persisted is the data layout used for persisting data on the file system.
	Persisted DataLayout = iota
	// Web is the data layout used for sending / receiving data as JSON via the
	// web API.
	Web
)

// SerializableItem describes an item that can be serialized.
// Serialization happens both when the item is sent to a client via the Web API,
// and when the item is persisted to the file system.
type SerializableItem interface {
	// SerializableView returns a data structure that represents the item's
	// content in the given layout. The returned data will be incorporated into
	// a larger structure that will be serialized as YAML or JSON.
	//
	// If you want to manually serialize the data, you may return a
	// - yaml.Node for layout == Persisted
	// - json.RawMessage for layout == Web
	// However, manual serialization should typically not be necessary.
	// The usual implementation should be to just return the item itself.
	SerializableView(env Environment, layout DataLayout) interface{}
}
