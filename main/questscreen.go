package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"reflect"

	"gopkg.in/yaml.v3"

	"github.com/QuestScreen/QuestScreen/app"
	"github.com/QuestScreen/QuestScreen/data"
	"github.com/QuestScreen/QuestScreen/display"
	"github.com/QuestScreen/QuestScreen/generated"
	"github.com/QuestScreen/QuestScreen/plugins"
	"github.com/QuestScreen/QuestScreen/shared"
	"github.com/QuestScreen/api/fonts"
	"github.com/QuestScreen/api/groups"
	"github.com/QuestScreen/api/modules"
	"github.com/QuestScreen/api/resources"
	"github.com/QuestScreen/api/server"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

// implements resources.Resource
type resourceFile struct {
	name string
	path string
}

func (r *resourceFile) Name() string {
	return r.name
}

func (r *resourceFile) Path() string {
	return r.path
}

type ownedResourceFile struct {
	resourceFile
	group  int
	system int
}

type moduleData struct {
	*modules.Module
	pluginIndex int
}

type pluginData struct {
	*app.Plugin
	id string
}

// QuestScreen is the main application. it implements app.App.
// this is logically a singleton, multiple instances are not supported.
type QuestScreen struct {
	appConfig
	dataDir             string
	fonts               []LoadedFontFamily
	modules             []moduleData
	plugins             []pluginData
	resourceCollections [][][]ownedResourceFile
	textures            []resources.Resource
	data                data.Data
	persistence         data.Persistence
	communication       data.Communication
	display             display.Display
	activeGroupIndex    int
	activeSystemIndex   int
	messages            []shared.Message
	html, js, css       []byte
	context             sdl.GLContext
}

// implements api.MessageSender
type messageCollector struct {
	owner       *QuestScreen
	moduleIndex shared.ModuleIndex
}

func (mc *messageCollector) Warning(text string) {
	mc.owner.messages = append(mc.owner.messages, shared.Message{
		IsError: false, ModuleIndex: mc.moduleIndex, Text: text})
}

func (mc *messageCollector) Error(text string) {
	mc.owner.messages = append(mc.owner.messages, shared.Message{
		IsError: true, ModuleIndex: mc.moduleIndex, Text: text})
}

// MessageSenderFor creates a new message sender for the given index.
func (qs *QuestScreen) MessageSenderFor(index shared.ModuleIndex) server.MessageSender {
	return &messageCollector{moduleIndex: index, owner: qs}
}

// PluginID returns the unique ID of the plugin with the given index.
func (qs *QuestScreen) PluginID(index int) string {
	return qs.plugins[index].id
}

// AddPlugin registers the given plugin's modules and config items with the app.
func (qs *QuestScreen) AddPlugin(id string, plugin *app.Plugin) error {
	for _, descr := range plugin.Modules {
		for i := range forbiddenNames {
			if descr.ID == forbiddenNames[i] {

				return fmt.Errorf("module id may not be one of %v", forbiddenNames)
			}
		}

		configType := reflect.TypeOf(descr.DefaultConfig)
		if configType.Kind() != reflect.Ptr {
			return errors.New("DefaultConfig's type is not a pointer to a struct")
		}
		configType = configType.Elem()
		if configType.Kind() != reflect.Struct {
			return errors.New("DefaultConfig's type is not a pointer to a struct")
		}
		for i := 0; i < configType.NumField(); i++ {
			fType := configType.Field(i).Type
			if fType.Kind() != reflect.Ptr {
				return errors.New("DefaultConfig." + configType.Field(i).Name + " is not a pointer")
			}
		}

		qs.modules = append(qs.modules, moduleData{descr, len(qs.plugins)})
	}
	qs.plugins = append(qs.plugins, pluginData{plugin, id})
	return nil
}

