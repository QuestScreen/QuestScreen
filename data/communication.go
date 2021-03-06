package data

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/QuestScreen/QuestScreen/app"
	"github.com/QuestScreen/QuestScreen/shared"
	"github.com/QuestScreen/QuestScreen/versioninfo"
	"github.com/QuestScreen/api/comms"
	"github.com/QuestScreen/api/config"
	"github.com/QuestScreen/api/groups"
	"github.com/QuestScreen/api/server"
)

// Communication implements (de)serialization of data for communication to the
// client via HTTP/JSON
type Communication struct {
	d *Data
}

func (c Communication) systems() []shared.System {
	ret := make([]shared.System, 0, len(c.d.systems))
	for i := range c.d.systems {
		ret = append(ret, shared.System{Name: c.d.systems[i].name,
			ID: c.d.systems[i].id})
	}
	return ret
}

func (c Communication) scenes(g *group) []shared.Scene {
	scenes := make([]shared.Scene, 0, len(g.scenes))
	for i := range g.scenes {
		s := &g.scenes[i]
		modules := make([]bool, c.d.owner.NumModules())
		for j := range modules {
			modules[j] = s.modules[j].enabled
		}
		scenes = append(scenes, shared.Scene{
			Name: g.scenes[i].name, ID: g.scenes[i].id, Modules: modules})
	}
	return scenes
}

func (c Communication) heroes(hl *heroList) []shared.Hero {
	ret := make([]shared.Hero, 0, len(hl.data))
	for i := range hl.data {
		h := &hl.data[i]
		ret = append(ret,
			shared.Hero{Name: h.name, ID: h.id, Description: h.description})
	}
	return ret
}

func (c Communication) groups() []shared.Group {
	ret := make([]shared.Group, 0, len(c.d.groups))
	for i := range c.d.groups {
		g := c.d.groups[i]

		ret = append(ret, shared.Group{Name: g.name,
			ID:          g.id,
			SystemIndex: g.systemIndex,
			Heroes:      c.heroes(&g.heroes),
			Scenes:      c.scenes(g)})
	}
	return ret
}

func (c Communication) modules(a app.App) []shared.Module {
	ret := make([]shared.Module, 0, a.NumModules())
	for i := shared.FirstModule; i < a.NumModules(); i++ {
		module := a.ModuleAt(i)
		modConfig := module.DefaultConfig
		modValue := reflect.ValueOf(modConfig).Elem()
		for ; modValue.Kind() == reflect.Interface ||
			modValue.Kind() == reflect.Ptr; modValue = modValue.Elem() {
		}
		cur := shared.Module{
			Name: module.Name,
			Path: a.PluginID(a.ModulePluginIndex(i)) + "/" + module.ID}
		ret = append(ret, cur)
	}
	return ret
}

func (c Communication) plugins(a app.App) []shared.Plugin {
	ret := make([]shared.Plugin, 0, a.NumPlugins())
	for i := 0; i < a.NumPlugins(); i++ {
		plugin := a.Plugin(i)
		descr := shared.Plugin{Name: plugin.Name, ID: a.PluginID(i)}
		for _, t := range plugin.SystemTemplates {
			descr.SystemTemplates = append(descr.SystemTemplates, shared.TemplateDescr{
				Name: t.Name, Description: t.Name,
			})
		}
		for _, t := range plugin.GroupTemplates {
			descr.GroupTemplates = append(descr.GroupTemplates, shared.TemplateDescr{
				Name: t.Name, Description: t.Description,
			})
		}
		for _, t := range plugin.SceneTemplates {
			descr.SceneTemplates = append(descr.SceneTemplates, shared.TemplateDescr{
				Name: t.Name, Description: t.Description,
			})
		}

		ret = append(ret, descr)
	}

	return ret
}

// StaticData returns a serializable view of all static data (i.e. data that
// will never change during the execution of PnPScreen)
func (c Communication) StaticData(a app.App, plugins interface{}) interface{} {
	textures := a.GetTextures()
	return shared.Static{Fonts: a.FontNames(), Textures: textures,
		Modules: c.modules(a), Plugins: c.plugins(a),
		NumPluginSystems: c.d.numPluginSystems,
		FontDir:          a.DataDir("fonts"), Messages: a.Messages(),
		AppVersion: versioninfo.CurrentVersion}
}

// ViewAll returns a serializable view of all data items that are not part of
// the state (systems, groups, scenes, heroes).
func (c Communication) ViewAll(app app.App) shared.Data {
	return shared.Data{Systems: c.systems(), Groups: c.groups()}
}

// ViewBaseConfig returns a serializable view of the base configuration.
func (c Communication) ViewBaseConfig() interface{} {
	return c.moduleConfigs(c.d.baseConfigs)
}

// UpdateBaseConfig parses the given config as JSON and updates the config data
func (c Communication) UpdateBaseConfig(raw []byte) server.Error {
	if err := c.loadModuleConfigs(raw, c.d.baseConfigs); err != nil {
		return err
	}
	return nil
}

