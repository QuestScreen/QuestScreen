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

	"gopkg.in/yaml.v3"
)

type configuredModuleState int32

const (
	moduleDisabled configuredModuleState = iota
	moduleEnabled
	moduleInherited
)

type moduleConfig struct {
	State  configuredModuleState
	Config interface{}
}

type systemConfig struct {
	Name    string
	DirName string `yaml:"-"`
	Modules map[string]*moduleConfig
}

type groupConfig struct {
	Name        string
	DirName     string `yaml:"-"`
	System      string
	SystemIndex int `yaml:"-"`
	Modules     map[string]*moduleConfig
}

type hero struct {
	Name        string
	Description string
	Group       string
}

type group struct {
	Config groupConfig
	Heroes []hero
}

// Config holds all configuration values of all modules.
//
// Configuration consists of three levels: The base level, the system level and
// the group level. At each level, values for any configuration item may be (but
// do not need to) be set. At runtime, the current configuration of each module
// is built by merging those three levels with the default values.
//
// The order of predescensce is: Group level, system level, base level, default
// values. Group and system levels only exist if a group or system is selected.
// Configuration is rebuilt whenever the selected group or system changes and
// whenever the configuration is edited (via web interface).
type Config struct {
	items       ConfigurableItemProvider
	baseConfigs map[string]*moduleConfig
	systems     []systemConfig
	groups      []group
}

// ConfigurableItem describes an item that may be configured via a Config object
type ConfigurableItem interface {
	// Name gives the name of this item.
	Name() string
	// Alphanumeric name used for:
	// * directories with module data
	// * HTTP setter endpoints
	// * IDs for menu setters
	// May not contain whitespace or special characters. Must be unique among loaded modules.
	InternalName() string
	// returns an empty configuration
	// The item defines the type of its configuration.
	EmptyConfig() interface{}
	// returns a configuration object with default values.
	// The configuration object must be a pointer and must
	DefaultConfig() interface{}
	// SetConfig sets current configuration for the item.
	SetConfig(config interface{})
	// GetConfig retrieves the current configuration of the item.
	GetConfig() interface{}
}

// ConfigurableItemProvider is basically a list of ConfigurableItem.
// This interface exists because you cannot cast an array of anything derived
// from ConfigurableItem to an array of ConfigurableItem.
type ConfigurableItemProvider interface {
	NumItems() int
	ItemAt(index int) ConfigurableItem
}

// NumSystems returns the number of available systems.
func (c *Config) NumSystems() int {
	return len(c.systems)
}

// SystemName returns the name of the system at the given index, which must be
// between 0 (included) and NumSystems() (excluded).
func (c *Config) SystemName(index int) string {
	return c.systems[index].Name
}

// SystemDirectory returns the name of the directory for the system at the given
// index, which must be between 0 (included) and NumSystems() (excluded).
// It is useful for retrieving data residing in the system's directory.
func (c *Config) SystemDirectory(index int) string {
	return c.systems[index].DirName
}

// NumGroups returns the number of available groups.
func (c *Config) NumGroups() int {
	return len(c.groups)
}

// GroupName returns the name of the group at the given index, which must be
// between 0 (included) and NumGroups() (excluded).
func (c *Config) GroupName(index int) string {
	return c.groups[index].Config.Name
}

// GroupDirectory returns the name of the directory for the group at the given
// index, which must be between 0 (included) and NumGroups() (excluded).
// It is useful for retrieving data residing in the group's directory.
func (c *Config) GroupDirectory(index int) string {
	return c.groups[index].Config.DirName
}

// GroupLinkedSystem returns the index of the system the group at the given
// index is linked to. The given index must be between 0 (included) and
// NumGroups() (excluded). If the group is not linked to a system,
// GroupLinkedSystem returns -1.
func (c *Config) GroupLinkedSystem(index int) int {
	return c.groups[index].Config.SystemIndex
}

// NumHeroes returns the number of heroes in the group of the given index, which
// must be between 0 (included) and NumGroups() (excluded).
func (c *Config) NumHeroes(groupIndex int) int {
	return len(c.groups[groupIndex].Heroes)
}

