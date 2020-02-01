package data

import (
	"log"
	"reflect"
	"sync"

	"github.com/QuestScreen/QuestScreen/api"
	"github.com/QuestScreen/QuestScreen/app"
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
	id          string
	description string
}

func (h *hero) Name() string {
	return h.name
}

func (h *hero) ID() string {
	return h.id
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
	UsesModule(moduleIndex app.ModuleIndex) bool
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

func (s *scene) UsesModule(moduleIndex app.ModuleIndex) bool {
	return s.modules[moduleIndex].enabled
}

type heroList struct {
	mutex sync.Mutex
	data  []hero
}

func (hl *heroList) Hero(index int) api.Hero {
	return &hl.data[index]
}

func (hl *heroList) HeroByID(id string) (index int, h api.Hero) {
	for i := range hl.data {
		if hl.data[i].id == id {
			return i, &hl.data[i]
		}
	}
	return -1, nil
}

func (hl *heroList) NumHeroes() int {
	return len(hl.data)
}

func (hl *heroList) Close() {
	hl.mutex.Unlock()
}

// Group describes a Pen & Paper group / party
type Group interface {
	Name() string
	ID() string
	SystemIndex() int
	NumScenes() int
	Scene(index int) Scene
	SceneByID(id string) (index int, s Scene)
	ViewHeroes() app.HeroView
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

func (g *group) SceneByID(id string) (index int, s Scene) {
	for i := range g.scenes {
		if g.scenes[i].id == id {
			return i, &g.scenes[i]
		}
	}
	return -1, nil
}

func (g *group) ViewHeroes() app.HeroView {
	g.heroes.mutex.Lock()
	return &g.heroes
}

// Data contains all non-transient data currently loaded by PnpScreen
type Data struct {
	owner            app.App
	baseConfigs      []interface{}
	systems          []*system
	groups           []*group
	numPluginSystems int
	State
}

// NumPluginSystems returns the number of systems required by plugins.
// these systems are always in front of the systems list.
func (d *Data) NumPluginSystems() int {
	return d.numPluginSystems
}

// NumSystems returns the number of available systems.
func (d *Data) NumSystems() int {
	return len(d.systems)
}

// System returns the system at the given index, which must be
// between 0 (included) and NumSystems() (excluded).
func (d *Data) System(index int) System {
	return d.systems[index]
}

// SystemByID returns the system with the given id, or nil if no such system
// exists.
func (d *Data) SystemByID(id string) (index int, value System) {
	for i := range d.systems {
		if d.systems[i].id == id {
			return i, d.systems[i]
		}
	}
	return -1, nil
}

// NumGroups returns the number of available groups.
func (d *Data) NumGroups() int {
	return len(d.groups)
}

// Group returns the group at the given index, which must be
// between 0 (included) and NumGroups() (excluded).
func (d *Data) Group(index int) Group {
	return d.groups[index]
}

// GroupByID returns the group with the given id, or nil if no such group
// exists.
func (d *Data) GroupByID(id string) (index int, value Group) {
	for i := range d.groups {
		if d.groups[i].id == id {
			return i, d.groups[i]
		}
	}
	return -1, nil
}

// LoadPersisted loads all config.yaml files and parses them according to the
// module's config types. Returns a Persistence value linked to the config that
// can be used to persist data into the file system, and a Communication value
// that (de)serializes data for client communication.
//
// All errors are logged and erratic files ignored.
// You must call LoadPersisted before doing anything with a Config value.
func (d *Data) LoadPersisted(owner app.App) (Persistence, Communication) {
	p := Persistence{d}
	d.owner = owner
	basePath := d.owner.DataDir("base", "config.yaml")
	ret, err := p.loadBase(basePath)
	if err != nil {
		log.Println(basePath+":", err)
	}
	d.baseConfigs = ret
	p.loadSystems()
	p.loadGroups()
	return p, Communication{d}
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
// configured in its base config and the current system, group and scene config.
// It returns the resulting configuration.
//
// systemIndex may be -1 (for groups without a defined system), groupIndex and
// sceneIndex may not.
func (d *Data) MergeConfig(moduleIndex app.ModuleIndex,
	systemIndex int, groupIndex int, sceneIndex int) interface{} {
	var configStack [5]*reflect.Value
	module := d.owner.ModuleAt(moduleIndex)

	defaultValues := module.Descriptor().DefaultConfig
	configType := reflect.TypeOf(defaultValues).Elem()

	{
		conf := d.groups[groupIndex].scenes[sceneIndex].modules[moduleIndex].config
		if conf == nil {
			configStack[0] = nil
		} else {
			configStack[0] = confValue(conf)
		}
	}
	{
		conf := d.groups[groupIndex].modules[moduleIndex]
		if conf == nil {
			panic("group config missing for " + module.Descriptor().ID)
		}
		configStack[1] = confValue(conf)
	}
	if systemIndex != -1 {
		conf := d.systems[systemIndex].modules[moduleIndex]
		if conf == nil {
			panic("system config missing for " + module.Descriptor().ID)
		}
		configStack[2] = confValue(conf)
	}

	baseConf := d.baseConfigs[moduleIndex]
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
