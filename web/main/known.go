package main

import (
	"fmt"

	"github.com/QuestScreen/QuestScreen/shared"
	"github.com/QuestScreen/QuestScreen/web"
	"github.com/QuestScreen/api/web/modules"
)

// pluginLoader implements web.PluginRegistrator
type pluginLoader struct {
	id    string
	index int
	tmp   shared.Static
}

func (pl *pluginLoader) RegisterModule(id string, constructor modules.Constructor) error {
	path := pl.id + "/" + id
	found := false
	for i := range pl.tmp.Modules {
		serverItem := &pl.tmp.Modules[i]
		if path == serverItem.Path {
			found = true
			module := &web.StaticData.Modules[i]
			if module.Constructor != nil {
				return fmt.Errorf("[plugin %s] duplicate module ID during registration: %s", pl.id, id)
			}
			module.Constructor = constructor
			module.Name = serverItem.Name
			module.ID = id
			module.PluginIndex = pl.index
			break
		}
	}
	if !found {
		return fmt.Errorf("[plugin %s] server doesn't know module %s", pl.id, id)
	}

	return nil
}
