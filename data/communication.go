package data

import (
	"bytes"
	"encoding/json"
	"errors"
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

type jsonItem struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type jsonHero struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type jsonGroup struct {
	Name        string     `json:"name"`
	ID          string     `json:"id"`
	SystemIndex int        `json:"systemIndex"`
	Heroes      []jsonHero `json:"heroes"`
	Scenes      []jsonItem `json:"scenes"`
}

type jsonModuleSetting struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type jsonModuleDesc struct {
	Name   string              `json:"name"`
	ID     string              `json:"id"`
	Config []jsonModuleSetting `json:"config"`
}

func jsonFonts(env api.Environment) []string {
	fonts := env.FontCatalog()
	ret := make([]string, 0, len(fonts))
	for i := range fonts {
		ret = append(ret, fonts[i].Name())
	}
	return ret
}

func (c Communication) systems() []jsonItem {
	ret := make([]jsonItem, 0, len(c.d.systems))
	for i := range c.d.systems {
		ret = append(ret, jsonItem{Name: c.d.systems[i].name,
			ID: c.d.systems[i].id})
	}
	return ret
}

func (c Communication) scenes(g *group) []jsonItem {
	scenes := make([]jsonItem, 0, len(g.scenes))
	for i := range g.scenes {
		scenes = append(scenes, jsonItem{
			Name: g.scenes[i].name, ID: g.scenes[i].id})
	}
	return scenes
}

func (c Communication) groups() []jsonGroup {
	ret := make([]jsonGroup, 0, len(c.d.groups))
	for i := range c.d.groups {
		source := &c.d.groups[i].heroes
		source.mutex.Lock()
		heroes := make([]jsonHero, 0, len(source.data))
		for j := range source.data {
			heroes = append(heroes,
				jsonHero{Name: source.data[j].Name(), ID: source.data[j].ID()})
		}
		source.mutex.Unlock()

		scenes := c.scenes(c.d.groups[i])

		ret = append(ret, jsonGroup{Name: c.d.groups[i].name,
			ID:          c.d.groups[i].id,
			SystemIndex: c.d.groups[i].systemIndex,
			Heroes:      heroes,
			Scenes:      scenes})
	}
	return ret
}

func (c Communication) modules(app app.App) []jsonModuleDesc {
	ret := make([]jsonModuleDesc, 0, app.NumModules())
	for i := api.ModuleIndex(0); i < app.NumModules(); i++ {
		module := app.ModuleAt(i).Descriptor()
		modConfig := c.d.baseConfigs[i]
		modValue := reflect.ValueOf(modConfig).Elem()
		for ; modValue.Kind() == reflect.Interface ||
			modValue.Kind() == reflect.Ptr; modValue = modValue.Elem() {
		}
		cur := jsonModuleDesc{
			Name:   module.Name,
			ID:     module.ID,
			Config: make([]jsonModuleSetting, 0, modValue.NumField())}
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
	}{Fonts: jsonFonts(app), Modules: c.modules(app),
		NumPluginSystems: c.d.numPluginSystems, Plugins: plugins}
}

// ViewAll returns a serializable view of all data items that are not part of
// the state (systems, groups, scenes, heroes).
func (c Communication) ViewAll(app app.App) interface{} {
	return struct {
		Systems []jsonItem  `json:"systems"`
		Groups  []jsonGroup `json:"groups"`
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

type groupUpdateReceiver struct {
	data struct {
		Name        string `json:"name"`
		SystemIndex int    `json:"systemIndex"`
	}
	maxSystems int
}

func (gur *groupUpdateReceiver) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &gur.data); err != nil {
		return err
	}
	if gur.data.Name == "" {
		return errors.New("name must not be empty")
	} else if gur.data.SystemIndex < 0 || gur.data.SystemIndex > gur.maxSystems {
		return fmt.Errorf("system index outside of required range [0..%d]",
			gur.maxSystems)
	}
	return nil
}

// UpdateGroup updates a group's name and linked system from a given JSON input.
// It returns the group's index on success.
func (c Communication) UpdateGroup(raw []byte, g Group) api.SendableError {
	value := groupUpdateReceiver{maxSystems: c.d.NumSystems() - 1}
	if err := api.ReceiveData(raw, &value); err != nil {
		return err
	}
	gr := g.(*group)
	gr.name = value.data.Name
	gr.systemIndex = value.data.SystemIndex
	return nil
}

// ViewGroups returns a serializable view of all groups, as it would be
// contained in Datasets.
func (c Communication) ViewGroups() interface{} {
	return c.groups()
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
	for i := api.ModuleIndex(0); i < c.d.owner.NumModules(); i++ {
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

func (c Communication) loadModuleConfigInto(target interface{},
	raw []json.RawMessage) api.SendableError {
	targetModule := reflect.ValueOf(target).Elem()
	targetModuleType := targetModule.Type()
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

		if err := targetSetting.(api.ConfigItem).LoadWeb(input, c.d.owner); err != nil {
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
	var i api.ModuleIndex
	for i = 0; i < api.ModuleIndex(len(targetConfigs)); i++ {
		conf := targetConfigs[i]
		if conf == nil {
			if raw[i] != nil {
				return &api.BadRequest{Message: "error in JSON structure",
					Inner: fmt.Errorf("got non-nil value for nil module")}
			}
		} else {
			if err := c.loadModuleConfigInto(conf, raw[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c Communication) moduleConfigs(config []interface{}) []jsonModuleConfig {
	ret := make([]jsonModuleConfig, 0, c.d.owner.NumModules())
	var i api.ModuleIndex
	for i = 0; i < c.d.owner.NumModules(); i++ {
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
	var i api.ModuleIndex
	for i = 0; i < c.d.owner.NumModules(); i++ {
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
	var i api.ModuleIndex
	scene := c.d.State.scenes[c.d.State.activeScene]
	for i = 0; i < a.NumModules(); i++ {
		if scene[i] != nil {
			list[i] = scene[i].WebView(a)
		}
	}
	return list
}
