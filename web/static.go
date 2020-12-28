package web

import (
	"github.com/QuestScreen/QuestScreen/shared"
	"github.com/QuestScreen/api/web/modules"
)

// MappedModule is a module known to the client.
type MappedModule struct {
	modules.Constructor
	PluginIndex int
	Name, ID    string
}

// StaticData is loaded at app start and is constant everafter.
var StaticData struct {
	Fonts            []string
	Textures         []string
	NumPluginSystems int
	Plugins          []shared.Plugin
	FontDir          string
	Messages         []shared.Message
	AppVersion       string
	Modules          []MappedModule
}