// HeroName returns the name of the hero at
// 0 <= heroIndex < NumHeroes(groupIndex) in the group at
// 0 <= groupIndex < NumGroups().
func (c *Config) HeroName(groupIndex int, heroIndex int) string {
	return c.groups[groupIndex].Heroes[heroIndex].Name
}

// HeroDescription returns the description of the hero at
// 0 <= heroIndex < NumHeroes(groupIndex) in the group at
// 0 <= groupIndex < NumGroups().
func (c *Config) HeroDescription(groupIndex int, heroIndex int) string {
	return c.groups[groupIndex].Heroes[heroIndex].Description
}

func findItem(items ConfigurableItemProvider, name string) (ConfigurableItem, int) {
	for i := 0; i < items.NumItems(); i++ {
		if items.ItemAt(i).Name() == name {
			return items.ItemAt(i), i
		}
	}
	return nil, -1
}

// necessary to re-use the JSON deserialization funcs for loading YAML
type dummyWriter struct {
	headers http.Header
}

func createDummyWriter() *dummyWriter {
	headers := make(http.Header)
	return &dummyWriter{headers: headers}
}

func (d *dummyWriter) Header() http.Header {
	return d.headers
}

func (d *dummyWriter) Write(data []byte) (int, error) {
	log.Println(data)
	return 0, nil
}

func (d *dummyWriter) WriteHeader(statusCode int) {}

func (s *StaticData) constructModuleConfigs(data map[string]*moduleConfig,
	raw map[string]*moduleConfig, items ConfigurableItemProvider) {
	foundModules := make([]bool, items.NumItems())
	dummy := createDummyWriter()
	for name, node := range raw {
		mod, index := findItem(items, name)
		if mod == nil {
			log.Println("Unknown module: " + name)
		} else {
			items, ok := node.Config.(map[string]interface{})
			if ok {
				target := mod.EmptyConfig()
				if s.loadModuleConfigInto(target, items, name, dummy) {
					foundModules[index] = true
					data[name] = &moduleConfig{
						State: node.State, Config: target}
				}
			} else {
				log.Println("Value of module " + name + " is not a mapping!")
			}
		}
	}
	for i := 0; i < items.NumItems(); i++ {
		if !foundModules[i] {
			mod := items.ItemAt(i)
			data[mod.Name()] = &moduleConfig{
				State: moduleInherited, Config: mod.EmptyConfig(),
			}
		}
	}
}

