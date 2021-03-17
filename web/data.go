package web

import (
	"github.com/QuestScreen/QuestScreen/shared"
	"github.com/QuestScreen/api/resources"
	"github.com/QuestScreen/api/web/modules"
)

// MappedModule is a module known to the client.
type MappedModule struct {
	modules.Constructor
	ConfigItems []modules.ConfigItem
	PluginIndex int
	Name, ID    string
}

func (mm MappedModule) StateBasePath() string {
	return "/state/" + StaticData.Plugins[mm.PluginIndex].ID + "/" +
		mm.ID + "/"
}

// StaticData is loaded when booting and is constant everafter.
var StaticData struct {
	Fonts            []string
	Textures         []resources.Resource
	NumPluginSystems int
	Plugins          []shared.Plugin
	FontDir          string
	Messages         []shared.Message
	AppVersion       string
	Modules          []MappedModule
}

// Data is loaded when booting and updated according to user actions.
var Data shared.Data
