package data

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/flyx/pnpscreen/api"
	"github.com/flyx/pnpscreen/app"
)

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

func (c *Config) jsonSystems() []jsonItem {
	ret := make([]jsonItem, 0, len(c.systems))
	for i := range c.systems {
		ret = append(ret, jsonItem{Name: c.systems[i].Name,
			ID: c.systems[i].ID})
	}
	return ret
}

func (c *Config) jsonGroups() []jsonGroup {
	ret := make([]jsonGroup, 0, len(c.groups))
	for i := range c.groups {
		source := c.groups[i].Heroes
		heroes := make([]jsonHero, 0, len(source))
		for j := range source {
			heroes = append(heroes, jsonHero{Name: source[j].Name()})
		}

		ret = append(ret, jsonGroup{Name: c.groups[i].Config.Name,
			ID:          c.groups[i].Config.ID,
			SystemIndex: c.groups[i].Config.SystemIndex,
			Heroes:      heroes})
	}
	return ret
}

func (c *Config) jsonModules(app app.App) []jsonModuleDesc {
	ret := make([]jsonModuleDesc, 0, app.NumModules())
	for i := api.ModuleIndex(0); i < app.NumModules(); i++ {
		module := app.ModuleAt(i)
		modConfig := c.baseConfigs[i]
		modValue := reflect.ValueOf(modConfig).Elem()
		for ; modValue.Kind() == reflect.Interface ||
			modValue.Kind() == reflect.Ptr; modValue = modValue.Elem() {
		}
		cur := jsonModuleDesc{
			Name:   module.Name(),
			ID:     module.ID(),
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

type jsonGlobal struct {
	Systems     []jsonItem       `json:"systems"`
	Groups      []jsonGroup      `json:"groups"`
	Fonts       []string         `json:"fonts"`
	Modules     []jsonModuleDesc `json:"modules"`
	ActiveGroup int              `json:"activeGroup"`
}

// BuildGlobalJSON serializes the global config & environment state
// to JSON. It contains a list of systems, groups, fonts, modules and
// the currently active group index.
func (c *Config) BuildGlobalJSON(
	app app.App, activeGroup int) ([]byte, error) {
	return json.Marshal(jsonGlobal{
		Systems: c.jsonSystems(), Groups: c.jsonGroups(),
		Fonts: jsonFonts(app), Modules: c.jsonModules(app),
		ActiveGroup: activeGroup,
	})
}

// BuildBaseJSON returns a JSON serialization of the base configuration
// of each module.
func (c *Config) BuildBaseJSON() ([]byte, error) {
	return json.Marshal(c.buildModuleConfigJSON(c.baseConfigs))
}

// LoadBaseJSON parses the given config as JSON, updates the internal
// config and writes it to the base/config.yaml file
func (c *Config) LoadBaseJSON(raw []byte) error {
	if err := c.loadJSONModuleConfigs(raw, c.baseConfigs); err != nil {
		return err
	}
	c.writeYamlBaseConfig()
	return nil
}

// BuildSystemJSON serializes the config of the given system identified by its
// external ID, to JSON
func (c *Config) BuildSystemJSON(system string) ([]byte, error) {
	for i := range c.systems {
		if c.systems[i].ID == system {
			return json.Marshal(c.buildModuleConfigJSON(c.systems[i].Modules))
		}
	}
	return nil, fmt.Errorf("unknown system \"%s\"", system)
}

// LoadSystemJSON parses the given config as JSON, updates the internal
// config and writes it to the system's config.yaml file
func (c *Config) LoadSystemJSON(raw []byte, system string) error {
	for i := range c.systems {
		if c.systems[i].ID == system {
			if err := c.loadJSONModuleConfigs(raw, c.systems[i].Modules); err != nil {
				return err
			}
			c.writeYamlSystemConfig(c.systems[i])
			return nil
		}
	}
	return fmt.Errorf("unknown system \"%s\"", system)
}

// BuildGroupJSON serializes the config of the given group to JSON
func (c *Config) BuildGroupJSON(group string) ([]byte, error) {
	for i := range c.groups {
		if c.groups[i].Config.ID == group {
			return json.Marshal(c.buildModuleConfigJSON(c.groups[i].Config.Modules))
		}
	}
	return nil, fmt.Errorf("unknown group \"%s\"", group)
}

// LoadGroupJSON parses the given config as JSON, updates the internal
// config and writes it to the group's config.yaml file.
func (c *Config) LoadGroupJSON(raw []byte, group string) error {
	for i := range c.groups {
		if c.groups[i].Config.ID == group {
			if err := c.loadJSONModuleConfigs(raw, c.groups[i].Config.Modules); err != nil {
				return err
			}
			c.writeYamlGroupConfig(c.groups[i].Config)
			return nil
		}
	}
	return fmt.Errorf("unknown group \"%s\"", group)
}

func (c *Config) loadJSONModuleConfigInto(target interface{},
	raw []interface{}) error {
	targetModule := reflect.ValueOf(target).Elem()
	targetModuleType := targetModule.Type()
	for i := 0; i < targetModuleType.NumField(); i++ {
		wasNil := false
		if targetModule.Field(i).IsNil() {
			targetModule.Field(i).Set(reflect.New(targetModuleType.Field(i).Type.Elem()))
			wasNil = true
		}
		targetSetting := targetModule.Field(i).Interface()
		inModuleConfig := raw[i].(map[string]interface{})

		if err := c.setModuleConfigFieldFrom(targetSetting, false, inModuleConfig); err != nil {
			if wasNil {
				targetModule.Field(i).Set(reflect.Zero(targetModuleType.Field(i).Type))
			}
			return err
		}
	}
	return nil
}

func (c *Config) loadJSONModuleConfigs(jsonInput []byte,
	targetConfigs []interface{}) error {
	var raw []interface{}
	if err := json.Unmarshal(jsonInput, &raw); err != nil {
		return err
	}
	var i api.ModuleIndex
	for i = 0; i < api.ModuleIndex(len(targetConfigs)); i++ {
		moduleFields, ok := raw[i].([]interface{})
		if !ok {
			return fmt.Errorf("data for module\"%s\" is not a JSON array",
				c.owner.ModuleAt(i).Name())
		}
		conf := targetConfigs[i]
		if err := c.loadJSONModuleConfigInto(conf, moduleFields); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) buildModuleConfigJSON(config []interface{}) []jsonModuleConfig {
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

// BuildStateJSON returns a JSON serialization of the current state.
func (c *Config) BuildStateJSON() ([]byte, error) {
	list := make([]interface{}, c.owner.NumModules())
	var i api.ModuleIndex
	for i = 0; i < c.owner.NumModules(); i++ {
		list[i] = c.owner.ModuleAt(i).State().ToJSON()
	}
	return json.Marshal(list)
}
