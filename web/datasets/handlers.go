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
	o.edited.Set(true)
}

func (o *editableText) ResetTo(value string) {
	o.Value.Set(value)
	o.edited.Set(false)
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

func (o *system) init(data *shared.System) {
	o.reset()
}

func (o *system) reset() {
	o.name.ResetTo(o.data.Name)
}

func (o *system) commit() {
	go func() {
		if err := server.Fetch(api.Put, "data/systems/"+o.data.ID,
			shared.SystemModificationRequest{Name: o.name.Value.Get()},
			&web.Data.Systems); err != nil {
			panic(err)
		}
		site.Refresh("s-" + o.data.ID)
	}()
}

func (o *hero) init(data *shared.Hero) {
	o.Name.Value.Set(data.Name)
	o.Description.Value.Set(data.Description)
}

func (o *group) init(data *shared.Group) {
	for _, s := range web.Data.Systems {
		o.Systems.AddItem(s.Name, false)
	}
	for _, s := range data.Scenes {
		o.Scenes.Append(newListItem(s.Name, true, o.Scenes.Len()))
	}
	for i := range data.Heroes {
		o.Heroes.Append(newHero(&data.Heroes[i]))
	}
	o.reset()
}

func (o *group) reset() {
	o.name.ResetTo(o.data.Name)
	o.Systems.SetItem(o.data.SystemIndex, true)
	o.systemEdited.Set(false)
}

func (o *group) commit() {
	go func() {
		if err := server.Fetch(api.Put, "data/groups/"+o.data.ID, o.name.Value.Get(),
			&web.Data.Groups); err != nil {
			panic(err)
		}
		site.Refresh("g-" + o.data.ID)
	}()
}

func (o *group) ItemClicked(index int) bool {
	o.systemEdited.Set(true)
	return true
}

func (o *scene) init(groupID string, data *shared.Scene) {
	o.reset()
}

func (o *scene) reset() {
	o.name.ResetTo(o.data.Name)
}

func (o *scene) commit() {
	panic("not implemented")
}