// ViewSystemConfig returns a serializable view the config of the given system
func (c Communication) ViewSystemConfig(s System) (interface{},
	server.Error) {
	return c.moduleConfigs(s.(*system).modules), nil
}

// ViewSystems returns a serializable view of all systems configs, as it would
// be contained in ViewAll.
func (c Communication) ViewSystems() interface{} {
	return c.systems()
}

// UpdateSystemConfig parses the given config as JSON and updates the internal config
func (c Communication) UpdateSystemConfig(raw []byte, s System) server.Error {
	return c.loadModuleConfigs(raw, s.(*system).modules)
}

// UpdateSystem updates a system's name from a given JSON input.
func (c Communication) UpdateSystem(raw []byte, s System) server.Error {
	data := struct {
		Name comms.ValidatedString `json:"name"`
	}{Name: comms.ValidatedString{MinLen: 1, MaxLen: -1}}
	if err := comms.ReceiveData(raw, &data); err != nil {
		return &server.BadRequest{Inner: err, Message: "received invalid data"}
	}
	s.(*system).name = data.Name.Value
	// TODO: sort system list anew
	return nil
}

// ViewGroupConfig returns a serializable view of the config of the given group
func (c Communication) ViewGroupConfig(g Group) interface{} {
	return c.moduleConfigs(g.(*group).modules)
}

// UpdateGroupConfig parses the given config as JSON and updates the internal config
func (c Communication) UpdateGroupConfig(raw []byte, g Group) server.Error {
	return c.loadModuleConfigs(raw, g.(*group).modules)
}

// UpdateGroup updates a group's name and linked system from a given JSON input.
func (c Communication) UpdateGroup(raw []byte, g Group) server.Error {
	value := struct {
		Name        comms.ValidatedString `json:"name"`
		SystemIndex comms.ValidatedInt    `json:"systemIndex"`
	}{
		Name:        comms.ValidatedString{MinLen: 1, MaxLen: -1},
		SystemIndex: comms.ValidatedInt{Min: -1, Max: c.d.NumSystems() - 1},
	}
	if err := comms.ReceiveData(raw, &comms.ValidatedStruct{Value: &value}); err != nil {
		return &server.BadRequest{Inner: err, Message: "received invalid data"}
	}
	gr := g.(*group)
	gr.name = value.Name.Value
	gr.systemIndex = value.SystemIndex.Value
	return nil
}

// ViewGroups returns a serializable view of all groups, as it would be
// contained in Datasets.
func (c Communication) ViewGroups() interface{} {
	return c.groups()
}

// UpdateHero updates a hero's name and description form a given JSON input.
func (c Communication) UpdateHero(raw []byte, h groups.Hero) server.Error {
	value := struct {
		Name        comms.ValidatedString `json:"name"`
		Description string                `json:"description"`
	}{
		Name: comms.ValidatedString{MinLen: 1, MaxLen: -1},
	}
	if err := comms.ReceiveData(raw, &comms.ValidatedStruct{Value: &value}); err != nil {
		return &server.BadRequest{Inner: err, Message: "received invalid data"}
	}
	he := h.(*hero)
	he.name = value.Name.Value
	he.description = value.Description
	return nil
}

// ViewHeroes returns a serializable view of all heroes, as it would be
// contained in Datasets
func (c Communication) ViewHeroes(hl groups.HeroList) interface{} {
	return c.heroes(hl.(*heroList))
}

// ViewSceneConfig returns a serializable view of the config of the given scene.
func (c Communication) ViewSceneConfig(s Scene) interface{} {
	return c.sceneConfig(s.(*scene).modules)
}

// ViewScenes returns a serializable view of all scenes, as it would be
// contained in ViewAll.
func (c Communication) ViewScenes(g Group) interface{} {
	return c.scenes(g.(*group))
}

// UpdateSceneConfig parses the given JSON input and updates the scene's config
func (c Communication) UpdateSceneConfig(raw []byte, s Scene) server.Error {
	simpleList := make([]interface{}, c.d.owner.NumModules())
	sc := s.(*scene)
	for i := shared.FirstModule; i < c.d.owner.NumModules(); i++ {
		simpleList[i] = sc.modules[i].config
	}
	return c.loadModuleConfigs(raw, simpleList)
}

func isNull(msg json.RawMessage) bool {
	state := 0
	for i := 0; i < len(msg); i++ {
		switch state {
		case 0:
			switch msg[i] {
			case ' ' | '\t' | '\n':
				break
			case 'n':
				state = 1
			default:
				return false
			}
		case 1:
			if msg[i] == 'u' {
				state = 2
			} else {
				return false
			}
		case 2 | 3:
			if msg[i] == 'l' {
				state++
			} else {
				return false
			}
		case 4:
			switch msg[i] {
			case ' ' | '\t' | '\n':
				break
			default:
				return false
			}
		}
	}
	return true
}

