package data

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/flyx/pnpscreen/api"
	"github.com/flyx/pnpscreen/app"
	"gopkg.in/yaml.v3"
)

// Persistence implements writing data to and loading data from the file system
type Persistence struct {
	d *Data
}

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

func (p Persistence) loadModuleConfigInto(target interface{},
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

		if err := targetSetting.LoadPersisted(&inValue, p.d.owner); err != nil {
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

func (p Persistence) loadModuleConfigs(
	raw map[string]map[string]yaml.Node) ([]interface{}, error) {
	ret := make([]interface{}, p.d.owner.NumModules())
	for name, rawItems := range raw {
		mod, index := findModule(p.d.owner, name)
		if mod == nil {
			return nil, fmt.Errorf("Unknown module \"%s\"", name)
		}

		target := reflect.New(configType(mod)).Interface()
		if p.loadModuleConfigInto(target, rawItems, mod.Descriptor().ID) {
			ret[index] = target
		}
	}
	var i api.ModuleIndex
	for i = 0; i < p.d.owner.NumModules(); i++ {
		if ret[i] == nil {
			mod := p.d.owner.ModuleAt(i)
			ret[i] = reflect.New(configType(mod)).Interface()
		}
	}
	return ret, nil
}

func (p Persistence) persistingModuleConfigs(moduleConfigs []interface{}) map[string]map[string]interface{} {
	ret := make(map[string]map[string]interface{})
	var i api.ModuleIndex
	for i = 0; i < p.d.owner.NumModules(); i++ {
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
				fields[fieldName] =
					fieldVal.Interface().(api.ConfigItem).PersistingView(p.d.owner)
			}
		}
		if fields != nil {
			ret[p.d.owner.ModuleAt(i).Descriptor().ID] = fields
		}
	}
	return ret
}

func strictUnmarshalYAML(input inputProvider, target interface{}) error {
	raw, err := input()
	if err != nil {
		return err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	return decoder.Decode(target)
}

func (p Persistence) loadBase(path string) ([]interface{}, error) {
	var data persistedBaseConfig
	if len(path) != 0 {
		if err := strictUnmarshalYAML(fileInput(path), &data); err != nil {
			return make([]interface{}, p.d.owner.NumModules()), err
		}
	} else {
		data.Modules = make(map[string]map[string]yaml.Node)
	}

	return p.loadModuleConfigs(data.Modules)
}

// WriteBase writes the current base configuration to the file system.
func (p Persistence) WriteBase() error {
	data := persistingBaseConfig{Modules: p.persistingModuleConfigs(p.d.baseConfigs)}
	dirPath := p.d.owner.DataDir("base")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return err
	}
	path := filepath.Join(dirPath, "config.yaml")
	raw, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, raw, 0644)
}

func (p Persistence) loadSystem(
	id string, input inputProvider) (*system, error) {
	var data persistedSystem
	if err := strictUnmarshalYAML(input, &data); err != nil {
		return nil, err
	}
	moduleConfigs, err := p.loadModuleConfigs(data.Modules)
	return &system{
		name:    data.Name,
		id:      id,
		modules: moduleConfigs}, err
}

// WriteSystem writes the given system to the file system.
func (p Persistence) WriteSystem(s System) error {
	value := s.(*system)
	data := persistingSystem{
		Name:    value.name,
		Modules: p.persistingModuleConfigs(value.modules),
	}
	dirPath := p.d.owner.DataDir("systems", value.id)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return err
	}
	path := filepath.Join(dirPath, "config.yaml")
	raw, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, raw, 0644)
}

func (p Persistence) createSystem(tmpl *api.SystemTemplate) (*system, error) {
	s, err := p.loadSystem(tmpl.ID, byteInput(tmpl.Config))
	if err != nil {
		return nil, err
	}
	if err = p.WriteSystem(s); err != nil {
		return nil, err
	}
	return s, nil
}

