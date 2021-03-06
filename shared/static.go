package shared

import "github.com/QuestScreen/api/resources"

// Static describes static data that will never change during the lifetime of
// the server/client.
type Static struct {
	Fonts            []string             `json:"fonts"`
	Textures         []resources.Resource `json:"textures"`
	NumPluginSystems int                  `json:"numPluginSystems"`
	Plugins          []Plugin             `json:"plugins"`
	FontDir          string               `json:"fontDir"`
	Messages         []Message            `json:"messages"`
	AppVersion       string               `json:"appVersion"`
	Modules          []Module             `json:"modules"`
}
