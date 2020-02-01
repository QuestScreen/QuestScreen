package data

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/flyx/pnpscreen/api"
	"github.com/flyx/pnpscreen/app"
)

// Communication implements (de)serialization of data for communication to the
// client via HTTP/JSON
type Communication struct {
	d *Data
}

type jsonModuleConfig []interface{}

type jsonSystem struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type jsonHero struct {
	Name        string `json:"name"`
	ID          string `json:"id"`
	Description string `json:"description"`
}

type jsonScene struct {
	Name    string `json:"name"`
	ID      string `json:"id"`
	Modules []bool `json:"modules"`
}

type jsonGroup struct {
	Name        string      `json:"name"`
	ID          string      `json:"id"`
	SystemIndex int         `json:"systemIndex"`
	Heroes      []jsonHero  `json:"heroes"`
	Scenes      []jsonScene `json:"scenes"`
}

type jsonModuleSetting struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type jsonModuleDesc struct {
	Name        string              `json:"name"`
	ID          string              `json:"id"`
	Config      []jsonModuleSetting `json:"config"`
	PluginIndex int                 `json:"pluginIndex"`
}

func (c Communication) systems() []jsonSystem {
	ret := make([]jsonSystem, 0, len(c.d.systems))
	for i := range c.d.systems {
		ret = append(ret, jsonSystem{Name: c.d.systems[i].name,
			ID: c.d.systems[i].id})
	}
	return ret
}

func (c Communication) scenes(g *group) []jsonScene {
	scenes := make([]jsonScene, 0, len(g.scenes))
	for i := range g.scenes {
		s := &g.scenes[i]
		modules := make([]bool, c.d.owner.NumModules())
		for j := range modules {
			modules[j] = s.modules[j].enabled
		}
		scenes = append(scenes, jsonScene{
			Name: g.scenes[i].name, ID: g.scenes[i].id, Modules: modules})
	}
	return scenes
}

func (c Communication) heroes(hl *heroList) []jsonHero {
	ret := make([]jsonHero, 0, len(hl.data))
	for i := range hl.data {
		h := &hl.data[i]
		ret = append(ret,
			jsonHero{Name: h.name, ID: h.id, Description: h.description})
	}
	return ret
}

func (c Communication) groups() []jsonGroup {
	ret := make([]jsonGroup, 0, len(c.d.groups))
	for i := range c.d.groups {
		g := c.d.groups[i]

		g.heroes.mutex.Lock()
		ret = append(ret, jsonGroup{Name: g.name,
			ID:          g.id,
			SystemIndex: g.systemIndex,
			Heroes:      c.heroes(&g.heroes),
			Scenes:      c.scenes(g)})
		g.heroes.mutex.Unlock()
	}
	return ret
}

func (c Communication) modules(a app.App) []jsonModuleDesc {
	ret := make([]jsonModuleDesc, 0, a.NumModules())
	for i := app.FirstModule; i < a.NumModules(); i++ {
		module := a.ModuleAt(i).Descriptor()
		modConfig := c.d.baseConfigs[i]
		modValue := reflect.ValueOf(modConfig).Elem()
		for ; modValue.Kind() == reflect.Interface ||
			modValue.Kind() == reflect.Ptr; modValue = modValue.Elem() {
		}
		cur := jsonModuleDesc{
			Name:        module.Name,
			ID:          module.ID,
			Config:      make([]jsonModuleSetting, 0, modValue.NumField()),
			PluginIndex: a.ModulePluginIndex(i)}
		for j := 0; j < modValue.NumField(); j++ {
			cur.Config = append(cur.Config, jsonModuleSetting{
				Name: modValue.Type().Field(j).Name,
				Type: modValue.Type().Field(j).Type.Elem().Name()})
		}
		ret = append(ret, cur)
	}
	return ret
}