// CreateSystem creates a new system with the given name.
func (p Persistence) CreateSystem(name string) api.SendableError {
	base := normalize(name)
	id := base
	num := 0
idCheckLoop:
	for {
		for i := range p.d.systems {
			if p.d.systems[i].id == id {
				num++
				id = base + strconv.Itoa(num)
				continue idCheckLoop
			}
		}
		break
	}
	s := &system{name: name, id: id, modules: make([]interface{}, p.d.owner.NumModules())}

	for i := api.ModuleIndex(0); i < p.d.owner.NumModules(); i++ {
		mod := p.d.owner.ModuleAt(i)
		s.modules[i] = reflect.New(configType(mod)).Interface()
	}

	if err := p.WriteSystem(s); err != nil {
		return &api.InternalError{
			Description: "failed to write system", Inner: err}
	}
	p.d.systems = append(p.d.systems, s)
	insertSorted(systemSortInterface{p.d.systems, p.d.numPluginSystems})
	return nil
}

// DeleteSystem deletes the system with the given ID.
//
// Groups linked to this system will have that link removed.
func (p Persistence) DeleteSystem(index int) api.SendableError {
	s := p.d.systems[index]
	if index < p.d.numPluginSystems {
		return &api.BadRequest{
			Message: "cannot delete plugin-provided system " + s.id}
	}
	for j := range p.d.groups {
		group := p.d.groups[j]
		if group.systemIndex == index {
			group.systemIndex = -1
		} else if group.systemIndex > index {
			group.systemIndex--
		} else {
			continue
		}
		if err := p.writeGroup(group); err != nil {
			log.Printf("[del system] while updating group %s:\n  %s\n",
				group.id, err.Error())
		}
	}
	path := p.d.owner.DataDir("systems", s.id)
	if err := os.RemoveAll(path); err != nil {
		log.Printf("[del system] while deleting %s:\n  %s\n", path, err.Error())
	}
	copy(p.d.systems[index:], p.d.systems[index+1:])
	p.d.systems[len(p.d.systems)-1] = nil
	p.d.systems = p.d.systems[:len(p.d.systems)-1]
	return nil
}

func (p Persistence) loadGroup(id string, input inputProvider) (*group, error) {
	var data persistedGroup
	if err := strictUnmarshalYAML(input, &data); err != nil {
		return nil, err
	}
	systemIndex := -1
	if data.System != "" {
		for i := 0; i < len(p.d.systems); i++ {
			if p.d.systems[i].id == data.System {
				systemIndex = i
				break
			}
		}
		if systemIndex == -1 {
			return nil,
				fmt.Errorf("Group config references unknown system \"%s\"", data.System)
		}
	}
	moduleConfigs, err := p.loadModuleConfigs(data.Modules)
	return &group{
		name:        data.Name,
		id:          id,
		systemIndex: systemIndex,
		modules:     moduleConfigs,
	}, err
}

func (p Persistence) writeGroup(value *group) error {
	data := persistingGroup{
		Name:    value.name,
		Modules: p.persistingModuleConfigs(value.modules),
	}
	if value.systemIndex != -1 {
		data.System = p.d.systems[value.systemIndex].id
	}
	dirPath := p.d.owner.DataDir("groups", value.id)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return err
	}
	path := filepath.Join(dirPath, "config.yaml")
	raw, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, raw, 0644)
}

// WriteGroup writes the group config to the file system.
func (p Persistence) WriteGroup(g Group) error {
	return p.writeGroup(g.(*group))
}

