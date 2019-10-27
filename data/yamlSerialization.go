package data

import (
	"io/ioutil"
	"log"
	"path/filepath"
	"reflect"

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

func (s *StaticData) loadYamlModuleConfigInto(target interface{},
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

		if err := s.setModuleConfigFieldFrom(targetSetting, true, inModuleConfig); err != nil {
			if wasNil {
				targetModule.Field(i).Set(reflect.Zero(targetModuleType.Field(i).Type))
			}
			log.Println(err)
			return false
		}
	}
	return true
}

func (s *StaticData) loadYamlModuleConfigs(
	raw map[string]map[string]interface{},
	items ConfigurableItemProvider) []interface{} {
	ret := make([]interface{}, items.NumItems())
	for name, rawItems := range raw {
		mod, index := findItem(items, name)
		if mod == nil {
			log.Println("Unknown module: " + name)
		} else {
			target := mod.EmptyConfig()
			if s.loadYamlModuleConfigInto(target, rawItems, mod.InternalName()) {
				ret[index] = target
			}
		}
	}
	for i := 0; i < items.NumItems(); i++ {
		if ret[i] == nil {
			ret[i] = items.ItemAt(i).EmptyConfig()
		}
	}
	return ret
}

func (s *StaticData) toYamlStructure(moduleConfigs []interface{}) map[string]map[string]interface{} {
	ret := make(map[string]map[string]interface{})
	for i := 0; i < len(moduleConfigs); i++ {
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
			ret[s.items.ItemAt(i).Name()] = fields
		}
	}
	return ret
}

func (s *StaticData) loadYamlBaseConfig(yamlInput []byte) []interface{} {
	var data yamlBaseConfig
	if err := yaml.Unmarshal(yamlInput, &data); err != nil {
		panic("while parsing base config: " + err.Error())
	}
	return s.loadYamlModuleConfigs(data.Modules, s.items)
}

func (s *StaticData) writeYamlBaseConfig(moduleConfigs []interface{}) {
	data := yamlBaseConfig{Modules: s.toYamlStructure(moduleConfigs)}
	path := filepath.Join(s.DataDir, "base", "config.yaml")
	raw, _ := yaml.Marshal(data)
	ioutil.WriteFile(path, raw, 0644)
}

func (s *StaticData) loadYamlSystemConfig(
	yamlInput []byte, dirName string) systemConfig {
	var data yamlSystemConfig
	if err := yaml.Unmarshal(yamlInput, &data); err != nil {
		panic("while parsing system config of " + dirName + ": " + err.Error())
	}
	return systemConfig{
		Name:    data.Name,
		DirName: dirName,
		Modules: s.loadYamlModuleConfigs(data.Modules, s.items)}
}

func (s *StaticData) writeYamlSystemConfig(config systemConfig) {
	data := yamlSystemConfig{
		Name:    config.Name,
		Modules: s.toYamlStructure(config.Modules),
	}
	path := filepath.Join(s.DataDir, "systems", config.DirName, "config.yaml")
	raw, _ := yaml.Marshal(data)
	ioutil.WriteFile(path, raw, 0644)
}

func (s *StaticData) loadYamlGroupConfig(
	yamlInput []byte, dirName string, systems []systemConfig) groupConfig {
	var data yamlGroupConfig
	if err := yaml.Unmarshal(yamlInput, &data); err != nil {
		panic("while parsing group config of " + dirName + ": " + err.Error())
	}
	for i := 0; i < len(systems); i++ {
		if systems[i].Name == data.System {
			return groupConfig{
				Name:        data.Name,
				DirName:     dirName,
				SystemIndex: i,
				Modules:     s.loadYamlModuleConfigs(data.Modules, s.items),
			}
		}
	}
	panic("Group config of " + dirName + " references unknown system \"" + data.System + "\"")
}

func (s *StaticData) writeYamlGroupConfig(group groupConfig, systems []systemConfig) {
	data := yamlGroupConfig{
		Name:    group.Name,
		System:  systems[group.SystemIndex].Name,
		Modules: s.toYamlStructure(group.Modules),
	}
	path := filepath.Join(s.DataDir, "groups", group.DirName, "config.yaml")
	raw, _ := yaml.Marshal(data)
	ioutil.WriteFile(path, raw, 0644)
}

// module name -> state value
type yamlGroupState map[string]interface{}

func (s *Store) loadYamlGroupState(yamlInput []byte) {
	var data yamlGroupState
	if err := yaml.Unmarshal(yamlInput, &data); err != nil {
		panic("while parsing group state: " + err.Error())
	}
	ret := make([]bool, s.items.NumItems())
	for k, v := range data {
		found := false
		for i := 0; i < s.items.NumItems(); i++ {
			if k == s.items.ItemAt(i).Name() {
				if err := s.items.ItemAt(i).GetState().LoadFrom(v, s); err != nil {
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
	for i := range ret {
		if !ret[i] {
			log.Println("missing state for module", s.items.ItemAt(i).Name(),
				", loading default")
			s.items.ItemAt(i).GetState().LoadFrom(nil, s)
		}
	}
}

// GenGroupStateYaml writes YAML output describing the state of the current
// module.
func (s *Store) GenGroupStateYaml() []byte {
	structure := make(yamlGroupState)
	for i := 0; i < s.items.NumItems(); i++ {
		item := s.items.ItemAt(i)
		structure[item.Name()] = item.GetState().ToYAML(s)
	}
	ret, _ := yaml.Marshal(structure)
	return ret
}
