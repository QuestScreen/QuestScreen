package shared

// Static describes static data that will never change during the lifetime of
// the server/client.
type Static struct {
	Fonts            []string    `json:"fonts"`
	Textures         []string    `json:"textures"`
	Modules          []Module    `json:"modules"`
	NumPluginSystems int         `json:"numPluginSystems"`
	Plugins          interface{} `json:"plugins"`
	FontDir          string      `json:"fontDir"`
	Messages         []Message   `json:"messages"`
	AppVersion       string      `json:"appVersion"`
}
