package shared

// ModuleIndex identifies a module internally.
type ModuleIndex int

// FirstModule is the index of the first module
const FirstModule ModuleIndex = 0

// ConfigItemIndex identifies a config item internally.
type ConfigItemIndex int

// FirstConfigItem is the index of the first config item.
const FirstConfigItem ConfigItemIndex = 0

// Message is a warning or an error that should be displayed on the starting
// screen of the client.
type Message struct {
	// true if this is an error, false if it's just a warning.
	IsError bool `json:"isError"`
	// Index of the module the message is issued from, -1 if none
	ModuleIndex ModuleIndex `json:"moduleIndex"`
	// text to display
	Text string `json:"text"`
}