// Init loads all config.yaml files and parses them according to the module's
// config types. Parsing errors lead to a panic while structural errors are
// logged and ignored.
//
// You must call Init before doing anything with a Config value.
func (c *Config) Init(static *StaticData, items ConfigurableItemProvider) {
	c.items = items
	c.groups = make([]group, 0, 16)

	rawBaseConfig, err := ioutil.ReadFile(filepath.Join(static.DataDir, "base", "config.yaml"))
	if err != nil {
		panic(err)
	}
	var nodes map[string]*moduleConfig
	err = yaml.Unmarshal(rawBaseConfig, &nodes)
	if err != nil {
		panic(err)
	}
	c.baseConfigs = make(map[string]*moduleConfig)
	static.constructModuleConfigs(c.baseConfigs, nodes, items)

	systemsDir := filepath.Join(static.DataDir, "systems")
	files, err := ioutil.ReadDir(systemsDir)
	if err == nil {
		for _, file := range files {
			if file.IsDir() {
				config, err := ioutil.ReadFile(filepath.Join(systemsDir, file.Name(), "config.yaml"))
				if err == nil {
					var tmp systemConfig
					err = yaml.Unmarshal(config, &tmp)
					if err == nil {
						finalConfig := systemConfig{
							Name: tmp.Name, DirName: file.Name(),
							Modules: make(map[string]*moduleConfig)}
						static.constructModuleConfigs(finalConfig.Modules, tmp.Modules, items)
						c.systems = append(c.systems, finalConfig)
					}
				} else {
					log.Println(err)
				}
			}
		}
	} else {
		log.Println(err)
	}

	groupsDir := filepath.Join(static.DataDir, "groups")
	files, err = ioutil.ReadDir(groupsDir)
	if err == nil {
		for _, file := range files {
			if file.IsDir() {
				config, err := ioutil.ReadFile(filepath.Join(groupsDir, file.Name(), "config.yaml"))
				if err == nil {
					var tmp groupConfig
					err = yaml.Unmarshal(config, &tmp)
					if err == nil {
						finalConfig := groupConfig{
							Name: tmp.Name, DirName: file.Name(), System: tmp.System,
							SystemIndex: -1, Modules: make(map[string]*moduleConfig)}
						if finalConfig.System != "" {
							for i := range c.systems {
								if c.systems[i].Name == finalConfig.System {
									finalConfig.SystemIndex = i
									break
								}
							}
							if finalConfig.SystemIndex == -1 {
								log.Println("unknown system name: " + finalConfig.System)
								finalConfig.System = ""
							}
						}
						static.constructModuleConfigs(finalConfig.Modules, tmp.Modules, items)
						c.groups = append(c.groups, group{
							Config: finalConfig, Heroes: make([]hero, 0, 16)})
					}
				} else {
					log.Println(err)
				}
			}
		}
	} else {
		log.Println(err)
	}

	heroesDir := filepath.Join(static.DataDir, "heroes")
	files, err = ioutil.ReadDir(heroesDir)
	if err == nil {
		for _, file := range files {
			if file.IsDir() {
				config, err := ioutil.ReadFile(filepath.Join(heroesDir, file.Name(), "config.yaml"))
				if err == nil {
					var h hero
					var target *group
					err = yaml.Unmarshal(config, &h)
					for i := range c.groups {
						if c.groups[i].Config.Name == h.Group {
							target = &c.groups[i]
							break
						}
					}
					if target == nil {
						log.Printf("Hero \"%s\" belongs to unknown group \"%s\"\n",
							h.Name, h.Group)
					} else {
						target.Heroes = append(target.Heroes, h)
					}
				} else {
					log.Println(err)
				}
			}
		}
	}
}

// UpdateConfig sets the configuration of the given module.
// It merges the default config with the configs from current system and group.
func (c *Config) UpdateConfig(defaultValues interface{}, item ConfigurableItem,
	systemIndex int, groupIndex int) {
	var configStack [4]*reflect.Value
	configType := reflect.TypeOf(defaultValues).Elem()

	if groupIndex != -1 {
		conf, ok := c.groups[groupIndex].Config.Modules[item.Name()]
		if !ok {
			panic("group config missing for " + item.Name())
		}
		val := reflect.ValueOf(conf.Config).Elem()
		for ; val.Kind() == reflect.Interface || val.Kind() == reflect.Ptr; val = val.Elem() {
		}
		if val.Kind() != reflect.Struct {
			panic("wrong kind of group value")
		}
		configStack[0] = &val
	}
	if systemIndex != -1 {
		conf, ok := c.systems[systemIndex].Modules[item.Name()]
		if !ok {
			panic("system config missing for " + item.Name())
		}
		val := reflect.ValueOf(conf.Config).Elem()
		for ; val.Kind() == reflect.Interface || val.Kind() == reflect.Ptr; val = val.Elem() {
		}
		if val.Kind() != reflect.Struct {
			panic("wrong kind of system value")
		}
		log.Println(val.Kind())
		configStack[1] = &val
	}

	baseConf, ok := c.baseConfigs[item.Name()]
	if !ok {
		panic("base config missing for " + item.Name())
	}
	baseValue := reflect.ValueOf(baseConf.Config).Elem()
	configStack[2] = &baseValue

	defaultValue := reflect.ValueOf(defaultValues).Elem()
	configStack[3] = &defaultValue

	result := reflect.New(configType)
	for i := 0; i < configType.NumField(); i++ {
		for j := 0; j < 4; j++ {
			if configStack[j] != nil {
				field := configStack[j].Field(i)
				result.Elem().Field(i).Set(field)
			}
		}
	}
	item.SetConfig(result.Interface())
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

type jsonConfigItem struct {
	Type    string
	Value   interface{}
	Default interface{}
}

type jsonModuleConfig struct {
	State        configuredModuleState
	DefaultState configuredModuleState
	Config       map[string]jsonConfigItem
}