// StaticData returns a serializable view of all static data (i.e. data that
// will never change during the execution of PnPScreen)
func (c Communication) StaticData(app app.App, plugins interface{}) interface{} {
	return struct {
		Fonts            []string         `json:"fonts"`
		Modules          []jsonModuleDesc `json:"modules"`
		NumPluginSystems int              `json:"numPluginSystems"`
		Plugins          interface{}      `json:"plugins"`
	}{Fonts: app.FontNames(), Modules: c.modules(app),
		NumPluginSystems: c.d.numPluginSystems, Plugins: plugins}
}

// ViewAll returns a serializable view of all data items that are not part of
// the state (systems, groups, scenes, heroes).
func (c Communication) ViewAll(app app.App) interface{} {
	return struct {
		Systems []jsonSystem `json:"systems"`
		Groups  []jsonGroup  `json:"groups"`
	}{Systems: c.systems(), Groups: c.groups()}
}

// ViewBaseConfig returns a serializable view of the base configuration.
func (c Communication) ViewBaseConfig() interface{} {
	return c.moduleConfigs(c.d.baseConfigs)
}

// UpdateBaseConfig parses the given config as JSON and updates the config data
func (c Communication) UpdateBaseConfig(raw []byte) api.SendableError {
	if err := c.loadModuleConfigs(raw, c.d.baseConfigs); err != nil {
		return err
	}
	return nil
}

// ViewSystemConfig returns a serializable view the config of the given system
func (c Communication) ViewSystemConfig(s System) (interface{},
	api.SendableError) {
	return c.moduleConfigs(s.(*system).modules), nil
}

// ViewSystems returns a serializable view of all systems configs, as it would
// be contained in ViewAll.
func (c Communication) ViewSystems() interface{} {
	return c.systems()
}

// UpdateSystemConfig parses the given config as JSON and updates the internal config
func (c Communication) UpdateSystemConfig(raw []byte, s System) api.SendableError {
	return c.loadModuleConfigs(raw, s.(*system).modules)
}

