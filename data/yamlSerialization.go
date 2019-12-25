package data

import (
	"bytes"
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

// yamlSystem is the structure system configuration
// is stored in YAML files. This differs from the internal
// structure as it uses module names and setting names as
// keys in YAML mappings rather than using a list which
// maps config items by position to a module / setting.
type yamlSystem struct {
	Name string
	// module name -> (setting name -> value)
	Modules map[string]map[string]interface{}
}

// yamlGroup is yamlSystem for groups
type yamlGroup struct {
	Name    string
	System  string
	Modules map[string]map[string]interface{}
}

type yamlHero struct {
	Name        string
	Description string
}

type yamlSceneModule struct {
	Enabled bool
	Config  map[string]interface{}
}

type yamlScene struct {
	Name    string
	Modules map[string]yamlSceneModule
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
		wasNil := false
		if targetModule.Field(i).IsNil() {
			targetModule.Field(i).Set(reflect.New(targetModuleType.Field(i).Type.Elem()))
			wasNil = true
		}
		targetSetting := targetModule.Field(i).Interface()

		if err := targetSetting.(api.ConfigItem).LoadFrom(inValue, c.owner, true); err != nil {
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
		if module.Descriptor().ID == id {
			return module, i
		}
	}
	return nil, -1
}

func configType(mod api.Module) reflect.Type {
	defaultType := reflect.TypeOf(mod.Descriptor().DefaultConfig)
	if defaultType.Kind() != reflect.Ptr ||
		defaultType.Elem().Kind() != reflect.Struct {
		panic("config type of module " + mod.Descriptor().ID +
			" is not pointer to struct!")
	}
	return defaultType.Elem()
}

func (c *Config) loadYamlModuleConfigs(
	raw map[string]map[string]interface{}) ([]interface{}, error) {
	ret := make([]interface{}, c.owner.NumModules())
	for name, rawItems := range raw {
		mod, index := findModule(c.owner, name)
		if mod == nil {
			return nil, fmt.Errorf("Unknown module \"%s\"", name)
		}

		target := reflect.New(configType(mod)).Interface()
		if c.loadYamlModuleConfigInto(target, rawItems, mod.Descriptor().ID) {
			ret[index] = target
		}
	}
	var i api.ModuleIndex
	for i = 0; i < c.owner.NumModules(); i++ {
		if ret[i] == nil {
			mod := c.owner.ModuleAt(i)
			ret[i] = reflect.New(configType(mod)).Interface()
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
			ret[c.owner.ModuleAt(i).Descriptor().ID] = fields
		}
	}
	return ret
}

func strictUnmarshalYAML(yamlInput []byte, target interface{}) error {
	decoder := yaml.NewDecoder(bytes.NewReader(yamlInput))
	decoder.KnownFields(true)
	return decoder.Decode(target)
}

func (c *Config) loadYamlBase(yamlInput []byte) ([]interface{}, error) {
	var data yamlBaseConfig
	if yamlInput != nil {
		if err := strictUnmarshalYAML(yamlInput, &data); err != nil {
			return make([]interface{}, c.owner.NumModules()), err
		}
	} else {
		data.Modules = make(map[string]map[string]interface{})
	}

	return c.loadYamlModuleConfigs(data.Modules)
}

func (c *Config) writeYamlBase() {
	data := yamlBaseConfig{Modules: c.toYamlStructure(c.baseConfigs)}
	path := c.owner.DataDir("base", "config.yaml")
	raw, _ := yaml.Marshal(data)
	ioutil.WriteFile(path, raw, 0644)
}

func (c *Config) loadYamlSystem(
	id string, yamlInput []byte) (*system, error) {
	var data yamlSystem
	if err := strictUnmarshalYAML(yamlInput, &data); err != nil {
		return nil, err
	}
	moduleConfigs, err := c.loadYamlModuleConfigs(data.Modules)
	return &system{
		name:    data.Name,
		id:      id,
		modules: moduleConfigs}, err
}

func (c *Config) writeYamlSystem(value *system) {
	data := yamlSystem{
		Name:    value.name,
		Modules: c.toYamlStructure(value.modules),
	}
	path := c.owner.DataDir("systems", value.id, "config.yaml")
	raw, _ := yaml.Marshal(data)
	ioutil.WriteFile(path, raw, 0644)
}

func (c *Config) loadYamlGroup(
	id string, yamlInput []byte) (*group, error) {
	var data yamlGroup
	if err := strictUnmarshalYAML(yamlInput, &data); err != nil {
		return nil, err
	}
	for i := 0; i < len(c.systems); i++ {
		if c.systems[i].id == data.System {
			moduleConfigs, err := c.loadYamlModuleConfigs(data.Modules)
			return &group{
				name:        data.Name,
				id:          id,
				systemIndex: i,
				modules:     moduleConfigs,
			}, err
		}
	}
	return nil,
		fmt.Errorf("Group config references unknown system \"%s\"", data.System)
}

func (c *Config) writeYamlGroup(value *group) {
	data := yamlGroup{
		Name:    value.name,
		System:  c.systems[value.systemIndex].id,
		Modules: c.toYamlStructure(value.modules),
	}
	path := c.owner.DataDir("groups", value.id, "config.yaml")
	raw, _ := yaml.Marshal(data)
	ioutil.WriteFile(path, raw, 0644)
}

func (c *Config) loadYamlHero(id string, yamlInput []byte) (hero, error) {
	var data yamlHero
	if err := strictUnmarshalYAML(yamlInput, &data); err != nil {
		return hero{}, err
	}
	return hero{name: data.Name, description: data.Description}, nil
}

func (c *Config) loadYamlScene(id string, yamlInput []byte) (scene, error) {
	var data yamlScene
	if err := strictUnmarshalYAML(yamlInput, &data); err != nil {
		return scene{}, err
	}
	ret := scene{name: data.Name, id: id,
		modules: make([]sceneModule, c.owner.NumModules())}
	for name, value := range data.Modules {
		mod, index := findModule(c.owner, name)
		if mod == nil {
			return scene{}, fmt.Errorf("Unknown module \"%s\"", name)
		}
		target := reflect.New(configType(mod)).Interface()
		if c.loadYamlModuleConfigInto(target, value.Config, mod.Descriptor().ID) {
			ret.modules[index] = sceneModule{enabled: value.Enabled, config: target}
		}
	}
	return ret, nil
}

type yamlGroupState struct {
	ActiveScene string `yaml:"activeScene"`
	// scene name -> (module name -> module config)
	Scenes map[string]map[string]interface{}
}

// LoadYamlGroupState loads the given YAML input into a GroupState object.
func LoadYamlGroupState(a app.App, g Group, yamlInput []byte) (GroupState, error) {
	var data yamlGroupState
	if err := strictUnmarshalYAML(yamlInput, &data); err != nil {
		return GroupState{}, err
	}
	ret := GroupState{activeScene: -1, scenes: make([][]api.ModuleState, g.NumScenes())}
	for i := 0; i < g.NumScenes(); i++ {
		if g.Scene(i).ID() == data.ActiveScene {
			ret.activeScene = i
			break
		}
	}
	if ret.activeScene == -1 {
		ret.activeScene = 0
		log.Printf("Unknown active scene for group %s: \"%s\"\n",
			g.Name(), data.ActiveScene)
	}
	sceneLoaded := make([]bool, g.NumScenes())
	for sceneName, sceneValue := range data.Scenes {
		sceneFound := false
		for i := 0; i < g.NumScenes(); i++ {
			sceneDescr := g.Scene(i)
			if sceneName == sceneDescr.ID() {
				sceneFound = true
				sceneData := make([]api.ModuleState, a.NumModules())
				moduleLoaded := make([]bool, a.NumModules())
				var j api.ModuleIndex
				for modName, modRaw := range sceneValue {
					moduleFound := false
					for j = 0; j < a.NumModules(); j++ {
						module := a.ModuleAt(j)
						if modName == module.Descriptor().ID {
							moduleFound = true
							if !sceneDescr.UsesModule(j) {
								log.Printf("Scene \"%s\": Data given for module %s"+
									" not used in the scene\n", sceneName, modName)
								break
							}

							state, err := module.CreateState(modRaw, a)
							if err != nil {
								log.Printf(
									"Scene \"%s\": Could not load state for module %s: %s\n",
									sceneName, modName, err.Error())
								break
							}
							sceneData[j] = state
							moduleLoaded[j] = true
						}
					}
					if !moduleFound {
						log.Printf("Scene \"%s\": Unknown module \"%s\"\n",
							sceneName, modName)
					}
				}
				for j = 0; j < a.NumModules(); j++ {
					if sceneDescr.UsesModule(j) && !moduleLoaded[j] {
						module := a.ModuleAt(j)
						log.Printf(
							"Scene \"%s\": Missing data for module %s, loading default\n",
							sceneName, module.Descriptor().ID)
						state, err := module.CreateState(nil, a)
						if err != nil {
							panic("Failed to create state with default values for module " +
								module.Descriptor().ID)
						}
						sceneData[j] = state
					}
				}
				ret.scenes[i] = sceneData
				sceneLoaded[i] = true
				break
			}
			if !sceneFound {
				log.Printf("Unknown scene \"%s\"", sceneName)
			}
		}
	}
	for i := 0; i < g.NumScenes(); i++ {
		if !sceneLoaded[i] {
			sceneDescr := g.Scene(i)
			log.Printf("Missing data for scene \"%s\", loading default\n",
				sceneDescr.ID())
			sceneData := make([]api.ModuleState, a.NumModules())
			var j api.ModuleIndex
			for j = 0; j < a.NumModules(); j++ {
				if sceneDescr.UsesModule(j) {
					module := a.ModuleAt(j)
					state, err := module.CreateState(nil, a)
					if err != nil {
						panic("Failed to create state with default values for module " +
							module.Descriptor().ID)
					}
					sceneData[j] = state
				}
			}
			ret.scenes[i] = sceneData
		}
	}
	return ret, nil
}

// BuildYaml writes YAML output describing the GroupState.
func (gs *GroupState) BuildYaml(a app.App, g Group) ([]byte, error) {
	structure := yamlGroupState{
		ActiveScene: g.Scene(gs.activeScene).ID(),
		Scenes:      make(map[string]map[string]interface{})}
	for i := 0; i < g.NumScenes(); i++ {
		sceneDescr := g.Scene(i)
		data := make(map[string]interface{})
		var j api.ModuleIndex
		for j = 0; j < a.NumModules(); j++ {
			if sceneDescr.UsesModule(j) {
				data[a.ModuleAt(j).Descriptor().ID] = gs.scenes[i][j].ToYAML(a)
			}
		}
		structure.Scenes[sceneDescr.ID()] = data
	}
	return yaml.Marshal(structure)
}
