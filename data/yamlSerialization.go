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

type persistedBaseConfig struct {
	Modules map[string]map[string]yaml.Node
}

type persistingBaseConfig struct {
	Modules map[string]map[string]interface{}
}

// persistedSystem is the structure system configuration
// is stored in YAML files. This differs from the internal
// structure as it uses module names and setting names as
// keys in YAML mappings rather than using a list which
// maps config items by position to a module / setting.
type persistedSystem struct {
	Name string
	// module name -> (setting name -> value)
	Modules map[string]map[string]yaml.Node
}

type persistingSystem struct {
	Name    string
	Modules map[string]map[string]interface{}
}

// persistedGroup is yamlSystem for groups
type persistedGroup struct {
	Name    string
	System  string
	Modules map[string]map[string]yaml.Node
}

type persistingGroup struct {
	Name    string
	System  string
	Modules map[string]map[string]interface{}
}

type yamlHero struct {
	Name        string
	Description string
}

type persistedSceneModule struct {
	Enabled bool
	Config  map[string]yaml.Node
}

type persistedScene struct {
	Name    string
	Modules map[string]persistedSceneModule
}

type persistingSceneModule struct {
	Enabled bool
	Config  interface{} // will be serialized to mapping since config must be a struct.
}

type persistingScene struct {
	Name    string
	Modules map[string]persistingSceneModule
}

func (c *Config) loadYamlModuleConfigInto(target interface{},
	values map[string]yaml.Node, moduleName string) bool {
	targetModule := reflect.ValueOf(target).Elem()
	targetModuleType := targetModule.Type()
	for i := 0; i < targetModuleType.NumField(); i++ {
		inValue, ok := values[targetModuleType.Field(i).Name]
		if !ok || inValue.Kind == yaml.ScalarNode && inValue.Tag == "!!null" {
			continue
		}
		wasNil := false
		if targetModule.Field(i).IsNil() {
			targetModule.Field(i).Set(reflect.New(targetModuleType.Field(i).Type.Elem()))
			wasNil = true
		}
		targetSetting := targetModule.Field(i).Interface().(api.ConfigItem)

		if err := targetSetting.LoadFrom(&inValue, c.owner, api.Persisted); err != nil {
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
	raw map[string]map[string]yaml.Node) ([]interface{}, error) {
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
				fields[fieldName] = fieldVal.Interface().(api.ConfigItem).SerializableView(c.owner, api.Persisted)
			}
		}
		if fields != nil {
			ret[c.owner.ModuleAt(i).Descriptor().ID] = fields
		}
	}
	return ret
}

func strictUnmarshalYAML(path string, target interface{}) error {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	return decoder.Decode(target)
}

func (c *Config) loadYamlBase(path string) ([]interface{}, error) {
	var data persistedBaseConfig
	if len(path) != 0 {
		if err := strictUnmarshalYAML(path, &data); err != nil {
			return make([]interface{}, c.owner.NumModules()), err
		}
	} else {
		data.Modules = make(map[string]map[string]yaml.Node)
	}

	return c.loadYamlModuleConfigs(data.Modules)
}

func (c *Config) writeYamlBase() {
	data := persistingBaseConfig{Modules: c.toYamlStructure(c.baseConfigs)}
	path := c.owner.DataDir("base", "config.yaml")
	raw, _ := yaml.Marshal(data)
	ioutil.WriteFile(path, raw, 0644)
}

func (c *Config) loadYamlSystem(
	id string, path string) (*system, error) {
	var data persistedSystem
	if err := strictUnmarshalYAML(path, &data); err != nil {
		return nil, err
	}
	moduleConfigs, err := c.loadYamlModuleConfigs(data.Modules)
	return &system{
		name:    data.Name,
		id:      id,
		modules: moduleConfigs}, err
}

func (c *Config) writeYamlSystem(value *system) error {
	data := persistingSystem{
		Name:    value.name,
		Modules: c.toYamlStructure(value.modules),
	}
	path := c.owner.DataDir("systems", value.id, "config.yaml")
	raw, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, raw, 0644)
}

func (c *Config) loadYamlGroup(
	id string, path string) (*group, error) {
	var data persistedGroup
	if err := strictUnmarshalYAML(path, &data); err != nil {
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

func (c *Config) writeYamlGroup(value *group) error {
	data := persistingGroup{
		Name:    value.name,
		System:  c.systems[value.systemIndex].id,
		Modules: c.toYamlStructure(value.modules),
	}
	path := c.owner.DataDir("groups", value.id, "config.yaml")
	raw, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, raw, 0644)
}

func (c *Config) loadYamlScene(id string, path string) (scene, error) {
	var data persistedScene
	if err := strictUnmarshalYAML(path, &data); err != nil {
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

func (c *Config) writeYamlScene(g *group, value *scene) error {
	data := persistingScene{
		Name: value.name, Modules: make(map[string]persistingSceneModule)}
	for i := api.ModuleIndex(0); i < c.owner.NumModules(); i++ {
		moduleDesc := c.owner.ModuleAt(i).Descriptor()
		moduleData := &value.modules[i]
		data.Modules[moduleDesc.ID] = persistingSceneModule{
			Enabled: moduleData.enabled, Config: moduleData.config}
	}
	path := c.owner.DataDir("groups", g.id, "scenes", value.id, "config.yaml")
	raw, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, raw, 0644)
}

func (c *Config) loadYamlHero(id string, path string) (hero, error) {
	var data yamlHero
	if err := strictUnmarshalYAML(path, &data); err != nil {
		return hero{}, err
	}
	return hero{name: data.Name, description: data.Description}, nil
}

type persistedGroupState struct {
	ActiveScene string `yaml:"activeScene"`
	// scene name -> (module name -> module config)
	Scenes map[string]map[string]yaml.Node
}

type persistingGroupState struct {
	ActiveScene string `yaml:"activeScene"`
	Scenes      map[string]map[string]interface{}
}

// LoadYamlGroupState loads the given YAML input into a GroupState object.
func LoadYamlGroupState(a app.App, g Group, path string) (*GroupState, error) {
	var data persistedGroupState
	if err := strictUnmarshalYAML(path, &data); err != nil {
		return nil, err
	}
	ret := &GroupState{
		activeScene: -1, scenes: make([][]api.ModuleState, g.NumScenes()),
		path: path, a: a, group: g}
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

							state, err := module.Descriptor().CreateState(&modRaw, a, j)
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
						state, err := module.Descriptor().CreateState(nil, a, j)
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
		}
		if !sceneFound {
			log.Printf("Unknown scene \"%s\"", sceneName)
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
					state, err := module.Descriptor().CreateState(nil, a, j)
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

func (gs *GroupState) buildYaml() ([]byte, error) {
	structure := persistingGroupState{
		ActiveScene: gs.group.Scene(gs.activeScene).ID(),
		Scenes:      make(map[string]map[string]interface{})}
	for i := 0; i < gs.group.NumScenes(); i++ {
		sceneDescr := gs.group.Scene(i)
		data := make(map[string]interface{})
		var j api.ModuleIndex
		for j = 0; j < gs.a.NumModules(); j++ {
			if sceneDescr.UsesModule(j) {
				data[gs.a.ModuleAt(j).Descriptor().ID] =
					gs.scenes[i][j].SerializableView(gs.a, api.Persisted)
			}
		}
		structure.Scenes[sceneDescr.ID()] = data
	}
	return yaml.Marshal(structure)
}
