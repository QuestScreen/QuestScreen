package data

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/flyx/pnpscreen/api"
	"github.com/flyx/pnpscreen/app"
	"gopkg.in/yaml.v3"
)

// Communication implements (de)serialization of data for communication to the
// client via HTTP/JSON
type Communication struct {
	*Config
}

type jsonModuleConfig []interface{}

type jsonItem struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type jsonHero struct {
	Name string `json:"name"`
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
	ret := make([]jsonItem, 0, len(c.Config.systems))
	for i := range c.Config.systems {
		ret = append(ret, jsonItem{Name: c.Config.systems[i].name,
			ID: c.Config.systems[i].id})
	}
	return ret
}

func (c Communication) groups() []jsonGroup {
	ret := make([]jsonGroup, 0, len(c.Config.groups))
	for i := range c.Config.groups {
		source := &c.Config.groups[i].heroes
		source.mutex.Lock()
		heroes := make([]jsonHero, 0, len(source.data))
		for j := range source.data {
			heroes = append(heroes, jsonHero{Name: source.data[j].Name()})
		}
		source.mutex.Unlock()

		scenes := make([]jsonItem, 0, len(c.Config.groups[i].scenes))
		for j := range c.Config.groups[i].scenes {
			scenes = append(scenes, jsonItem{
				Name: c.Config.groups[i].scenes[j].name,
				ID:   c.Config.groups[i].scenes[j].id,
			})
		}

		ret = append(ret, jsonGroup{Name: c.Config.groups[i].name,
			ID:          c.Config.groups[i].id,
			SystemIndex: c.Config.groups[i].systemIndex,
			Heroes:      heroes,
			Scenes:      scenes})
	}
	return ret
}

func (c Communication) modules(app app.App) []jsonModuleDesc {
	ret := make([]jsonModuleDesc, 0, app.NumModules())
	for i := api.ModuleIndex(0); i < app.NumModules(); i++ {
		module := app.ModuleAt(i).Descriptor()
		modConfig := c.baseConfigs[i]
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
		NumPluginSystems: c.numPluginSystems, Plugins: plugins}
}

// Datasets returns a serializable view of all dataset items (systems, groups,
// scenes, heroes).
func (c Communication) Datasets(app app.App) interface{} {
	return struct {
		Systems []jsonItem  `json:"systems"`
		Groups  []jsonGroup `json:"groups"`
	}{Systems: c.systems(), Groups: c.groups()}
}

// Base returns a serializable view of the base configuration.
func (c Communication) Base() interface{} {
	return c.moduleConfigs(c.baseConfigs)
}

// LoadBase parses the given config as JSON and updates the internal config
func (c Communication) LoadBase(raw []byte) error {
	if err := c.loadModuleConfigs(raw, c.baseConfigs); err != nil {
		return err
	}
	return nil
}

// System returns a serializable view the config of the given system identified
// by its ID
func (c Communication) System(systemID string) (interface{}, error) {
	for i := range c.Config.systems {
		if c.Config.systems[i].id == systemID {
			return c.moduleConfigs(c.Config.systems[i].modules), nil
		}
	}
	return nil, fmt.Errorf("unknown system \"%s\"", systemID)
}

// Systems returns a serializable view of all systems configs, as it would be
// contained in Datasets.
func (c Communication) Systems() interface{} {
	return c.systems()
}

// LoadSystem parses the given config as JSON and updates the internal config
func (c Communication) LoadSystem(raw []byte, systemID string) (System, error) {
	for i := range c.Config.systems {
		if c.Config.systems[i].id == systemID {
			if err := c.loadModuleConfigs(raw, c.Config.systems[i].modules); err != nil {
				return nil, err
			}
			return c.Config.systems[i], nil
		}
	}
	return nil, fmt.Errorf("unknown system \"%s\"", systemID)
}

// Group returns a serializable view of the config of the given group
func (c Communication) Group(groupID string) (interface{}, error) {
	for i := range c.Config.groups {
		if c.Config.groups[i].id == groupID {
			return c.moduleConfigs(c.Config.groups[i].modules), nil
		}
	}
	return nil, fmt.Errorf("unknown group \"%s\"", groupID)
}

