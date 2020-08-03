// Package shared defines data structures that are used both by the server and
// by the web client.
// These are typically data types transferred as JSON.
package shared

// ModuleConfig is a list of configuration items.
type ModuleConfig []interface{}

// System describes a pen & paper roleplaying system.
type System struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// Hero describes a hero in a group.
type Hero struct {
	Name        string `json:"name"`
	ID          string `json:"id"`
	Description string `json:"description"`
}

// Scene describes a scene of a group.
type Scene struct {
	Name    string `json:"name"`
	ID      string `json:"id"`
	Modules []bool `json:"modules"`
}

// Group describes a dataset for a pen & paper roleplaying group.
type Group struct {
	Name        string  `json:"name"`
	ID          string  `json:"id"`
	SystemIndex int     `json:"systemIndex"`
	Heroes      []Hero  `json:"heroes"`
	Scenes      []Scene `json:"scenes"`
}

// ModuleSetting describes a setting of a module.
type ModuleSetting struct {
	Name      string          `json:"name"`
	TypeIndex ConfigItemIndex `json:"typeIndex"`
}

// Module describes a loaded module.
type Module struct {
	Name string `json:"name"`
	// <plugin-id>/<module-id>
	Path   string          `json:"path"`
	Config []ModuleSetting `json:"config"`
}

// Plugin describes a loaded plugin.
type Plugin struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}
