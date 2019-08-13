/*
Package config implements loading and writing the config.yaml files.
*/
package config

import (
	"io/ioutil"
	"log"
	"os/user"
	"path/filepath"

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

type transientModuleConfig struct {
	State  configuredModuleState
	Config yaml.Node
}

type systemConfig struct {
	Name    string
	DirName string `yaml:"-"`
	Modules map[string]moduleConfig
}

type transientSystemConfig struct {
	Name    string
	Modules map[string]transientModuleConfig
}

type groupConfig struct {
	Name        string
	DirName     string `yaml:"-"`
	System      string
	SystemIndex int `yaml:"-"`
	Modules     map[string]moduleConfig
}

type transientGroupConfig struct {
	Name    string
	System  string
	Modules map[string]transientModuleConfig
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
	baseConfigs map[string]moduleConfig
	systems     []systemConfig
	groups      []group
	DataDir     string
}

// ConfigurableItem describes an item that may be configured via a Config object
type ConfigurableItem interface {
	// Name gives the name of this item.
	Name() string
	// ToConfig parses a YAML node inside config yaml and returns the result.
	// the module defines the type of its configuration; however the configuration
	// belongs to SharedData.
	ToConfig(node *yaml.Node) (interface{}, error)
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

func findItem(items ConfigurableItemProvider, name string) ConfigurableItem {
	for i := 0; i < items.NumItems(); i++ {
		if items.ItemAt(i).Name() == name {
			return items.ItemAt(i)
		}
	}
	return nil
}

func constructModuleConfigs(data map[string]moduleConfig,
	raw map[string]transientModuleConfig, items ConfigurableItemProvider) {
	for name, node := range raw {
		mod := findItem(items, name)
		if mod == nil {
			log.Println("Unknown module: " + name)
		} else {
			evaluated, err := mod.ToConfig(&node.Config)
			if err == nil {
				data[name] = moduleConfig{
					State: node.State, Config: evaluated}
			} else {
				log.Println(err)
			}
		}
	}
}

func buildConfig(basic interface{}, module interface{},
	group interface{}) interface{} {
	/*configType := reflect.TypeOf(basic)
	basicVal := reflect.ValueOf(basic)
	moduleVal := reflect.ValueOf(module)
	groupVal := reflect.ValueOf(group)*/
	// TODO
	return nil
}

// Init loads all config.yaml files and parses them according to the module's
// config types. Parsing errors lead to a panic while structural errors are
// logged and ignored.
//
// You must call Init before doing anything with a Config value.
func (c *Config) Init(items ConfigurableItemProvider) {
	usr, _ := user.Current()
	c.groups = make([]group, 0, 16)
	c.DataDir = filepath.Join(usr.HomeDir, ".local", "share", "rpscreen")

	rawBaseConfig, err := ioutil.ReadFile(filepath.Join(c.DataDir, "config.yaml"))
	if err != nil {
		panic(err)
	}
	var nodes map[string]transientModuleConfig
	err = yaml.Unmarshal(rawBaseConfig, &nodes)
	if err != nil {
		panic(err)
	}
	c.baseConfigs = make(map[string]moduleConfig)
	constructModuleConfigs(c.baseConfigs, nodes, items)

	systemsDir := filepath.Join(c.DataDir, "systems")
	files, err := ioutil.ReadDir(systemsDir)
	if err == nil {
		for _, file := range files {
			if file.IsDir() {
				config, err := ioutil.ReadFile(filepath.Join(systemsDir, file.Name(), "config.yaml"))
				if err == nil {
					var tmp transientSystemConfig
					err = yaml.Unmarshal(config, &tmp)
					if err == nil {
						finalConfig := systemConfig{
							Name: tmp.Name, DirName: file.Name(),
							Modules: make(map[string]moduleConfig)}
						constructModuleConfigs(finalConfig.Modules, tmp.Modules, items)
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

	groupsDir := filepath.Join(c.DataDir, "groups")
	files, err = ioutil.ReadDir(groupsDir)
	if err == nil {
		for _, file := range files {
			if file.IsDir() {
				config, err := ioutil.ReadFile(filepath.Join(groupsDir, file.Name(), "config.yaml"))
				if err == nil {
					var tmp transientGroupConfig
					err = yaml.Unmarshal(config, &tmp)
					if err == nil {
						finalConfig := groupConfig{
							Name: tmp.Name, DirName: file.Name(), System: tmp.System,
							SystemIndex: -1, Modules: make(map[string]moduleConfig)}
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
						constructModuleConfigs(finalConfig.Modules, tmp.Modules, items)
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

	heroesDir := filepath.Join(c.DataDir, "heroes")
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
