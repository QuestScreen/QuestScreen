package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"

	"github.com/QuestScreen/QuestScreen/api"
	"github.com/QuestScreen/QuestScreen/app"
	base "github.com/QuestScreen/QuestScreen/base"
	"github.com/QuestScreen/QuestScreen/data"
	"github.com/QuestScreen/QuestScreen/display"
	"github.com/QuestScreen/QuestScreen/web"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

// implements api.Resource
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
	api.Module
	pluginIndex int
}

// QuestScreen is the main application. it implements app.App.
// this is logically a singleton, multiple instances are not supported.
type QuestScreen struct {
	dataDir             string
	fonts               []api.FontFamily
	modules             []moduleData
	plugins             []*api.Plugin
	resourceCollections [][][]ownedResourceFile
	textures            []api.Resource
	data                data.Data
	persistence         data.Persistence
	communication       data.Communication
	display             display.Display
	activeGroupIndex    int
	activeSystemIndex   int
	html, js, css       []byte
}

func appendAssets(buffer []byte, paths ...string) []byte {
	for i := range paths {
		buffer = append(buffer, web.MustAsset(paths[i])...)
		buffer = append(buffer, '\n')
	}
	return buffer
}

// Init initializes the static data
func (qs *QuestScreen) Init(fullscreen bool, events display.Events, port uint16) {
	// create window and renderer
	var flags uint32 = sdl.WINDOW_OPENGL | sdl.WINDOW_ALLOW_HIGHDPI
	if fullscreen {
		flags |= sdl.WINDOW_FULLSCREEN_DESKTOP
	}
	window, err := sdl.CreateWindow("QuestScreen", sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED, 800, 600, flags)
	if err != nil {
		panic(err)
	}

	renderer, err := sdl.CreateRenderer(window, -1,
		sdl.RENDERER_ACCELERATED|sdl.RENDERER_TARGETTEXTURE)
	if err != nil {
		window.Destroy()
		panic(err)
	}

	_, height, _ := renderer.GetOutputSize()

	usr, _ := user.Current()

	qs.dataDir = filepath.Join(usr.HomeDir, ".local", "share", "questscreen")
	fontSizeMap := [6]int32{height / 37, height / 27, height / 19,
		height / 13, height / 8, height / 4}
	qs.fonts = createFontCatalog(qs.dataDir, fontSizeMap)
	if qs.fonts == nil {
		panic("No font available. PnP Screen needs at least one font.")
	}
	qs.modules = make([]moduleData, 0, 32)
	qs.resourceCollections = make([][][]ownedResourceFile, 0, 32)
	qs.activeGroupIndex = -1
	qs.activeSystemIndex = -1

	qs.html = appendAssets(qs.html, "web/html/index-top.html")
	qs.js = appendAssets(qs.js, "web/js/template.js", "web/js/controls.js",
		"web/js/popup.js", "web/js/datasets.js",
		"web/js/config.js", "web/js/app.js", "web/js/state.js", "web/js/configitems.js")
	qs.css = appendAssets(qs.css, "web/css/style.css", "web/css/color.css")
	if err = qs.registerPlugin(&base.Base, renderer); err != nil {
		panic(err)
	}
	qs.html = appendAssets(qs.html, "web/html/index-bottom.html")
	qs.js = appendAssets(qs.js, "web/js/init.js")

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
		qs, events, fullscreen, port, window, renderer); err != nil {
		panic(err)
	}
}

// DataDir returns the path to the subdirectory specified by the given list of
// subdirs inside QuestScreen's data directory
func (qs *QuestScreen) DataDir(subdirs ...string) string {
	return filepath.Join(append([]string{qs.dataDir}, subdirs...)...)
}

// ModuleAt returns the module at the given index
func (qs *QuestScreen) ModuleAt(index app.ModuleIndex) api.Module {
	return qs.modules[index].Module
}

func (qs *QuestScreen) moduleByID(id string) (index int, module api.Module) {
	for i := range qs.modules {
		if qs.modules[i].Module.Descriptor().ID == id {
			return i, qs.modules[i].Module
		}
	}
	return -1, nil
}

// ModulePluginIndex returns the plugin the provides the module at the given index
func (qs *QuestScreen) ModulePluginIndex(index app.ModuleIndex) int {
	return qs.modules[index].pluginIndex
}

// NumModules returns the number of registered modules
func (qs *QuestScreen) NumModules() app.ModuleIndex {
	return app.ModuleIndex(len(qs.modules))
}

type moduleContext struct {
	*QuestScreen
	moduleIndex app.ModuleIndex
	heroes      api.HeroList
}

// GetResources filters resources by current group and system.
func (mc *moduleContext) GetResources(index api.ResourceCollectionIndex) []api.Resource {
	return mc.QuestScreen.GetResources(mc.moduleIndex, index)
}

func (mc *moduleContext) NumFontFamilies() int {
	return len(mc.fonts)
}

func (mc *moduleContext) FontFamilyName(index int) string {
	return mc.fonts[index].Name()
}

func (mc *moduleContext) NumHeroes() int {
	return mc.heroes.NumHeroes()
}

