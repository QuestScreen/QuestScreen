package main

import (
	"github.com/QuestScreen/QuestScreen/web"
	"github.com/QuestScreen/QuestScreen/web/comms"
	"github.com/QuestScreen/QuestScreen/web/config"
	"github.com/QuestScreen/QuestScreen/web/datasets"
	"github.com/QuestScreen/QuestScreen/web/home"
	"github.com/QuestScreen/QuestScreen/web/site"
	api "github.com/QuestScreen/api/web"
)

func main() {
	var loader pluginLoader
	if err := comms.Fetch(api.Get, "/static", nil, &loader.tmp); err != nil {
		panic(err)
	}
	web.StaticData.Fonts = loader.tmp.Fonts
	web.StaticData.Textures = loader.tmp.Textures
	web.StaticData.NumPluginSystems = loader.tmp.NumPluginSystems
	web.StaticData.Plugins = loader.tmp.Plugins
	web.StaticData.FontDir = loader.tmp.FontDir
	web.StaticData.Messages = loader.tmp.Messages
	headerDisabled := false
	for _, msg := range web.StaticData.Messages {
		if msg.ModuleIndex == -1 {
			headerDisabled = true
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

	if err := comms.Fetch(api.Get, "/data", nil, &web.Data); err != nil {
		panic(err)
	}

	datasets.Register()
	home.Register()
	config.Register()
	site.Boot(headerDisabled)
}