// LoadGroup parses the given config as JSON and updates the internal config
func (c Communication) LoadGroup(raw []byte, groupID string) (Group, error) {
	for i := range c.Config.groups {
		if c.Config.groups[i].id == groupID {
			if err := c.loadModuleConfigs(raw, c.Config.groups[i].modules); err != nil {
				return nil, err
			}
			return c.Config.groups[i], nil
		}
	}
	return nil, fmt.Errorf("unknown group \"%s\"", groupID)
}

// Groups returns a serializable view of all groups configs, as it would be
// contained in Datasets.
func (c Communication) Groups() interface{} {
	return c.groups()
}

// Scene returns a serializable view of the config of the given scene of the
// given group.
func (c Communication) Scene(groupID string, sceneID string) (interface{}, error) {
	for i := range c.Config.groups {
		group := c.Config.groups[i]
		if group.id == groupID {
			for j := range group.scenes {
				scene := &group.scenes[j]
				if scene.id == sceneID {
					return c.sceneConfig(scene.modules), nil
				}
			}
		}
	}
	return nil, fmt.Errorf("unknown scene \"%s/%s\"", groupID, sceneID)
}

// LoadScene parses the given config as JSON and updates the internal config
func (c Communication) LoadScene(
	raw []byte, groupID string, sceneID string) (Group, Scene, error) {
	for i := range c.Config.groups {
		group := c.Config.groups[i]
		if group.id == groupID {
			for j := range group.scenes {
				scene := &group.scenes[j]
				if scene.id == sceneID {
					simpleList := make([]interface{}, c.owner.NumModules())
					for i := api.ModuleIndex(0); i < c.owner.NumModules(); i++ {
						simpleList[i] = scene.modules[i].config
					}
					if err := c.loadModuleConfigs(raw, simpleList); err != nil {
						return nil, nil, err
					}
					return group, scene, nil
				}
			}
			break
		}
	}
	return nil, nil, fmt.Errorf("unknown scene \"%s/%s\"", groupID, sceneID)
}

func (c Communication) loadModuleConfigInto(target interface{},
	raw []yaml.Node) error {
	targetModule := reflect.ValueOf(target).Elem()
	targetModuleType := targetModule.Type()
	for i := 0; i < targetModuleType.NumField(); i++ {
		node := &raw[i]
		if node.Kind == yaml.ScalarNode && node.Tag == "!!null" {
			targetModule.Field(i).Set(reflect.Zero(targetModuleType.Field(i).Type))
			continue
		}
		wasNil := false
		if targetModule.Field(i).IsNil() {
			targetModule.Field(i).Set(reflect.New(targetModuleType.Field(i).Type.Elem()))
			wasNil = true
		}
		targetSetting := targetModule.Field(i).Interface()

		if err := targetSetting.(api.ConfigItem).LoadFrom(
			node, c.owner, api.Web); err != nil {
			if wasNil {
				targetModule.Field(i).Set(reflect.Zero(targetModuleType.Field(i).Type))
			}
			return err
		}
	}
	return nil
}

func (c Communication) loadModuleConfigs(jsonInput []byte,
	targetConfigs []interface{}) error {
	var raw [][]yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(jsonInput))
	decoder.KnownFields(true)
	if err := decoder.Decode(&raw); err != nil {
		return err
	}
	var i api.ModuleIndex
	for i = 0; i < api.ModuleIndex(len(targetConfigs)); i++ {
		conf := targetConfigs[i]
		if conf == nil {
			if raw[i] != nil {
				return fmt.Errorf("got non-nil value for nil module")
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
	ret := make([]jsonModuleConfig, 0, c.owner.NumModules())
	var i api.ModuleIndex
	for i = 0; i < c.owner.NumModules(); i++ {
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
	ret := make([]jsonModuleConfig, 0, c.owner.NumModules())
	var i api.ModuleIndex
	for i = 0; i < c.owner.NumModules(); i++ {
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

// CommunicateSceneState returns a serializatable view of the current scene.
func (gs *GroupState) CommunicateSceneState(a app.App) interface{} {
	list := make([]interface{}, a.NumModules())
	var i api.ModuleIndex
	scene := gs.scenes[gs.activeScene]
	for i = 0; i < a.NumModules(); i++ {
		if scene[i] != nil {
			list[i] = scene[i].SerializableView(a, api.Web)
		}
	}
	return list
}
