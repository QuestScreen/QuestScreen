package datasets

import (
	"github.com/QuestScreen/QuestScreen/shared"
	"github.com/QuestScreen/QuestScreen/web"
	"github.com/QuestScreen/QuestScreen/web/controls"
	"github.com/QuestScreen/QuestScreen/web/server"
	"github.com/QuestScreen/QuestScreen/web/site"
	api "github.com/QuestScreen/api/web/server"
)

func (o *EditableText) setEdited() {
	o.Edited.Set(true)
}

func (o *ListItem) clicked(index int) {
}

type systemItemsController struct {
	*Base
}

func (c *systemItemsController) delete(index int) {
	go func() {
		data := web.Page.Data()
		system := data.Systems[index]
		if ok := site.Popup.Confirm("Really delete system " + system.Name + "?"); ok {
			if err := server.Fetch(
				api.Delete, "data/systems/"+system.ID, nil, &data.Systems); err != nil {
				panic(err)
			}
			c.SystemList.RemoveAll()
			c.regenSystems()
			// TODO: regen menu
		}
	}()
}

type groupItemsController struct {
	*Base
}

func (c *groupItemsController) delete(index int) {
	go func() {
		data := web.Page.Data()
		group := data.Groups[index]
		if ok := site.Popup.Confirm("Really delete group " + group.Name + "?"); ok {
			if err := server.Fetch(api.Delete, "data/groups/"+group.ID, nil, &data.Groups); err != nil {
				panic(err)
			}
			c.GroupList.RemoveAll()
			c.regenGroups()
			// TODO: regen menu
		}
	}()
}

func (o *Base) regenSystems() {
	for index, system := range web.Page.Data().Systems {
		item := NewListItem(system.Name,
			index >= web.StaticData.NumPluginSystems, index)
		o.SystemList.Append(item)
	}
}

func (o *Base) regenGroups() {
	for index, group := range web.Page.Data().Groups {
		o.GroupList.Append(NewListItem(group.Name, true, index))
	}
}

func (o *Base) init() {
	o.sc.Base = o
	o.gc.Base = o
	o.SystemList.DefaultController = &o.sc
	o.GroupList.DefaultController = &o.gc
	o.regenGroups()
	o.regenSystems()
}

func (o *Base) onInclude() {
	web.Page.SetTitle("Dataset Base", "", web.BackButtonBack)
}

func (o *Base) addSystem() {
	go func() {
		name := site.Popup.TextInput("Create system", "Name:")
		if name != nil {
			if err := server.Fetch(api.Post, "data/systems", *name,
				&web.Page.Data().Systems); err != nil {
				panic(err)
			}
			// TODO: regen menu
			o.SystemList.RemoveAll()
			o.regenSystems()
		}
	}()
}

func (o *Base) addGroup() {
	go func() {
		pluginIndex, templateIndex, name :=
			site.Popup.TemplateSelect(controls.GroupTemplate)
		if pluginIndex != -1 {
			if err := server.Fetch(api.Post, "data/groups", shared.GroupCreationRequest{
				name, pluginIndex, templateIndex}, &web.Page.Data().Groups); err != nil {
				panic(err)
			}
			// TODO: regen menu
			o.GroupList.RemoveAll()
			o.regenGroups()
		}
	}()
}