func (qs *QuestScreen) loadConfig(path string, width int32, height int32,
	port uint16, fullscreen bool) error {
	input, err := ioutil.ReadFile(path)
	if err == nil {
		decoder := yaml.NewDecoder(bytes.NewReader(input))
		decoder.KnownFields(true)
		err = decoder.Decode(&qs.appConfig)
		if err != nil {
			return err
		}
	} else {
		if os.IsNotExist(err) {
			qs.appConfig = defaultConfig()
			output, err := yaml.Marshal(&qs.appConfig)
			if err != nil {
				panic(err)
			}
			err = ioutil.WriteFile(path, output, 0644)
			if err != nil {
				log.Println("unable to write config file: " + err.Error())
			} else {
				log.Println("Wrote default config file " + path)
			}
		} else {
			return err
		}
	}
	if width != 0 && height != 0 {
		qs.appConfig.width = width
		qs.appConfig.height = height
		qs.appConfig.fullscreen = false
	} else if fullscreen {
		qs.appConfig.fullscreen = true
	}
	if port != 0 {
		qs.appConfig.port = port
	}
	return nil
}

// Init initializes the static data
func (qs *QuestScreen) Init(fullscreen bool, width int32, height int32,
	events display.Events, port uint16) {
	mc := messageCollector{owner: qs, moduleIndex: -1}

	usr, _ := user.Current()

	qs.dataDir = filepath.Join(usr.HomeDir, ".local", "share", "questscreen")
	os.MkdirAll(qs.DataDir("base"), 0755)
	os.MkdirAll(qs.DataDir("fonts"), 0755)
	os.MkdirAll(qs.DataDir("plugins"), 0755)
	os.MkdirAll(qs.DataDir("groups"), 0755)
	os.MkdirAll(qs.DataDir("systems"), 0755)
	os.MkdirAll(qs.DataDir("textures"), 0755)
	if err := qs.loadConfig(filepath.Join(qs.dataDir, "config.yaml"),
		width, height, port, fullscreen); err != nil {
		log.Printf("unable to read config. error was:\n  %s\n", err.Error())
		return
	}

	setGLAttributes()
	sdl.GLSetAttribute(sdl.GL_DOUBLEBUFFER, 1)

	// create window and renderer
	var flags uint32 = sdl.WINDOW_OPENGL | sdl.WINDOW_ALLOW_HIGHDPI
	if qs.fullscreen {
		flags |= sdl.WINDOW_FULLSCREEN_DESKTOP
	}
	window, err := sdl.CreateWindow("QuestScreen", sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED, qs.width, qs.height, flags)
	if err != nil {
		panic(err)
	}

	qs.context, err = window.GLCreateContext()
	if err != nil {
		panic(err)
	}
	sdl.GLSetSwapInterval(0)

	_, oHeight := window.GLGetDrawableSize()

	fontSizeMap := [6]int32{oHeight / 37, oHeight / 27, oHeight / 19,
		oHeight / 13, oHeight / 8, oHeight / 4}
	fontPath := filepath.Join(qs.dataDir, "fonts")
	qs.fonts = createFontCatalog(fontPath, fontSizeMap)
	if len(qs.fonts) == 0 {
		mc.Error("Did not find any fonts. " +
			"Please place at least one TTF/OTF font file in " + fontPath)
	}
	qs.modules = make([]moduleData, 0, 32)
	qs.resourceCollections = make([][][]ownedResourceFile, 0, 32)
	qs.activeGroupIndex = -1
	qs.activeSystemIndex = -1

	qs.html = generated.MustAsset("assets/index.html")
	qs.js = generated.MustAsset("assets/main.js")
	qs.css = generated.MustAsset("assets/style.css")
	qs.css = append(qs.css, '\n')
	qs.css = append(qs.css, generated.MustAsset("assets/color.css")...)

	plugins.LoadPlugins(qs)

	texturePath := qs.DataDir("textures")
	textureFiles, err := ioutil.ReadDir(texturePath)
	if err == nil {
		for _, file := range textureFiles {
			if !file.IsDir() && file.Name()[0] != '.' {
				path := filepath.Join(texturePath, file.Name())
				if _, err := os.Stat(path); err != nil {
					log.Printf("could not read file %s: %s\n", path, err.Error())
					continue
				}
				qs.textures = append(qs.textures, &resourceFile{
					name: file.Name(), path: path})
			}
		}
	}
	qs.persistence, qs.communication = qs.data.LoadPersisted(qs)
	qs.loadModuleResources()

	if err := qs.display.Init(
		qs, events, qs.fullscreen, qs.port, qs.keyActions,
		window); err != nil {
		panic(err)
	}
}

