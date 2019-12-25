/*
Package data implements loading and writing configuration and state data.
*/
package data

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"reflect"
	"sync"

	"github.com/flyx/pnpscreen/api"
	"github.com/flyx/pnpscreen/app"
)

// System describes a Pen & Paper system.
type System interface {
	Name() string
	// ID returns the unique ID of this system.
	// This is the name of the system's data directory.
	ID() string
}

type system struct {
	name    string
	id      string
	modules []interface{}
}

func (s *system) Name() string {
	return s.name
}

func (s *system) ID() string {
	return s.id
}

// implements api.Hero
type hero struct {
	name        string
	description string
}

func (h *hero) Name() string {
	return h.name
}

func (h *hero) Description() string {
	return h.description
}

type sceneModule struct {
	enabled bool
	config  interface{}
}

// Scene describes a collection of modules that will be rendered together.
type Scene interface {
	Name() string
	ID() string
	UsesModule(moduleIndex api.ModuleIndex) bool
}

type scene struct {
	name    string
	id      string
	modules []sceneModule
}

func (s *scene) Name() string {
	return s.name
}

func (s *scene) ID() string {
	return s.id
}

func (s *scene) UsesModule(moduleIndex api.ModuleIndex) bool {
	return s.modules[moduleIndex].enabled
}

type heroList struct {
	mutex sync.Mutex
	data  []hero
}

func (h *heroList) Hero(index int) api.Hero {
	return &h.data[index]
}

func (h *heroList) NumHeroes() int {
	return len(h.data)
}

func (h *heroList) Close() {
	h.mutex.Unlock()
}

// Group describes a Pen & Paper group / party
type Group interface {
	Name() string
	ID() string
	SystemIndex() int
	NumScenes() int
	Scene(index int) Scene
	ViewHeroes() api.HeroView
}

// group implements api.HeroList.
type group struct {
	name        string
	id          string
	systemIndex int
	modules     []interface{}
	heroes      heroList
	scenes      []scene
}

func (g *group) Name() string {
	return g.name
}

func (g *group) ID() string {
	return g.id
}

func (g *group) SystemIndex() int {
	return g.systemIndex
}

func (g *group) NumScenes() int {
	return len(g.scenes)
}

func (g *group) Scene(index int) Scene {
	return &g.scenes[index]
}

func (g *group) ViewHeroes() api.HeroView {
	g.heroes.mutex.Lock()
	return &g.heroes
}

// Config holds all configuration values of all modules.
//
// Configuration consists of four levels: The base level, the system level,
// the group level and the scene level. At each level, values for any
// configuration item may (but do not need to) be set. At runtime, the
// current configuration of each module is built by merging those four levels
// on top of the default values.
//
// The order of predescensce is: Scene level, Group level, system level,
// base level, default values.
// Configuration is rebuilt whenever the selected scene or group changes and
// whenever the configuration is edited (via web interface).
type Config struct {
	owner       app.App
	baseConfigs []interface{}
	systems     []*system
	groups      []*group
}

// NumSystems returns the number of available systems.
func (c *Config) NumSystems() int {
	return len(c.systems)
}

// System returns the system at the given index, which must be
// between 0 (included) and NumSystems() (excluded).
func (c *Config) System(index int) System {
	return c.systems[index]
}

// NumGroups returns the number of available groups.
func (c *Config) NumGroups() int {
	return len(c.groups)
}

// Group returns the group at the given index, which must be
// between 0 (included) and NumGroups() (excluded).
func (c *Config) Group(index int) Group {
	return c.groups[index]
}

func (c *Config) loadBase() []interface{} {
	path := c.owner.DataDir("base", "config.yaml")
	rawBaseConfig, err := ioutil.ReadFile(path)
	if err != nil {
		log.Println(path+":", err)
		return nil
	}
	ret, err := c.loadYamlBase(rawBaseConfig)
	if err != nil {
		log.Println(path+":", err)
	}
	return ret
}

func (c *Config) loadSystems() []*system {
	ret := make([]*system, 0, 16)
	systemsDir := c.owner.DataDir("systems")
	files, err := ioutil.ReadDir(systemsDir)
	if err == nil {
		for _, file := range files {
			if file.IsDir() {
				path := filepath.Join(systemsDir, file.Name(), "config.yaml")
				raw, err := ioutil.ReadFile(path)
				if err == nil {
					config, err := c.loadYamlSystem(file.Name(), raw)
					if err == nil {
						ret = append(ret, config)
					} else {
						log.Println(path+":", err)
					}
				} else {
					log.Println(path+":", err)
				}
			}
		}
	} else {
		log.Println(err)
	}
	return ret
}

