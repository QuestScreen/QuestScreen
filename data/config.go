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

type systemConfig struct {
	Name    string
	DirName string
	Modules []interface{}
}

type groupConfig struct {
	Name        string
	DirName     string
	SystemIndex int
	Modules     []interface{}
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
	baseConfigs []interface{}
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
	// The configuration object must be a pointer.
	DefaultConfig() interface{}
	// SetConfig sets current configuration for the item.
	// This must be done in the OpenGL thread since the calculated configuration
	// belongs to the module.
	// returns true iff the module must re-render via RebuildState().
	SetConfig(config interface{}) bool
	// GetConfig retrieves the current configuration of the item.
	GetConfig() interface{}
	// GetState retrieves the current state of the item.
	GetState() ModuleState
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

// Init loads all config.yaml files and parses them according to the module's
// config types. Parsing errors lead to a panic while structural errors are
// logged and ignored.
//
// You must call Init before doing anything with a Config value.
func (c *Config) Init(static *StaticData) {
	rawBaseConfig, err := ioutil.ReadFile(filepath.Join(static.DataDir, "base", "config.yaml"))
	if err != nil {
		panic(err)
	}
	c.baseConfigs = static.loadYamlBaseConfig(rawBaseConfig)

	c.systems = make([]systemConfig, 0, 16)
	systemsDir := filepath.Join(static.DataDir, "systems")
	files, err := ioutil.ReadDir(systemsDir)
	if err == nil {
		for _, file := range files {
			if file.IsDir() {
				config, err := ioutil.ReadFile(filepath.Join(systemsDir, file.Name(), "config.yaml"))
				if err == nil {
					c.systems = append(c.systems, static.loadYamlSystemConfig(config, file.Name()))
				} else {
					log.Println(err)
				}
			}
		}
	} else {
		log.Println(err)
	}

	c.groups = make([]group, 0, 16)
	groupsDir := filepath.Join(static.DataDir, "groups")
	files, err = ioutil.ReadDir(groupsDir)
	if err == nil {
		for _, file := range files {
			if file.IsDir() {
				config, err := ioutil.ReadFile(filepath.Join(groupsDir, file.Name(), "config.yaml"))
				if err == nil {
					config := static.loadYamlGroupConfig(config, file.Name(), c.systems)
					c.groups = append(c.groups, group{
						Config: config,
						Heroes: make([]hero, 0, 16),
					})
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

// MergeConfig merges the item's default configuration with the values
// configured in its base config and the current system and group config.
// It returns the resulting configuration.
func (c *Config) MergeConfig(staticData *StaticData,
	itemIndex int, systemIndex int, groupIndex int) interface{} {
	var configStack [4]*reflect.Value
	item := staticData.items.ItemAt(itemIndex)

	defaultValues := item.DefaultConfig()
	configType := reflect.TypeOf(defaultValues).Elem()

	if groupIndex != -1 {
		conf := c.groups[groupIndex].Config.Modules[itemIndex]
		if conf == nil {
			panic("group config missing for " + item.Name())
		}
		val := reflect.ValueOf(conf).Elem()
		for ; val.Kind() == reflect.Interface || val.Kind() == reflect.Ptr; val = val.Elem() {
		}
		if val.Kind() != reflect.Struct {
			panic("wrong kind of group value")
		}
		configStack[0] = &val
	}
	if systemIndex != -1 {
		conf := c.systems[systemIndex].Modules[itemIndex]
		if conf == nil {
			panic("system config missing for " + item.Name())
		}
		val := reflect.ValueOf(conf).Elem()
		for ; val.Kind() == reflect.Interface || val.Kind() == reflect.Ptr; val = val.Elem() {
		}
		if val.Kind() != reflect.Struct {
			panic("wrong kind of system value")
		}
		configStack[1] = &val
	}

	baseConf := c.baseConfigs[itemIndex]
	if baseConf == nil {
		panic("base config missing for " + item.Name())
	}
	baseValue := reflect.ValueOf(baseConf).Elem()
	configStack[2] = &baseValue

	defaultValue := reflect.ValueOf(defaultValues).Elem()
	configStack[3] = &defaultValue

	result := reflect.New(configType)
	for i := 0; i < configType.NumField(); i++ {
		for j := 0; j < 4; j++ {
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