// DataDir returns the path to the subdirectory specified by the given list of
// subdirs inside QuestScreen's data directory
func (qs *QuestScreen) DataDir(subdirs ...string) string {
	return filepath.Join(append([]string{qs.dataDir}, subdirs...)...)
}

// ModuleAt returns the module at the given index
func (qs *QuestScreen) ModuleAt(index shared.ModuleIndex) *modules.Module {
	return qs.modules[index].Module
}

func (qs *QuestScreen) moduleByID(id string) (index int, module *modules.Module) {
	for i := range qs.modules {
		if qs.modules[i].Module.ID == id {
			return i, qs.modules[i].Module
		}
	}
	return -1, nil
}

// ModulePluginIndex returns the plugin the provides the module at the given index
func (qs *QuestScreen) ModulePluginIndex(index shared.ModuleIndex) int {
	return qs.modules[index].pluginIndex
}

// NumModules returns the number of registered modules
func (qs *QuestScreen) NumModules() shared.ModuleIndex {
	return shared.ModuleIndex(len(qs.modules))
}

// Messages returns the messages generated on app startup
func (qs *QuestScreen) Messages() []shared.Message {
	return qs.messages
}

type moduleContext struct {
	*QuestScreen
	moduleIndex shared.ModuleIndex
}

// GetResources filters resources by current group and system.
func (mc *moduleContext) GetResources(index resources.CollectionIndex) []resources.Resource {
	return mc.QuestScreen.GetResources(mc.moduleIndex, index)
}

func (mc *moduleContext) FontFamilyName(index int) string {
	return mc.fonts[index].Name()
}

type emptyHeroList struct{}

func (emptyHeroList) Hero(index int) groups.Hero {
	panic("out of range!")
}

func (emptyHeroList) NumHeroes() int {
	return 0
}

func (emptyHeroList) Close() {}

func (emptyHeroList) HeroByID(id string) (index int, h groups.Hero) {
	return -1, nil
}

// ServerContext returns a server context for the module at the given index
func (qs *QuestScreen) ServerContext(moduleIndex shared.ModuleIndex) server.Context {
	return &moduleContext{QuestScreen: qs, moduleIndex: moduleIndex}
}

// NumPlugins returns the number of registered plugins
func (qs *QuestScreen) NumPlugins() int {
	return len(qs.plugins)
}

// Plugin returns the plugin with the given index
func (qs *QuestScreen) Plugin(index int) *app.Plugin {
	return qs.plugins[index].Plugin
}

func appendBySelector(resources []ownedResourceFile, basePath string,
	selector resources.Selector, group int, system int) []ownedResourceFile {
	if selector.Name == "" {
		if _, err := os.Stat(basePath); os.IsNotExist(err) {
			return resources
		}
		files, err := ioutil.ReadDir(basePath)
		if err == nil {
			for _, file := range files {
				if !file.IsDir() && file.Name()[0] != '.' {
					path := filepath.Join(basePath, file.Name())
					if _, err := os.Stat(path); err != nil {
						log.Printf("could not read file %s: %s\n", path, err.Error())
						continue
					}
					if len(selector.Suffixes) > 0 {
						suffix := filepath.Ext(path)
						found := false
						for i := range selector.Suffixes {
							if suffix == selector.Suffixes[i] {
								found = true
								break
							}
						}
						if !found {
							continue
						}
					}

					resources = append(resources, ownedResourceFile{
						resourceFile: resourceFile{name: file.Name(), path: path},
						group:        group, system: system})
				}
			}
		} else {
			log.Println(err)
		}
	} else {
		path := filepath.Join(basePath, selector.Name)
		_, err := os.Stat(path)
		if err == nil {
			resources = append(resources, ownedResourceFile{
				resourceFile: resourceFile{name: selector.Name, path: path},
				group:        group, system: system,
			})
		} else if !os.IsNotExist(err) {
			log.Printf("could not read file %s: %s\n", path, err.Error())
		}
	}
	return resources
}

