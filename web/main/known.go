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
			break
		}
	}
	if !found {
		return fmt.Errorf("[plugin %s] server doesn't know module %s", pl.id, id)
	}

	return nil
}
