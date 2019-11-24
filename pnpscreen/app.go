package main

import (
	"errors"
	"io/ioutil"
	"log"
	"os/user"
	"path/filepath"

	"github.com/flyx/pnpscreen/modules"
	"github.com/flyx/pnpscreen/web"

	"github.com/flyx/pnpscreen/display"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"

	"github.com/flyx/pnpscreen/api"
	"github.com/flyx/pnpscreen/data"
)

// implements api.Resource
type resourceFile struct {
	name   string
	path   string
	group  int
	system int
}

func (r *resourceFile) Name() string {
	return r.name
}

func (r *resourceFile) Path() string {
	return r.path
}

type moduleEntry struct {
	module  api.Module
	enabled bool
}

// app is the main application. it implements api.Environment and app.App.
// this is logically a singleton, multiple instances are not supported.
type app struct {
	dataDir             string
	fonts               []api.FontFamily
	defaultBorderWidth  int32
	modules             []moduleEntry
	resourceCollections [][][]resourceFile
	config              data.Config
	display             display.Display
	activeGroup         int
	activeSystem        int
	html, js            []byte
}

// Init initializes the static data
func (a *app) Init(fullscreen bool, events display.Events, port uint16) {
	// create window and renderer
	var flags uint32 = sdl.WINDOW_OPENGL | sdl.WINDOW_ALLOW_HIGHDPI
	if fullscreen {
		flags |= sdl.WINDOW_FULLSCREEN_DESKTOP
	}
	window, err := sdl.CreateWindow("pnpscreen", sdl.WINDOWPOS_UNDEFINED,
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

	a.dataDir = filepath.Join(usr.HomeDir, ".local", "share", "pnpscreen")
	a.defaultBorderWidth = height / 133
	fontSizeMap := [6]int32{height / 37, height / 27, height / 19,
		height / 13, height / 8, height / 4}
	a.fonts = createFontCatalog(a.dataDir, fontSizeMap)
	if a.fonts == nil {
		panic("No font available. PnP Screen needs at least one font.")
	}
	a.modules = make([]moduleEntry, 0, 32)
	a.resourceCollections = make([][][]resourceFile, 0, 32)
	a.activeGroup = -1
	a.activeSystem = -1

	a.html = append(a.html, web.MustAsset("web/html/index-top.html")...)
	a.js = append(a.js, web.MustAsset("web/js/template.js")...)
	a.js = append(a.js, '\n')
	a.js = append(a.js, web.MustAsset("web/js/config.js")...)
	a.js = append(a.js, '\n')
	a.js = append(a.js, web.MustAsset("web/js/app.js")...)
	a.js = append(a.js, '\r')
	a.js = append(a.js, web.MustAsset("web/js/state.js")...)
	if err = a.registerPlugin(&modules.Base{}, renderer); err != nil {
		panic(err)
	}
	a.html = append(a.html, '\n')
	a.html = append(a.html, web.MustAsset("web/html/index-bottom.html")...)
	a.js = append(a.js, '\n')
	a.js = append(a.js, web.MustAsset("web/js/init.js")...)

	a.config.Init(a)
	a.loadModuleResources()
	if err := a.display.Init(a, events, fullscreen, port, window, renderer); err != nil {
		panic(err)
	}
}

func (a *app) DataDir(subdirs ...string) string {
	return filepath.Join(append([]string{a.dataDir}, subdirs...)...)
}

func (a *app) ModuleAt(index api.ModuleIndex) api.Module {
	return a.modules[index].module
}

func (a *app) NumModules() api.ModuleIndex {
	return api.ModuleIndex(len(a.modules))
}

func (a *app) ModuleEnabled(index api.ModuleIndex) bool {
	return a.modules[index].enabled
}

func (a *app) DefaultBorderWidth() int32 {
	return a.defaultBorderWidth
}

func (a *app) FontCatalog() []api.FontFamily {
	return a.fonts
}

func appendDir(resources []resourceFile, path string, group int, system int) []resourceFile {
	files, err := ioutil.ReadDir(path)
	if err == nil {
		for _, file := range files {
			if !file.IsDir() && file.Name()[0] != '.' {
				resources = append(resources, resourceFile{
					name: file.Name(), path: filepath.Join(path, file.Name()),
					group: group, system: system})
			}
		}
	} else {
		log.Println(err)
	}
	return resources
}

// listFiles queries the list of all files matching the given selector.
// Never returns directories.
func (a *app) listFiles(
	id string, selector api.ResourceSelector) []resourceFile {
	resources := make([]resourceFile, 0, 64)
	resources = appendDir(
		resources, a.DataDir("base", id, selector.Subdirectory), -1, -1)
	for i := 0; i < a.config.NumSystems(); i++ {
		if a.config.SystemID(i) != "" {
			resources = appendDir(
				resources, a.DataDir("systems",
					a.config.SystemID(i), id, selector.Subdirectory), -1, i)
		}
	}
	for i := 0; i < a.config.NumGroups(); i++ {
		if a.config.GroupID(i) != "" {
			resources = appendDir(resources, a.DataDir("groups",
				a.config.GroupID(i), id, selector.Subdirectory), i, -1)
		}
	}
	return resources
}

func (a *app) registerModule(module api.Module, renderer *sdl.Renderer) error {
	if err := module.Init(renderer, a, api.ModuleIndex(len(a.modules))); err != nil {
		a.resourceCollections = a.resourceCollections[:len(a.resourceCollections)-1]
		return err
	}
	a.modules = append(a.modules, moduleEntry{module: module, enabled: true})
	return nil
}

func (a *app) registerPlugin(plugin api.Plugin, renderer *sdl.Renderer) error {
	println("Loading plugin", plugin.Name())
	a.js = append(a.js, '\n')
	a.js = append(a.js, plugin.AdditionalJS()...)
	a.html = append(a.html, '\n')
	a.html = append(a.html, plugin.AdditionalHTML()...)
	modules := plugin.Modules()
	for i := range modules {
		if err := a.registerModule(modules[i], renderer); err != nil {
			return err
		}
	}
	return nil
}

func (a *app) loadModuleResources() {
	for i := range a.modules {
		module := a.modules[i].module
		collections := make([][]resourceFile, 0, 32)
		selectors := module.ResourceCollections()
		for i := range selectors {
			collections = append(collections, a.listFiles(module.ID(), selectors[i]))
		}
		a.resourceCollections = append(a.resourceCollections, collections)
	}
}

type emptyHeroList struct{}

func (emptyHeroList) Item(index int) api.Hero {
	panic("out of range!")
}

func (emptyHeroList) Length() int {
	return 0
}

func (a *app) Heroes() api.HeroList {
	if a.activeGroup == -1 {
		return emptyHeroList{}
	}
	return a.config.GroupHeroes(a.activeGroup)
}

// Font returns the font face of the selected font.
func (a *app) Font(
	fontFamily int, style api.FontStyle, size api.FontSize) *ttf.Font {
	return a.fonts[fontFamily].Styled(style).Font(size)
}

func (a *app) findModule(name string) (api.Module, int) {
	for i := range a.modules {
		if a.modules[i].module.Name() == name {
			return a.modules[i].module, i
		}
	}
	return nil, -1
}

// GetResources filters resources by current group and system.
func (a *app) GetResources(
	moduleIndex api.ModuleIndex, index api.ResourceCollectionIndex) []api.Resource {
	complete := a.resourceCollections[moduleIndex][index]
	ret := make([]api.Resource, 0, len(complete))
	for i := range complete {
		if (complete[i].group == -1 || complete[i].group == a.activeGroup) &&
			(complete[i].system == -1 || complete[i].system == a.activeSystem) {
			ret = append(ret, &complete[i])
		}
	}
	return ret
}

func (a *app) pathToState() string {
	return filepath.Join(a.dataDir, "groups",
		a.config.GroupID(a.activeGroup), "state.yaml")
}

// SetActiveGroup changes the active group to the group at the given index.
// it loads the state of that group into all modules.
func (a *app) setActiveGroup(index int) error {
	if index < 0 || index >= a.config.NumGroups() {
		return errors.New("index out of range")
	}
	a.activeGroup = index
	a.activeSystem = a.config.GroupLinkedSystem(index)
	stateInput, _ := ioutil.ReadFile(a.pathToState())
	a.config.LoadYamlGroupState(stateInput)
	return nil
}

func (a *app) destroy() {
	a.display.Destroy()
}