// listFiles queries the list of all files matching the given selector.
// Never returns directories.
func (qs *QuestScreen) listFiles(
	id string, selector resources.Selector) []ownedResourceFile {
	resources := make([]ownedResourceFile, 0, 64)
	for i := 0; i < qs.data.NumGroups(); i++ {
		group := qs.data.Group(i)
		basePath := qs.DataDir("groups", group.ID(), id, selector.Subdirectory)
		resources = appendBySelector(resources, basePath, selector, i, -1)
	}
	for i := 0; i < qs.data.NumSystems(); i++ {
		resources = appendBySelector(resources,
			qs.DataDir("systems", qs.data.System(i).ID(), id, selector.Subdirectory),
			selector, -1, i)
	}
	resources = appendBySelector(
		resources, qs.DataDir("base", id, selector.Subdirectory), selector, -1, -1)
	return resources
}

var forbiddenNames = [7]string{"scenes", "heroes", "fonts", "textures",
	"plugins", "config.yaml", "state.yaml"}

func (qs *QuestScreen) loadModuleResources() {
	for i := range qs.modules {
		descr := qs.modules[i]
		collections := make([][]ownedResourceFile, 0, 32)
		selectors := descr.ResourceCollections
		for i := range selectors {
			collections = append(collections, qs.listFiles(descr.ID, selectors[i]))
		}
		qs.resourceCollections = append(qs.resourceCollections, collections)
	}
}

func (qs *QuestScreen) activeGroup() data.Group {
	if qs.activeGroupIndex == -1 {
		return nil
	}
	return qs.data.Group(qs.activeGroupIndex)
}

// ActiveGroup returns the currently active group, or nil if no group is active.
func (qs *QuestScreen) ActiveGroup() groups.Group {
	return qs.activeGroup()
}

// Font returns the font face of the selected font.
func (qs *QuestScreen) Font(
	fontFamily int, style fonts.Style, size fonts.Size) *ttf.Font {
	return qs.fonts[fontFamily].Styled(style).Font(size)
}

// FontNames returns a list of the names of all loaded font families
func (qs *QuestScreen) FontNames() []string {
	ret := make([]string, 0, len(qs.fonts))
	for i := range qs.fonts {
		ret = append(ret, qs.fonts[i].Name())
	}
	return ret
}

// NumFontFamilies returns the number of font families
func (qs *QuestScreen) NumFontFamilies() int {
	return len(qs.fonts)
}

// GetResources filters resources by current group and system.
func (qs *QuestScreen) GetResources(
	moduleIndex shared.ModuleIndex, index resources.CollectionIndex) []resources.Resource {
	complete := qs.resourceCollections[moduleIndex][index]
	ret := make([]resources.Resource, 0, len(complete))
	for i := range complete {
		if (complete[i].group == -1 || complete[i].group == qs.activeGroupIndex) &&
			(complete[i].system == -1 || complete[i].system == qs.activeSystemIndex) {
			ret = append(ret, &complete[i])
			if qs.modules[moduleIndex].ResourceCollections[index].Name != "" {
				// single file
				return ret
			}
		}
	}
	return ret
}

// GetTextures filters textures by current group and system.
func (qs *QuestScreen) GetTextures() []resources.Resource {
	return qs.textures
}

func (qs *QuestScreen) pathToState() string {
	return filepath.Join(qs.dataDir, "groups",
		qs.activeGroup().ID(), "state.yaml")
}

// SetActiveGroup changes the active group to the group at the given index.
// it loads the state of that group into all modules.
//
// Returns the index of the currently active scene inside the group
func (qs *QuestScreen) setActiveGroup(index int) (int, server.Error) {
	qs.activeGroupIndex = index
	if index == -1 {
		qs.activeSystemIndex = -1
		return -1, nil
	}
	group := qs.activeGroup()
	qs.activeSystemIndex = group.SystemIndex()
	groupState, err := qs.persistence.LoadState(group, qs.pathToState())
	if err != nil {
		return -1, &server.InternalError{
			Description: "Failed to set active group", Inner: err}
	}
	return groupState.ActiveScene(), nil
}

func (qs *QuestScreen) destroy() {
	sdl.GLDeleteContext(qs.context)
	qs.display.Destroy()
}
