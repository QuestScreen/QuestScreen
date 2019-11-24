package api

// Plugin describes a module provider.
type Plugin interface {
	// Name returns the name of the plugin.
	Name() string
	// Modules returns the list of modules provided by this plugin.
	Modules() []Module
	// AdditionalJS returns JavaScript source needed to support the
	// plugin's modules on the client side.
	AdditionalJS() []byte
	// AdditionalHTML returns HTML templates needed to support the
	// plugin's modules on the client side.
	AdditionalHTML() []byte
}