// UpdateScene updates a scene's name
func (c Communication) UpdateScene(raw []byte, g Group, s Scene) server.Error {
	var modules []bool
	data := struct {
		Name    comms.ValidatedString `json:"name"`
		Modules comms.ValidatedSlice  `json:"modules"`
	}{Name: comms.ValidatedString{MinLen: 1, MaxLen: -1},
		Modules: comms.ValidatedSlice{Data: &modules,
			MinItems: int(c.d.owner.NumModules()),
			MaxItems: int(c.d.owner.NumModules())}}
	if err := comms.ReceiveData(raw, &data); err != nil {
		return &server.BadRequest{Inner: err, Message: "received invalid data"}
	}
	sc := s.(*scene)
	sc.name = data.Name.Value
	for i := range modules {
		sc.modules[i].enabled = modules[i]
	}
	// TODO: sort scene list anew
	return nil
}

func (c Communication) loadModuleConfigInto(
	moduleIndex shared.ModuleIndex, target interface{},
	raw []json.RawMessage) server.Error {
	targetModule := reflect.ValueOf(target).Elem()
	targetModuleType := targetModule.Type()
	ctx := c.d.owner.ServerContext(moduleIndex)
	for i := 0; i < targetModuleType.NumField(); i++ {
		input := raw[i]
		if isNull(input) {
			targetModule.Field(i).Set(reflect.Zero(targetModuleType.Field(i).Type))
			continue
		}
		wasNil := false
		if targetModule.Field(i).IsNil() {
			targetModule.Field(i).Set(reflect.New(targetModuleType.Field(i).Type.Elem()))
			wasNil = true
		}
		targetSetting := targetModule.Field(i).Interface()

		if err := targetSetting.(config.Item).Receive(input, ctx); err != nil {
			if wasNil {
				targetModule.Field(i).Set(reflect.Zero(targetModuleType.Field(i).Type))
			}
			return &server.BadRequest{Message: "error in JSON structure", Inner: err}
		}
	}
	return nil
}

func (c Communication) loadModuleConfigs(jsonInput []byte,
	targetConfigs []interface{}) server.Error {
	var raw [][]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(jsonInput))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return &server.BadRequest{Message: "error in JSON structure", Inner: err}
	}
	for i := shared.FirstModule; i < shared.ModuleIndex(len(targetConfigs)); i++ {
		conf := targetConfigs[i]
		if conf == nil {
			if raw[i] != nil {
				return &server.BadRequest{Message: "error in JSON structure",
					Inner: fmt.Errorf("got non-nil value for nil module")}
			}
		} else {
			if err := c.loadModuleConfigInto(i, conf, raw[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c Communication) moduleConfigs(config []interface{}) []shared.ModuleConfig {
	ret := make([]shared.ModuleConfig, 0, c.d.owner.NumModules())
	for i := shared.FirstModule; i < c.d.owner.NumModules(); i++ {
		moduleConfig := config[i]
		itemValue := reflect.ValueOf(moduleConfig).Elem()
		for ; itemValue.Kind() == reflect.Interface ||
			itemValue.Kind() == reflect.Ptr; itemValue = itemValue.Elem() {
		}
		jsonConfig := make(shared.ModuleConfig, 0, itemValue.NumField())
		for j := 0; j < itemValue.NumField(); j++ {
			jsonConfig = append(jsonConfig, itemValue.Field(j).Interface())
		}
		ret = append(ret, jsonConfig)
	}
	return ret
}

func (c Communication) sceneConfig(config []sceneModule) []shared.ModuleConfig {
	ret := make([]shared.ModuleConfig, 0, c.d.owner.NumModules())
	for i := shared.FirstModule; i < c.d.owner.NumModules(); i++ {
		moduleConfig := config[i]
		var jsonConfig shared.ModuleConfig
		if moduleConfig.config != nil {
			itemValue := reflect.ValueOf(moduleConfig.config).Elem()
			for ; itemValue.Kind() == reflect.Interface ||
				itemValue.Kind() == reflect.Ptr; itemValue = itemValue.Elem() {
			}
			jsonConfig = make(shared.ModuleConfig, 0, itemValue.NumField())
			for j := 0; j < itemValue.NumField(); j++ {
				jsonConfig = append(jsonConfig, itemValue.Field(j).Interface())
			}
		}
		ret = append(ret, jsonConfig)
	}
	return ret
}

// ViewSceneState returns the current scene's serialized state
func (c Communication) ViewSceneState(a app.App) []json.RawMessage {
	list := make([]json.RawMessage, a.NumModules())
	scene := c.d.State.scenes[c.d.State.activeScene]
	for i := shared.FirstModule; i < a.NumModules(); i++ {
		if scene[i] != nil {
			var err error
			list[i], err = json.Marshal(scene[i].Send(a.ServerContext(i)))
			if err != nil {
				panic(err)
			}
		}
	}
	return list
}
