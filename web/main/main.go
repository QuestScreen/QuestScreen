package main

import (
	"github.com/QuestScreen/QuestScreen/web"
	"github.com/QuestScreen/QuestScreen/web/info"
	"github.com/QuestScreen/QuestScreen/web/server"
	api "github.com/QuestScreen/api/web/server"
)

func main() {
	var loader pluginLoader
	if err := server.Fetch(api.Get, "/static", nil, &loader.tmp); err != nil {
		panic(err)
	}
	web.StaticData.Fonts = loader.tmp.Fonts
	web.StaticData.Textures = loader.tmp.Textures
	web.StaticData.NumPluginSystems = loader.tmp.NumPluginSystems
	web.StaticData.Plugins = loader.tmp.Plugins
	web.StaticData.FontDir = loader.tmp.FontDir
	web.StaticData.Messages = loader.tmp.Messages
	web.StaticData.AppVersion = loader.tmp.AppVersion
	web.StaticData.Modules = make([]web.MappedModule, len(loader.tmp.Modules))
	loadPlugins(&loader)

	for i, m := range loader.tmp.Modules {
		if web.StaticData.Modules[i].Constructor == nil {
			panic("server module " + m.Path + " unknown")
		}
	}
	Page.Set(info.NewPage(web.StaticData.AppVersion))
}
