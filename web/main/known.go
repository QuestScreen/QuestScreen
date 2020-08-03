package main

import (
	"fmt"

	"github.com/QuestScreen/QuestScreen/shared"
	"github.com/QuestScreen/QuestScreen/web"
	api "github.com/QuestScreen/api/web"
)

// pluginLoader implements web.PluginRegistrator
type pluginLoader struct {
	id  string
	tmp shared.Static
}

func (pl *pluginLoader) RegisterConfigItem(id string, constructor api.ConfigItemConstructor) error {
	path := pl.id + "/" + id
	found := false
	for i := range pl.tmp.ConfigItems {
		if path == pl.tmp.ConfigItems[i] {
			found = true
			if web.StaticData.ConfigItems[i] != nil {
				return fmt.Errorf("[plugin %s] duplicate config item ID during registration: %s", pl.id, id)
			}
			web.StaticData.ConfigItems[i] = constructor
			break
		}
	}
	if !found {
		return fmt.Errorf("[plugin %s] server doesn't know config item %s", pl.id, id)
	}
	return nil
}

func (pl *pluginLoader) RegisterModule(id string, constructor api.ModuleConstructor) error {
	path := pl.id + "/" + id
	found := false
	for i := range pl.tmp.Modules {
		serverItem := &pl.tmp.Modules[i]
		if path == serverItem.Path {
			found = true
			appItem := &web.StaticData.Modules[i]
			if appItem.Constructor != nil {
				return fmt.Errorf("[plugin %s] duplicate module ID during registration: %s", pl.id, id)
			}
			appItem.Constructor = constructor
			appItem.Settings = make([]web.ModuleConfigItem, len(serverItem.Config))
			for i := range serverItem.Config {
				appItem.Settings[i] = web.ModuleConfigItem{Name: serverItem.Config[i].Name,
					Index: serverItem.Config[i].TypeIndex}
			}
			break
		}
	}
	if !found {
		return fmt.Errorf("[plugin %s] server doesn't know module %s", pl.id, id)
	}

	return nil
}