// CreateGroup creates a new group with the given name, creating an alphanumeric
// ID from the name. It appends the group to the list of groups.
// The group's initial configuration is given via a GroupTemplate, which must
// not be nil.
func (p Persistence) CreateGroup(
	name string, tmpl *api.GroupTemplate, sceneTmpls []api.SceneTemplate) error {
	if tmpl == nil {
		return errors.New("missing group template")
	}
	base := strings.ToLower(normalize(name))
	id := base
	num := 0
	if base == "" {
		base = "group"
		id = "group1"
		num++
	}
idCheckLoop:
	for {
		for i := range p.d.groups {
			if p.d.groups[i].id == id {
				num++
				id = base + strconv.Itoa(num)
				continue idCheckLoop
			}
		}
		break
	}
	g, err := p.loadGroup(id, byteInput(tmpl.Config))
	if err != nil {
		return errors.New("could not load group config template:\n  " + err.Error())
	}
	if err = os.MkdirAll(p.d.owner.DataDir("groups", id, "scenes"), 0755); err != nil {
		return err
	}
	g.name = name
	g.scenes = make([]scene, 0, 16)
	if err = p.writeGroup(g); err != nil {
		os.RemoveAll(p.d.owner.DataDir("groups", id))
		return err
	}
	for i := range tmpl.Scenes {
		if err := p.CreateScene(
			g, tmpl.Scenes[i].Name, &sceneTmpls[tmpl.Scenes[i].TmplIndex]); err != nil {
			os.RemoveAll(p.d.owner.DataDir("groups", id))
			return err
		}
	}
	p.d.groups = append(p.d.groups, g)
	insertSorted(groupSortInterface{p.d.groups})
	return nil
}

// DeleteGroup deletes the group with the given ID.
func (p Persistence) DeleteGroup(index int) {
	g := p.d.groups[index]
	path := p.d.owner.DataDir("groups", g.id)
	if err := os.RemoveAll(path); err != nil {
		log.Printf("[del group] while deleting %s\n  %s\n", path, err.Error())
	}
	copy(p.d.groups[index:], p.d.groups[index+1:])
	p.d.groups[len(p.d.groups)-1] = nil
	p.d.groups = p.d.groups[:len(p.d.groups)-1]
}

func (p Persistence) loadScene(id string, input inputProvider) (scene, error) {
	var data persistedScene
	if err := strictUnmarshalYAML(input, &data); err != nil {
		return scene{}, err
	}
	ret := scene{name: data.Name, id: id,
		modules: make([]sceneModule, p.d.owner.NumModules())}
	for name, value := range data.Modules {
		mod, index := findModule(p.d.owner, name)
		if mod == nil {
			return scene{}, fmt.Errorf("Unknown module \"%s\"", name)
		}
		target := reflect.New(configType(mod)).Interface()
		if p.loadModuleConfigInto(target, value.Config, mod.Descriptor().ID) {
			ret.modules[index] = sceneModule{enabled: value.Enabled, config: target}
		}
	}
	return ret, nil
}

func (p Persistence) writeScene(g *group, value *scene) error {
	data := persistingScene{
		Name: value.name, Modules: make(map[string]persistingSceneModule)}
	for i := api.ModuleIndex(0); i < p.d.owner.NumModules(); i++ {
		moduleDesc := p.d.owner.ModuleAt(i).Descriptor()
		moduleData := &value.modules[i]
		data.Modules[moduleDesc.ID] = persistingSceneModule{
			Enabled: moduleData.enabled, Config: moduleData.config}
	}
	dirPath := p.d.owner.DataDir("groups", g.id, "scenes", value.id)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return err
	}
	path := filepath.Join(dirPath, "config.yaml")
	raw, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, raw, 0644)
}

// WriteScene writes the given scene of the given group to the file system.
func (p Persistence) WriteScene(g Group, s Scene) error {
	return p.writeScene(g.(*group), s.(*scene))
}

