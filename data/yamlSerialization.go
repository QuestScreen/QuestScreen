package data

import (
	"fmt"
	"io/ioutil"
	"log"
	"reflect"

	"github.com/flyx/pnpscreen/api"
	"github.com/flyx/pnpscreen/app"
	"gopkg.in/yaml.v3"
)

type yamlBaseConfig struct {
	Modules map[string]map[string]interface{}
}

// yamlSystemConfig is the structure system configuration
// is stored in YAML files. This differs from the internal
// structure as it uses module names and setting names as
// keys in YAML mappings rather than using a list which
// maps config items by position to a module / setting.
type yamlSystemConfig struct {
	Name string
	// module name -> (setting name -> value)
	Modules map[string]map[string]interface{}
}

// yamlGroupConfig is yamlSystemConfig for groups
type yamlGroupConfig struct {
	Name    string
	System  string
	Modules map[string]map[string]interface{}
}

func (c *Config) loadYamlModuleConfigInto(target interface{},
	values map[string]interface{}, moduleName string) bool {
	targetModule := reflect.ValueOf(target).Elem()
	targetModuleType := targetModule.Type()
	for i := 0; i < targetModuleType.NumField(); i++ {
		inValue, ok := values[targetModuleType.Field(i).Name]
		if !ok {
			continue
		}
		inModuleConfig, ok := inValue.(map[string]interface{})
		// this is a fix for a problem in the yaml lib that
		// leads to yaml giving the type map[interface{}]interface{}.
		if !ok {
			raw, ok := inValue.(map[interface{}]interface{})
			if !ok {
				panic("value of " + moduleName + "." +
					targetModuleType.Field(i).Name + " is not a mapping")
			}
			inModuleConfig = make(map[string]interface{})
			for key, value := range raw {
				stringKey, ok := key.(string)
				if !ok {
					panic("value of" + moduleName + "." +
						targetModuleType.Field(i).Name + " contains non-string key")
				}
				inModuleConfig[stringKey] = value
			}
		}
		wasNil := false
		if targetModule.Field(i).IsNil() {
			targetModule.Field(i).Set(reflect.New(targetModuleType.Field(i).Type.Elem()))
			wasNil = true
		}
		targetSetting := targetModule.Field(i).Interface()

		if err := c.setModuleConfigFieldFrom(targetSetting, true, inModuleConfig); err != nil {
			if wasNil {
				targetModule.Field(i).Set(reflect.Zero(targetModuleType.Field(i).Type))
			}
			log.Println(err)
			return false
		}
	}
	return true
}

func findModule(owner app.App, id string) (api.Module, api.ModuleIndex) {
	var i api.ModuleIndex
	for i = 0; i < owner.NumModules(); i++ {
		module := owner.ModuleAt(i)
		if module.ID() == id {
			return module, i
		}
	}
	return nil, -1
}

func (c *Config) loadYamlModuleConfigs(
	raw map[string]map[string]interface{}) ([]interface{}, error) {
	ret := make([]interface{}, c.owner.NumModules())
	for name, rawItems := range raw {
		mod, index := findModule(c.owner, name)
		if mod == nil {
			return nil, fmt.Errorf("Unknown module \"%s\"", name)
		}
		target := mod.EmptyConfig()
		if c.loadYamlModuleConfigInto(target, rawItems, mod.ID()) {
			ret[index] = target
		}
	}
	var i api.ModuleIndex
	for i = 0; i < c.owner.NumModules(); i++ {
		if ret[i] == nil {
			ret[i] = c.owner.ModuleAt(i).EmptyConfig()
		}
	}
	return ret, nil
}

func (c *Config) toYamlStructure(moduleConfigs []interface{}) map[string]map[string]interface{} {
	ret := make(map[string]map[string]interface{})
	var i api.ModuleIndex
	for i = 0; i < c.owner.NumModules(); i++ {
		var fields map[string]interface{}

		moduleConfig := moduleConfigs[i]
		value := reflect.ValueOf(moduleConfig)
		valueType := reflect.TypeOf(moduleConfig)
		for valueType.Kind() == reflect.Interface ||
			valueType.Kind() == reflect.Ptr {
			valueType = valueType.Elem()
			value = value.Elem()
		}
		if valueType.Kind() != reflect.Struct || value.Kind() != reflect.Struct {
			panic("value type is not a struct!")
		}
		for j := 0; j < valueType.NumField(); j++ {
			tagVal, ok := valueType.Field(j).Tag.Lookup("yaml")
			fieldName := valueType.Field(j).Name
			if ok {
				if tagVal == "-" {
					continue
				}
				fieldName = tagVal
			}
			fieldVal := value.Field(j)
			if !fieldVal.IsNil() {
				if fields == nil {
					fields = make(map[string]interface{})
				}
				fields[fieldName] = fieldVal.Interface()
			}
		}
		if fields != nil {
			ret[c.owner.ModuleAt(i).ID()] = fields
		}
	}
	return ret
}