func (c *Config) loadHeroes(groupPath string) []hero {
	ret := make([]hero, 0, 16)
	heroesDir := filepath.Join(groupPath, "heroes")
	files, err := ioutil.ReadDir(heroesDir)
	if err == nil {
		for _, file := range files {
			if file.IsDir() {
				path := filepath.Join(heroesDir, file.Name(), "config.yaml")
				raw, err := ioutil.ReadFile(path)
				var h hero
				if err == nil {
					h, err = c.loadYamlHero(file.Name(), raw)
				}
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

func (c *Config) loadScenes(groupPath string) []scene {
	ret := make([]scene, 0, 16)
	scenesDir := filepath.Join(groupPath, "scenes")
	files, err := ioutil.ReadDir(scenesDir)
	if err == nil {
		for _, file := range files {
			if file.IsDir() {
				path := filepath.Join(scenesDir, file.Name(), "config.yaml")
				raw, err := ioutil.ReadFile(path)
				var s scene
				if err == nil {
					s, err = c.loadYamlScene(file.Name(), raw)
				}
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

func (c *Config) loadGroups() []*group {
	ret := make([]*group, 0, 16)
	groupsDir := c.owner.DataDir("groups")
	files, err := ioutil.ReadDir(groupsDir)
	if err == nil {
		for _, file := range files {
			if file.IsDir() {
				path := filepath.Join(groupsDir, file.Name())
				configPath := filepath.Join(path, "config.yaml")
				raw, err := ioutil.ReadFile(configPath)
				if err == nil {
					g, err := c.loadYamlGroup(file.Name(), raw)
					if err != nil {
						log.Println(path+":", err)
					} else {
						g.heroes.data = c.loadHeroes(path)
						g.scenes = c.loadScenes(path)
						if len(g.scenes) == 0 {
							log.Println(path + ": no valid scenes available")
						} else {
							ret = append(ret, g)
						}
					}
				} else {
					log.Println(configPath+":", err)
				}
			}
		}
	} else {
		log.Println(err)
	}
	return ret
}

// Init loads all config.yaml files and parses them according to the module's
// config types. Parsing errors lead to a panic while structural errors are
// logged and ignored.
//
// You must call Init before doing anything with a Config value.
func (c *Config) Init(owner app.App) {
	c.owner = owner
	c.baseConfigs = c.loadBase()
	c.systems = c.loadSystems()
	c.groups = c.loadGroups()
}

func confValue(conf interface{}) *reflect.Value {
	val := reflect.ValueOf(conf).Elem()
	for ; val.Kind() == reflect.Interface ||
		val.Kind() == reflect.Ptr; val = val.Elem() {
	}
	if val.Kind() != reflect.Struct {
		panic("wrong kind of config value (not a struct, but a " +
			val.Kind().String() + ")")
	}
	return &val
}

// MergeConfig merges the item's default configuration with the values
// configured in its base config and the current system and group config.
// It returns the resulting configuration.
//
// systemIndex may be -1 (for groups without a defined system), groupIndex and
// sceneIndex may not.
func (c *Config) MergeConfig(
	moduleIndex api.ModuleIndex, systemIndex int, groupIndex int,
	sceneIndex int) interface{} {
	var configStack [5]*reflect.Value
	module := c.owner.ModuleAt(moduleIndex)

	defaultValues := module.Descriptor().DefaultConfig
	configType := reflect.TypeOf(defaultValues).Elem()

	{
		conf := c.groups[groupIndex].scenes[sceneIndex].modules[moduleIndex].config
		if conf == nil {
			configStack[0] = nil
		} else {
			configStack[0] = confValue(conf)
		}
	}
	{
		conf := c.groups[groupIndex].modules[moduleIndex]
		if conf == nil {
			panic("group config missing for " + module.Descriptor().ID)
		}
		configStack[1] = confValue(conf)
	}
	if systemIndex != -1 {
		conf := c.systems[systemIndex].modules[moduleIndex]
		if conf == nil {
			panic("system config missing for " + module.Descriptor().ID)
		}
		configStack[2] = confValue(conf)
	}

	baseConf := c.baseConfigs[moduleIndex]
	if baseConf == nil {
		panic("base config missing for " + module.Descriptor().ID)
	}
	baseValue := reflect.ValueOf(baseConf).Elem()
	configStack[3] = &baseValue

	defaultValue := reflect.ValueOf(defaultValues).Elem()
	configStack[4] = &defaultValue

	result := reflect.New(configType)
	for i := 0; i < configType.NumField(); i++ {
		for j := 0; j < 5; j++ {
			if configStack[j] != nil {
				field := configStack[j].Field(i)
				if !field.IsNil() {
					result.Elem().Field(i).Set(field)
					break
				}
			}
		}
	}
	return result.Interface()
}

// SendAsJSON sends the given data as JSON file
func SendAsJSON(w http.ResponseWriter, data interface{}) {
	b, err := json.Marshal(data)
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	} else {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