// UpdateSystem updates a system's name from a given JSON input.
func (c Communication) UpdateSystem(raw []byte, s System) api.SendableError {
	data := struct {
		Name api.ValidatedString `json:"name"`
	}{Name: api.ValidatedString{MinLen: 1, MaxLen: -1}}
	if err := api.ReceiveData(raw, &data); err != nil {
		return err
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
func (c Communication) UpdateGroupConfig(raw []byte, g Group) api.SendableError {
	return c.loadModuleConfigs(raw, g.(*group).modules)
}

// UpdateGroup updates a group's name and linked system from a given JSON input.
func (c Communication) UpdateGroup(raw []byte, g Group) api.SendableError {
	value := struct {
		Name        api.ValidatedString `json:"name"`
		SystemIndex api.ValidatedInt    `json:"systemIndex"`
	}{
		Name:        api.ValidatedString{MinLen: 1, MaxLen: -1},
		SystemIndex: api.ValidatedInt{Min: -1, Max: c.d.NumSystems() - 1},
	}
	if err := api.ReceiveData(raw, &api.ValidatedStruct{Value: &value}); err != nil {
		return err
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
func (c Communication) UpdateHero(raw []byte, h api.Hero) api.SendableError {
	value := struct {
		Name        api.ValidatedString `json:"name"`
		Description string              `json:"description"`
	}{
		Name: api.ValidatedString{MinLen: 1, MaxLen: -1},
	}
	if err := api.ReceiveData(raw, &api.ValidatedStruct{Value: &value}); err != nil {
		return err
	}
	he := h.(*hero)
	he.name = value.Name.Value
	he.description = value.Description
	return nil
}

// ViewHeroes returns a serializable view of all heroes, as it would be
// contained in Datasets
func (c Communication) ViewHeroes(hl api.HeroList) interface{} {
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
func (c Communication) UpdateSceneConfig(raw []byte, s Scene) api.SendableError {
	simpleList := make([]interface{}, c.d.owner.NumModules())
	sc := s.(*scene)
	for i := app.FirstModule; i < c.d.owner.NumModules(); i++ {
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
func (c Communication) UpdateScene(raw []byte, g Group, s Scene) api.SendableError {
	var modules []bool
	data := struct {
		Name    api.ValidatedString `json:"name"`
		Modules api.ValidatedSlice  `json:"modules"`
	}{Name: api.ValidatedString{MinLen: 1, MaxLen: -1},
		Modules: api.ValidatedSlice{Data: &modules,
			MinItems: int(c.d.owner.NumModules()),
			MaxItems: int(c.d.owner.NumModules())}}
	if err := api.ReceiveData(raw, &data); err != nil {
		return err
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
	moduleIndex app.ModuleIndex, target interface{},
	raw []json.RawMessage) api.SendableError {
	targetModule := reflect.ValueOf(target).Elem()
	targetModuleType := targetModule.Type()
	heroes := c.d.owner.ViewHeroes()
	defer heroes.Close()
	ctx := c.d.owner.ServerContext(moduleIndex, heroes)
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

		if err := targetSetting.(api.ConfigItem).LoadWeb(input, ctx); err != nil {
			if wasNil {
				targetModule.Field(i).Set(reflect.Zero(targetModuleType.Field(i).Type))
			}
			return &api.BadRequest{Message: "error in JSON structure", Inner: err}
		}
	}
	return nil
}

func (c Communication) loadModuleConfigs(jsonInput []byte,
	targetConfigs []interface{}) api.SendableError {
	var raw [][]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(jsonInput))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return &api.BadRequest{Message: "error in JSON structure", Inner: err}
	}
	for i := app.FirstModule; i < app.ModuleIndex(len(targetConfigs)); i++ {
		conf := targetConfigs[i]
		if conf == nil {
			if raw[i] != nil {
				return &api.BadRequest{Message: "error in JSON structure",
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

func (c Communication) moduleConfigs(config []interface{}) []jsonModuleConfig {
	ret := make([]jsonModuleConfig, 0, c.d.owner.NumModules())
	for i := app.FirstModule; i < c.d.owner.NumModules(); i++ {
		moduleConfig := config[i]
		itemValue := reflect.ValueOf(moduleConfig).Elem()
		for ; itemValue.Kind() == reflect.Interface ||
			itemValue.Kind() == reflect.Ptr; itemValue = itemValue.Elem() {
		}
		jsonConfig := make(jsonModuleConfig, 0, itemValue.NumField())
		for j := 0; j < itemValue.NumField(); j++ {
			jsonConfig = append(jsonConfig, itemValue.Field(j).Interface())
		}
		ret = append(ret, jsonConfig)
	}
	return ret
}

func (c Communication) sceneConfig(config []sceneModule) []jsonModuleConfig {
	ret := make([]jsonModuleConfig, 0, c.d.owner.NumModules())
	for i := app.FirstModule; i < c.d.owner.NumModules(); i++ {
		moduleConfig := config[i]
		var jsonConfig jsonModuleConfig
		if moduleConfig.config != nil {
			itemValue := reflect.ValueOf(moduleConfig.config).Elem()
			for ; itemValue.Kind() == reflect.Interface ||
				itemValue.Kind() == reflect.Ptr; itemValue = itemValue.Elem() {
			}
			jsonConfig = make(jsonModuleConfig, 0, itemValue.NumField())
			for j := 0; j < itemValue.NumField(); j++ {
				jsonConfig = append(jsonConfig, itemValue.Field(j).Interface())
			}
		}
		ret = append(ret, jsonConfig)
	}
	return ret
}

// ViewSceneState returns a serializatable view of the current scene.
func (c Communication) ViewSceneState(a app.App) interface{} {
	list := make([]interface{}, a.NumModules())
	scene := c.d.State.scenes[c.d.State.activeScene]
	heroes := c.d.owner.ViewHeroes()
	defer heroes.Close()
	for i := app.FirstModule; i < a.NumModules(); i++ {
		if scene[i] != nil {
			list[i] = scene[i].WebView(a.ServerContext(i, heroes))
		}
	}
	return list
}
