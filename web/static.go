package web

import (
	"github.com/QuestScreen/QuestScreen/shared"
	"github.com/QuestScreen/api/web"
)

// ModuleConfigItem is part of a module's configuration.
type ModuleConfigItem struct {
	Name  string
	Index shared.ConfigItemIndex
}

// MappedModule is a module known to the client.
type MappedModule struct {
	Constructor web.ModuleConstructor
	Settings    []ModuleConfigItem
}

// StaticData is loaded at app start and constant everafter.
var StaticData struct {
	Fonts            []string
	Textures         []string
	NumPluginSystems int
	Plugins          []shared.Plugin
	FontDir          string
	Messages         []shared.Message
	AppVersion       string
	Modules          []MappedModule
	ConfigItems      []web.ConfigItemConstructor
}
