package main

import (
	"github.com/QuestScreen/QuestScreen/web"
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
	for _, msg := range web.StaticData.Messages {
		if msg.ModuleIndex == -1 {
			Header.Disabled.Set(true)
			break
		}
	}
	web.StaticData.AppVersion = loader.tmp.AppVersion
	web.StaticData.Modules = make([]web.MappedModule, len(loader.tmp.Modules))
	if err := registerPlugins(&loader); err != nil {
		panic("while loading modules: " + err.Error())
	}

	for i, m := range loader.tmp.Modules {
		if web.StaticData.Modules[i].Constructor == nil {
			panic("server module " + m.Path + " unknown")
		}
	}

	app := &App{}
	app.Init()
	web.Page = app
	TitleContent.Controller = app
	Header.Controller = app
	app.ShowInfo()
}
