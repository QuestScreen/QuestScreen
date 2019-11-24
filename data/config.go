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

	"github.com/flyx/pnpscreen/api"
	"github.com/flyx/pnpscreen/app"
	"gopkg.in/yaml.v3"
)

type systemConfig struct {
	Name    string
	ID      string
	Modules []interface{}
}

type groupConfig struct {
	Name        string
	ID          string
	SystemIndex int
	Modules     []interface{}
}

type yamlHero struct {
	Name        string
	Description string
	Group       string
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

// group implements api.HeroList.
type group struct {
	Config groupConfig
	Heroes []hero
}

func (g *group) Item(index int) api.Hero {
	return &g.Heroes[index]
}

func (g *group) Length() int {
	return len(g.Heroes)
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
	owner       app.App
	baseConfigs []interface{}
	systems     []systemConfig
	groups      []group
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

// SystemID returns the name of the directory for the system at the given
// index, which must be between 0 (included) and NumSystems() (excluded).
// It is useful for retrieving data residing in the system's directory.
func (c *Config) SystemID(index int) string {
	return c.systems[index].ID
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

// GroupID returns the name of the directory for the group at the given
// index, which must be between 0 (included) and NumGroups() (excluded).
// It is useful for retrieving data residing in the group's directory.
func (c *Config) GroupID(index int) string {
	return c.groups[index].Config.ID
}

// GroupLinkedSystem returns the index of the system the group at the given
// index is linked to. The given index must be between 0 (included) and
// NumGroups() (excluded). If the group is not linked to a system,
// GroupLinkedSystem returns -1.
func (c *Config) GroupLinkedSystem(index int) int {
	return c.groups[index].Config.SystemIndex
}

// GroupHeroes returns the list of heroes of the group with the given index.
func (c *Config) GroupHeroes(index int) api.HeroList {
	return &c.groups[index]
}

// Init loads all config.yaml files and parses them according to the module's
// config types. Parsing errors lead to a panic while structural errors are
// logged and ignored.
//
// You must call Init before doing anything with a Config value.
func (c *Config) Init(owner app.App) {
	c.owner = owner

	path := owner.DataDir("base", "config.yaml")
	rawBaseConfig, err := ioutil.ReadFile(path)
	if err != nil {
		log.Println(path+":", err)
	} else {
		c.baseConfigs, err = c.loadYamlBaseConfig(rawBaseConfig)
		if err != nil {
			log.Println(path+":", err)
		}
	}

	c.systems = make([]systemConfig, 0, 16)
	systemsDir := owner.DataDir("systems")
	files, err := ioutil.ReadDir(systemsDir)
	if err == nil {
		for _, file := range files {
			if file.IsDir() {
				path := filepath.Join(systemsDir, file.Name(), "config.yaml")
				raw, err := ioutil.ReadFile(path)
				if err == nil {
					config, err := c.loadYamlSystemConfig(file.Name(), raw)
					if err == nil {
						c.systems = append(c.systems, config)
					} else {
						log.Println(path+":", err)
					}
				} else {
					log.Println(err)
				}
			}
		}
	} else {
		log.Println(err)
	}

	c.groups = make([]group, 0, 16)
	groupsDir := owner.DataDir("groups")
	files, err = ioutil.ReadDir(groupsDir)
	if err == nil {
		for _, file := range files {
			if file.IsDir() {
				path := filepath.Join(groupsDir, file.Name(), "config.yaml")
				raw, err := ioutil.ReadFile(path)
				if err == nil {
					config, err := c.loadYamlGroupConfig(file.Name(), raw)
					if err != nil {
						log.Println(path+":", err)
					} else {
						c.groups = append(c.groups, group{
							Config: config,
							Heroes: make([]hero, 0, 16),
						})
					}
				} else {
					log.Println(err)
				}
			}
		}
	} else {
		log.Println(err)
	}

	heroesDir := owner.DataDir("heroes")
	files, err = ioutil.ReadDir(heroesDir)
	if err == nil {
		for _, file := range files {
			if file.IsDir() {
				path := filepath.Join(heroesDir, file.Name(), "config.yaml")
				config, err := ioutil.ReadFile(path)
				if err == nil {
					var h yamlHero
					var target *group
					err = yaml.Unmarshal(config, &h)
					for i := range c.groups {
						if c.groups[i].Config.ID == h.Group {
							target = &c.groups[i]
							break
						}
					}
					if target == nil {
						log.Printf("%s: Hero \"%s\" belongs to unknown group \"%s\"\n",
							path, h.Name, h.Group)
					} else {
						target.Heroes = append(target.Heroes, hero{
							name: h.Name, description: h.Description})
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
func (c *Config) MergeConfig(moduleIndex api.ModuleIndex, systemIndex int, groupIndex int) interface{} {
	var configStack [4]*reflect.Value
	module := c.owner.ModuleAt(moduleIndex)

	defaultValues := module.DefaultConfig()
	configType := reflect.TypeOf(defaultValues).Elem()

	if groupIndex != -1 {
		conf := c.groups[groupIndex].Config.Modules[moduleIndex]
		if conf == nil {
			panic("group config missing for " + module.Name())
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
		conf := c.systems[systemIndex].Modules[moduleIndex]
		if conf == nil {
			panic("system config missing for " + module.Name())
		}
		val := reflect.ValueOf(conf).Elem()
		for ; val.Kind() == reflect.Interface || val.Kind() == reflect.Ptr; val = val.Elem() {
		}
		if val.Kind() != reflect.Struct {
			panic("wrong kind of system value")
		}
		configStack[1] = &val
	}

	baseConf := c.baseConfigs[moduleIndex]
	if baseConf == nil {
		panic("base config missing for " + module.Name())
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