func (c *Config) loadYamlBaseConfig(yamlInput []byte) ([]interface{}, error) {
	var data yamlBaseConfig
	if yamlInput != nil {
		if err := yaml.Unmarshal(yamlInput, &data); err != nil {
			return nil, err
		}
	} else {
		data.Modules = make(map[string]map[string]interface{})
	}

	return c.loadYamlModuleConfigs(data.Modules)
}

func (c *Config) writeYamlBaseConfig() {
	data := yamlBaseConfig{Modules: c.toYamlStructure(c.baseConfigs)}
	path := c.owner.DataDir("base", "config.yaml")
	raw, _ := yaml.Marshal(data)
	ioutil.WriteFile(path, raw, 0644)
}

func (c *Config) loadYamlSystemConfig(
	id string, yamlInput []byte) (systemConfig, error) {
	var data yamlSystemConfig
	if err := yaml.Unmarshal(yamlInput, &data); err != nil {
		return systemConfig{}, err
	}
	moduleConfigs, err := c.loadYamlModuleConfigs(data.Modules)
	return systemConfig{
		Name:    data.Name,
		ID:      id,
		Modules: moduleConfigs}, err
}

func (c *Config) writeYamlSystemConfig(config systemConfig) {
	data := yamlSystemConfig{
		Name:    config.Name,
		Modules: c.toYamlStructure(config.Modules),
	}
	path := c.owner.DataDir("systems", config.ID, "config.yaml")
	raw, _ := yaml.Marshal(data)
	ioutil.WriteFile(path, raw, 0644)
}

func (c *Config) loadYamlGroupConfig(
	id string, yamlInput []byte) (groupConfig, error) {
	var data yamlGroupConfig
	if err := yaml.Unmarshal(yamlInput, &data); err != nil {
		return groupConfig{}, err
	}
	for i := 0; i < len(c.systems); i++ {
		if c.systems[i].ID == data.System {
			moduleConfigs, err := c.loadYamlModuleConfigs(data.Modules)
			return groupConfig{
				Name:        data.Name,
				ID:          id,
				SystemIndex: i,
				Modules:     moduleConfigs,
			}, err
		}
	}
	return groupConfig{},
		fmt.Errorf("Group config references unknown system \"%s\"", data.System)
}

func (c *Config) writeYamlGroupConfig(group groupConfig) {
	data := yamlGroupConfig{
		Name:    group.Name,
		System:  c.systems[group.SystemIndex].ID,
		Modules: c.toYamlStructure(group.Modules),
	}
	path := c.owner.DataDir("groups", group.ID, "config.yaml")
	raw, _ := yaml.Marshal(data)
	ioutil.WriteFile(path, raw, 0644)
}

// LoadYamlGroupState loads the given YAML input into the current group's state.
func (c *Config) LoadYamlGroupState(yamlInput []byte) {
	// module name -> state value
	var data map[string]interface{}
	if err := yaml.Unmarshal(yamlInput, &data); err != nil {
		panic("while parsing group state: " + err.Error())
	}
	ret := make([]bool, c.owner.NumModules())
	var i api.ModuleIndex
	for k, v := range data {
		found := false
		for i = 0; i < c.owner.NumModules(); i++ {
			module := c.owner.ModuleAt(i)
			if k == module.ID() {
				if err := module.State().LoadFrom(v, c.owner); err != nil {
					log.Println("Could not load state", k, ":", err.Error())
				} else {
					ret[i] = true
				}
				found = true
				break
			}
		}
		if !found {
			log.Println("while loading state: unknown module \"", k, "\"")
		}
	}
	for i = 0; i < c.owner.NumModules(); i++ {
		if !ret[i] {
			module := c.owner.ModuleAt(i)
			log.Printf("missing state for module \"%s\", loading default",
				module.ID())
			module.State().LoadFrom(nil, c.owner)
		}
	}
}

// BuildStateYaml writes YAML output describing the state of the current
// module.
func (c *Config) BuildStateYaml() ([]byte, error) {
	structure := make(map[string]interface{})
	var i api.ModuleIndex
	for i = 0; i < c.owner.NumModules(); i++ {
		module := c.owner.ModuleAt(i)
		structure[module.ID()] = module.State().ToYAML(c.owner)
	}
	return yaml.Marshal(structure)
}
