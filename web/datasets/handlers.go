package datasets

import (
	"syscall/js"

	"github.com/QuestScreen/QuestScreen/shared"
	"github.com/QuestScreen/QuestScreen/web"
	"github.com/QuestScreen/QuestScreen/web/comms"
	"github.com/QuestScreen/QuestScreen/web/site"
	api "github.com/QuestScreen/api/web"
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
	system := web.Data.Systems[index]
	site.Popup.Confirm("Really delete system "+system.Name+"?", func() {
		if err := comms.Fetch(
			api.Delete, "data/systems/"+system.ID, nil, &web.Data.Systems); err != nil {
			panic(err)
		}
		site.Refresh("")
	})
}

type groupItemsController struct {
	*base
}

func (c *groupItemsController) delete(index int) {
	group := web.Data.Groups[index]
	site.Popup.Confirm("Really delete group "+group.Name+"?", func() {
		if err := comms.Fetch(api.Delete, "data/groups/"+group.ID, nil, &web.Data.Groups); err != nil {
			panic(err)
		}
		site.Refresh("")
	})
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

func newBase() *base {
	ret := new(base)
	ret.init()
	return ret
}

func (o *base) init() {
	o.askewInit()
	o.sc.base = o
	o.gc.base = o
	o.SystemList.DefaultController = &o.sc
	o.GroupList.DefaultController = &o.gc
	o.regenGroups()
	o.regenSystems()
}

func (o *base) addSystem() {
	js.Global().Get("console").Call("log", "adding system")
	site.Popup.TextInput("Create system", "Name:", func(name string) {
		if err := comms.Fetch(api.Post, "data/systems", name,
			&web.Data.Systems); err != nil {
			panic(err)
		}
		site.Refresh("")
	})
}

func (o *base) addGroup() {
	TemplateSelect(&site.Popup, GroupTemplate, func(pluginIndex int, templateIndex int, name string) {
		if err := comms.Fetch(api.Post, "data/groups", shared.GroupCreationRequest{
			name, pluginIndex, templateIndex}, &web.Data.Groups); err != nil {
			panic(err)
		}
		site.Refresh("")
	})
}

func newSystem(data *shared.System) *system {
	ret := new(system)
	ret.init(data)
	return ret
}

func (o *system) init(data *shared.System) {
	o.askewInit(data)
	o.reset()
}

func (o *system) reset() {
	o.name.ResetTo(o.data.Name)
}

func (o *system) commit() {
	go func() {
		if err := comms.Fetch(api.Put, "data/systems/"+o.data.ID,
			shared.SystemModificationRequest{Name: o.name.Value.Get()},
			&web.Data.Systems); err != nil {
			panic(err)
		}
		site.Refresh("s-" + o.data.ID)
	}()
}

func newGroup(data *shared.Group) *group {
	ret := new(group)
	ret.init(data)
	return ret
}

func (o *group) init(data *shared.Group) {
	o.askewInit(data)
	for _, s := range web.Data.Systems {
		o.Systems.AddItem(s.Name, false)
	}
	o.reset()

	for _, s := range data.Scenes {
		o.Scenes.Append(newListItem(s.Name, true, o.Scenes.Len()))
	}

	o.refreshHeroData()
}

func (o *group) reset() {
	o.name.ResetTo(o.data.Name)
	o.Systems.SetItem(o.data.SystemIndex, true)
	o.systemEdited.Set(false)
}

func (o *group) commit() {
	go func() {
		if err := comms.Fetch(api.Put, "data/groups/"+o.data.ID, o.name.Value.Get(),
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

func (o *group) delete(index int) {
	if len(o.data.Scenes) == 1 {
		site.Popup.ErrorMsg("Cannot delete last scene (group needs at least one scene).", nil)
		return
	}
	s := o.data.Scenes[index]
	site.Popup.Confirm("Really delete scene "+s.Name+"?", func() {
		if err := comms.Fetch(api.Delete,
			"data/groups/"+o.data.ID+"/scenes/"+s.ID, nil, &o.data.Scenes); err != nil {
			panic(err)
		}
		site.Refresh("g-" + o.data.ID)
	})
}

func (o *group) createScene() {
	TemplateSelect(&site.Popup, SceneTemplate, func(pluginIndex, templateIndex int, name string) {
		if err := comms.Fetch(api.Post, "data/groups/"+o.data.ID+"/scenes",
			shared.SceneCreationRequest{
				name, pluginIndex, templateIndex}, &o.data.Scenes); err != nil {
			panic(err)
		}
		site.Refresh("g-" + o.data.ID)
	})
}

func (o *group) createHero() {
	site.Popup.TextInput("Create Hero", "Name:", func(name string) {
		if err := comms.Fetch(api.Post, "data/groups/"+o.data.ID+"/heroes",
			name, &o.data.Heroes); err != nil {
			panic(err)
		}
		o.heroChooser.DestroyAll()
		for i, h := range o.data.Heroes {
			o.heroChooser.Append(newHeroButton(h.Name, i))
		}
		o.heroChooser.Item(o.heroChooser.Len() - 1).selected.Set(true)
		o.hero.disabled.Set(false)
		o.hero.loadHero(&o.data.Heroes[len(o.data.Heroes)-1])
	})
}

func (o *group) heroClicked(index int) {
	o.hero.loadHero(&o.data.Heroes[index])
	for i := 0; i < o.heroChooser.Len(); i++ {
		o.heroChooser.Item(i).selected.Set(i == index)
	}
}

func (o *group) refreshHeroData() {
	var oldID string
	if o.hero.data != nil {
		oldID = o.hero.data.ID
	}
	o.heroChooser.DestroyAll()
	o.hero.disabled.Set(len(o.data.Heroes) == 0)
	var selectedIndex int
	if len(o.data.Heroes) > 0 {
		for i, h := range o.data.Heroes {
			o.heroChooser.Append(newHeroButton(h.Name, i))
			if h.ID == oldID {
				selectedIndex = i
			}
		}
		o.heroChooser.Item(selectedIndex).selected.Set(true)
		o.hero.loadHero(&o.data.Heroes[selectedIndex])
	}
}

func (o *heroForm) loadHero(data *shared.Hero) {
	o.data = data
	o.reset()
}

func (o *heroForm) reset() {
	o.Name.ResetTo(o.data.Name)
	o.Description.ResetTo(o.data.Description)
}

func (o *heroForm) commit() {
	go func() {
		if err := comms.Fetch(api.Put,
			"data/groups/"+o.g.ID+"/heroes/"+o.data.ID,
			shared.HeroModificationRequest{Name: o.Name.Value.Get(),
				Description: o.Description.Value.Get()}, &o.g.Heroes); err != nil {
			panic(err)
		}
		o.Controller.refreshHeroData()
	}()
}

func (o *heroForm) delete() {
	go func() {
		if err := comms.Fetch(api.Delete,
			"data/groups/"+o.g.ID+"/heroes/"+o.data.ID,
			shared.HeroModificationRequest{Name: o.Name.Value.Get(),
				Description: o.Description.Value.Get()}, &o.g.Heroes); err != nil {
			panic(err)
		}
		o.Controller.refreshHeroData()
	}()
}

func (o *sceneModule) Swapped() {
	o.edited.Set(o.Toggle.Value.Get() != o.origValue)
}

func (o *sceneModule) reset() {
	o.Toggle.Value.Set(o.origValue)
	o.edited.Set(false)
}

func newScene(g *shared.Group, sceneIndex int) *scene {
	ret := new(scene)
	ret.init(g, sceneIndex)
	return ret
}

func (o *scene) init(g *shared.Group, sceneIndex int) {
	o.askewInit(g, sceneIndex)
	o.reset()
	for i, m := range web.StaticData.Modules {
		ui := newSceneModule(web.StaticData.Plugins[m.PluginIndex].Name, m.Name,
			o.g.Scenes[o.sceneIndex].Modules[i])
		o.modules.Append(ui)
		ui.Toggle.Value.Set(ui.origValue)
	}
}

func (o *scene) reset() {
	o.name.ResetTo(o.g.Scenes[o.sceneIndex].Name)
	for i := 0; i < o.modules.Len(); i++ {
		o.modules.Item(i).reset()
	}
}

func (o *scene) commit() {
	go func() {
		modules := make([]bool, o.modules.Len())
		for i := 0; i < o.modules.Len(); i++ {
			modules[i] = o.modules.Item(i).Toggle.Value.Get()
		}
		if err := comms.Fetch(api.Put,
			"data/groups/"+o.g.ID+"/scenes/"+o.g.Scenes[o.sceneIndex].ID,
			shared.SceneModificationRequest{Name: o.name.Value.Get(),
				Modules: modules}, &o.g.Scenes); err != nil {
			panic(err)
		}
		s := &o.g.Scenes[o.sceneIndex]
		for i := 0; i < o.modules.Len(); i++ {
			o.modules.Item(i).origValue = s.Modules[i]
		}
		o.reset()
	}()
}