// CreateScene creates a new scene from the given template in the given group.
func (p Persistence) CreateScene(g Group, name string, tmpl *api.SceneTemplate) error {
	base := strings.ToLower(normalize(name))
	id := base
	num := 0
	gr := g.(*group)
idCheckLoop:
	for {
		for i := range gr.scenes {
			if gr.scenes[i].id == id {
				num++
				id = base + strconv.Itoa(num)
				continue idCheckLoop
			}
		}
		break
	}
	s, err := p.loadScene(id, byteInput(tmpl.Config))
	if err != nil {
		return err
	}
	s.name = name
	if err = p.writeScene(gr, &s); err != nil {
		return err
	}
	gr.scenes = append(gr.scenes, s)
	return nil
}

// DeleteScene deletes the scene with the given id from the given group.
func (p Persistence) DeleteScene(g Group, index int) error {
	gr := g.(*group)
	if index < 0 || index >= g.NumScenes() {
		return errors.New("index out of range")
	}

	path := p.d.owner.DataDir("groups", gr.id, "scenes", gr.scenes[index].id)
	if err := os.RemoveAll(path); err != nil {
		log.Printf("[del scene] while deleting %s\n  %s\n", path, err.Error())
	}
	copy(gr.scenes[index:], gr.scenes[index+1:])
	gr.scenes[len(gr.scenes)-1].modules = nil
	gr.scenes = gr.scenes[:len(gr.scenes)-1]
	return nil
}

func (p Persistence) loadHero(id string, path string) (hero, error) {
	var data yamlHero
	if err := strictUnmarshalYAML(fileInput(path), &data); err != nil {
		return hero{}, err
	}
	return hero{name: data.Name, id: id, description: data.Description}, nil
}

func (p Persistence) loadSystems() {
	unsorted := make([]*system, 0, 16)
	systemsDir := p.d.owner.DataDir("systems")
	files, err := ioutil.ReadDir(systemsDir)
	if err == nil {
		for _, file := range files {
			if file.IsDir() {
				path := filepath.Join(systemsDir, file.Name(), "config.yaml")
				config, err := p.loadSystem(file.Name(), fileInput(path))
				if err == nil {
					unsorted = append(unsorted, config)
				} else {
					log.Println(path+":", err)
				}
			}
		}
	} else {
		log.Println(err)
	}
	// sort systems: first come all systems required by plugins, then
	// all other systems.

	p.d.systems = make([]*system, 0, len(unsorted)+p.d.owner.NumPlugins())
	for i := 0; i < p.d.owner.NumPlugins(); i++ {
		plugin := p.d.owner.Plugin(i)
		for j := range plugin.SystemTemplates {
			found := false
			for k := range unsorted {
				if unsorted[k].id == plugin.SystemTemplates[j].ID {
					p.d.systems = append(p.d.systems, unsorted[k])
					unsorted[k] = nil
					found = true
					break
				}
			}
			if !found {
				log.Printf(
					"creating system %s which is missing but required by plugin %s\n",
					plugin.SystemTemplates[j].ID, plugin.Name)
				s, err := p.createSystem(&plugin.SystemTemplates[j])
				if err != nil {
					log.Println("  failed to create system: " + err.Error())
				} else {
					p.d.systems = append(p.d.systems, s)
				}
			}
		}
	}
	sort.Sort(systemSortInterface{p.d.systems, 0})

	p.d.numPluginSystems = len(p.d.systems)
	for i := range unsorted {
		if unsorted[i] != nil {
			p.d.systems = append(p.d.systems, unsorted[i])
		}
	}
	sort.Sort(systemSortInterface{p.d.systems, p.d.numPluginSystems})
}

func (p Persistence) loadGroups() {
	p.d.groups = make([]*group, 0, 16)
	groupsDir := p.d.owner.DataDir("groups")
	files, err := ioutil.ReadDir(groupsDir)
	if err != nil {
		log.Println("while loading groups: " + err.Error())
		return
	}

	for _, file := range files {
		if file.IsDir() {
			path := filepath.Join(groupsDir, file.Name())
			configPath := filepath.Join(path, "config.yaml")
			g, err := p.loadGroup(file.Name(), fileInput(configPath))
			if err != nil {
				log.Println(configPath+":", err)
			} else {
				g.heroes.data = p.loadHeroes(path)
				g.scenes = p.loadScenes(path)
				if len(g.scenes) == 0 {
					log.Println(path + ": no valid scenes available")
				} else {
					p.d.groups = append(p.d.groups, g)
				}
			}
		}
	}

	sort.Sort(groupSortInterface{p.d.groups})
}

