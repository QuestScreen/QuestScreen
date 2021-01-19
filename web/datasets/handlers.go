package datasets

import (
	"github.com/QuestScreen/QuestScreen/shared"
	"github.com/QuestScreen/QuestScreen/web"
	"github.com/QuestScreen/QuestScreen/web/controls"
	"github.com/QuestScreen/QuestScreen/web/server"
	"github.com/QuestScreen/QuestScreen/web/site"
	api "github.com/QuestScreen/api/web/server"
)

func (o *editableText) setEdited() {
	o.Edited.Set(true)
}

func (o *listItem) clicked(index int) {
}

type systemItemsController struct {
	*base
}

func (c *systemItemsController) delete(index int) {
	go func() {
		system := web.Data.Systems[index]
		if ok := site.Popup.Confirm("Really delete system " + system.Name + "?"); ok {
			if err := server.Fetch(
				api.Delete, "data/systems/"+system.ID, nil, &web.Data.Systems); err != nil {
				panic(err)
			}
			site.Refresh("")
		}
	}()
}

type groupItemsController struct {
	*base
}

func (c *groupItemsController) delete(index int) {
	go func() {
		group := web.Data.Groups[index]
		if ok := site.Popup.Confirm("Really delete group " + group.Name + "?"); ok {
			if err := server.Fetch(api.Delete, "data/groups/"+group.ID, nil, &web.Data.Groups); err != nil {
				panic(err)
			}
			site.Refresh("")
		}
	}()
}

func (o *base) regenSystems() {
	for index, system := range web.Data.Systems {
		item := newListItem(system.Name,
			index >= web.StaticData.NumPluginSystems, index)
		o.SystemList.Append(item)
	}
}

func (o *base) regenGroups() {
	for index, group := range web.Data.Groups {
		o.GroupList.Append(newListItem(group.Name, true, index))
	}
}

func (o *base) init() {
	o.sc.base = o
	o.gc.base = o
	o.SystemList.DefaultController = &o.sc
	o.GroupList.DefaultController = &o.gc
	o.regenGroups()
	o.regenSystems()
}

func (o *base) addSystem() {
	go func() {
		name := site.Popup.TextInput("Create system", "Name:")
		if name != nil {
			if err := server.Fetch(api.Post, "data/systems", *name,
				&web.Data.Systems); err != nil {
				panic(err)
			}
			site.Refresh("")
		}
	}()
}

func (o *base) addGroup() {
	go func() {
		pluginIndex, templateIndex, name :=
			site.Popup.TemplateSelect(controls.GroupTemplate)
		if pluginIndex != -1 {
			if err := server.Fetch(api.Post, "data/groups", shared.GroupCreationRequest{
				name, pluginIndex, templateIndex}, &web.Data.Groups); err != nil {
				panic(err)
			}
			site.Refresh("")
		}
	}()
}