func (mc *moduleContext) HeroID(index int) string {
	return mc.heroes.Hero(index).ID()
}

type emptyHeroList struct{}

func (emptyHeroList) Hero(index int) api.Hero {
	panic("out of range!")
}

func (emptyHeroList) NumHeroes() int {
	return 0
}

func (emptyHeroList) Close() {}

func (emptyHeroList) HeroByID(id string) (index int, h api.Hero) {
	return -1, nil
}

// ServerContext returns a server context for the module at the given index
func (qs *QuestScreen) ServerContext(moduleIndex app.ModuleIndex,
	heroes api.HeroList) api.ServerContext {
	var h api.HeroList
	if heroes == nil {
		h = emptyHeroList{}
	} else {
		h = heroes
	}

	return &moduleContext{QuestScreen: qs, moduleIndex: moduleIndex, heroes: h}
}

// NumPlugins returns the number of registered plugins
func (qs *QuestScreen) NumPlugins() int {
	return len(qs.plugins)
}

// Plugin returns the plugin with the given index
func (qs *QuestScreen) Plugin(index int) *api.Plugin {
	return qs.plugins[index]
}

func appendBySelector(resources []ownedResourceFile, basePath string,
	selector api.ResourceSelector, group int, system int) []ownedResourceFile {
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
	id string, selector api.ResourceSelector) []ownedResourceFile {
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

var forbiddenNames = [5]string{"scenes", "heroes", "textures", "config.yaml", "state.yaml"}

func (qs *QuestScreen) registerModule(descr *api.ModuleDescriptor,
	renderer *sdl.Renderer) error {
	for i := range forbiddenNames {
		if descr.ID == forbiddenNames[i] {
			return fmt.Errorf("module id may not be one of %v", forbiddenNames)
		}
	}
	module, err := descr.CreateModule(renderer)
	if err != nil {
		return err
	}
	qs.modules = append(qs.modules, moduleData{module, len(qs.plugins)})
	return nil
}

func (qs *QuestScreen) registerPlugin(plugin *api.Plugin, renderer *sdl.Renderer) error {
	log.Println("Loading plugin", plugin.Name)
	if js := plugin.AdditionalJS; js != nil {
		qs.js = append(qs.js, '\n')
		qs.js = append(qs.js, js...)
	}
	if html := plugin.AdditionalHTML; html != nil {
		qs.html = append(qs.html, '\n')
		qs.html = append(qs.html, html...)
	}
	if css := plugin.AdditionalCSS; css != nil {
		qs.css = append(qs.css, '\n')
		qs.css = append(qs.css, css...)
	}
	modules := plugin.Modules
	for i := range modules {
		if err := qs.registerModule(modules[i], renderer); err != nil {
			log.Println("While registering module " + plugin.Name + " > " + modules[i].Name + ":")
			log.Println("  " + err.Error())
		}
	}
	qs.plugins = append(qs.plugins, plugin)
	return nil
}

func (qs *QuestScreen) loadModuleResources() {
	for i := range qs.modules {
		descr := qs.modules[i].Descriptor()
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

// Font returns the font face of the selected font.
func (qs *QuestScreen) Font(
	fontFamily int, style api.FontStyle, size api.FontSize) *ttf.Font {
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

// GetResources filters resources by current group and system.
func (qs *QuestScreen) GetResources(
	moduleIndex app.ModuleIndex, index api.ResourceCollectionIndex) []api.Resource {
	complete := qs.resourceCollections[moduleIndex][index]
	ret := make([]api.Resource, 0, len(complete))
	for i := range complete {
		if (complete[i].group == -1 || complete[i].group == qs.activeGroupIndex) &&
			(complete[i].system == -1 || complete[i].system == qs.activeSystemIndex) {
			ret = append(ret, &complete[i])
			if qs.modules[moduleIndex].Descriptor().ResourceCollections[index].Name != "" {
				// single file
				return ret
			}
		}
	}
	return ret
}

// GetTextures filters textures by current group and system.
func (qs *QuestScreen) GetTextures() []api.Resource {
	return qs.textures
}

// ViewHeroes returns a view of the heroes of the active group
func (qs *QuestScreen) ViewHeroes() app.HeroView {
	if qs.activeGroupIndex == -1 {
		return emptyHeroList{}
	}
	return qs.activeGroup().ViewHeroes()
}

func (qs *QuestScreen) pathToState() string {
	return filepath.Join(qs.dataDir, "groups",
		qs.activeGroup().ID(), "state.yaml")
}

// SetActiveGroup changes the active group to the group at the given index.
// it loads the state of that group into all modules.
//
// Returns the index of the currently active scene inside the group
func (qs *QuestScreen) setActiveGroup(index int) (int, api.SendableError) {
	qs.activeGroupIndex = index
	group := qs.activeGroup()
	qs.activeSystemIndex = group.SystemIndex()
	groupState, err := qs.persistence.LoadState(group, qs.pathToState())
	if err != nil {
		return -1, &api.InternalError{
			Description: "Failed to set active group", Inner: err}
	}
	return groupState.ActiveScene(), nil
}

func (qs *QuestScreen) destroy() {
	qs.display.Destroy()
}