func (p Persistence) loadScenes(groupPath string) []scene {
	ret := make([]scene, 0, 16)
	scenesDir := filepath.Join(groupPath, "scenes")
	files, err := ioutil.ReadDir(scenesDir)
	if err == nil {
		for _, file := range files {
			if file.IsDir() {
				path := filepath.Join(scenesDir, file.Name(), "config.yaml")
				var s scene
				s, err = p.loadScene(file.Name(), fileInput(path))
				if err == nil {
					ret = append(ret, s)
				} else {
					log.Println(path+":", err)
				}
			}
		}
	}
	return ret
}

func (p Persistence) loadHeroes(groupPath string) []hero {
	ret := make([]hero, 0, 16)
	heroesDir := filepath.Join(groupPath, "heroes")
	files, err := ioutil.ReadDir(heroesDir)
	if err == nil {
		for _, file := range files {
			if file.IsDir() {
				path := filepath.Join(heroesDir, file.Name(), "config.yaml")
				var h hero
				h, err = p.loadHero(file.Name(), path)
				if err == nil {
					ret = append(ret, h)
				} else {
					log.Println(path+":", err)
				}
			}
		}
	}
	return ret
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

// LoadState loads the given YAML input into a State object and stores that
// into the linked data object.
func (p Persistence) LoadState(a app.App, g Group, path string) (*State, error) {
	var data persistedGroupState
	if err := strictUnmarshalYAML(fileInput(path), &data); err != nil {
		log.Println(path + ": unable to load, loading default. error was:")
		log.Println("  " + err.Error())
		data.ActiveScene = g.Scene(0).ID()
	}
	p.d.State.activeScene = -1
	p.d.State.scenes = make([][]api.ModuleState, g.NumScenes())
	p.d.State.path = path
	p.d.State.a = a
	p.d.State.group = g
	for i := 0; i < g.NumScenes(); i++ {
		if g.Scene(i).ID() == data.ActiveScene {
			p.d.activeScene = i
			break
		}
	}
	if p.d.activeScene == -1 {
		p.d.activeScene = 0
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
				p.d.State.scenes[i] = sceneData
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
			p.d.State.scenes[i] = sceneData
		}
	}
	return &p.d.State, nil
}

func (s *State) buildYaml() ([]byte, error) {
	structure := persistingGroupState{
		ActiveScene: s.group.Scene(s.activeScene).ID(),
		Scenes:      make(map[string]map[string]interface{})}
	for i := 0; i < s.group.NumScenes(); i++ {
		sceneDescr := s.group.Scene(i)
		data := make(map[string]interface{})
		var j api.ModuleIndex
		for j = 0; j < s.a.NumModules(); j++ {
			if sceneDescr.UsesModule(j) {
				data[s.a.ModuleAt(j).Descriptor().ID] =
					s.scenes[i][j].PersistingView(s.a)
			}
		}
		structure.Scenes[sceneDescr.ID()] = data
	}
	return yaml.Marshal(structure)
}

// WriteState writes the group state to its YAML file.
// The actual writing operation is done asynchronous.
func (p Persistence) WriteState() {
	raw, err := p.d.State.buildYaml()
	if err != nil {
		log.Printf("%s[w]: %s", p.d.State.path, err)
	} else {
		go func(content []byte, s *State) {
			s.writeMutex.Lock()
			defer s.writeMutex.Unlock()
			ioutil.WriteFile(s.path, content, 0644)
		}(raw, &p.d.State)
	}
}
