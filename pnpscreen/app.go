package main

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"

	"github.com/flyx/pnpscreen/api"
	base "github.com/flyx/pnpscreen/base"
	"github.com/flyx/pnpscreen/data"
	"github.com/flyx/pnpscreen/display"
	"github.com/flyx/pnpscreen/web"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
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

// app is the main application. it implements api.Environment and app.App.
// this is logically a singleton, multiple instances are not supported.
type app struct {
	dataDir             string
	fonts               []api.FontFamily
	defaultBorderWidth  int32
	modules             []api.Module
	plugins             []*api.Plugin
	resourceCollections [][][]resourceFile
	config              data.Config
	persistence         data.Persistence
	communication       data.Communication
	groupState          *data.GroupState
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
	a.modules = make([]api.Module, 0, 32)
	a.resourceCollections = make([][][]resourceFile, 0, 32)
	a.activeGroupIndex = -1
	a.activeSystemIndex = -1

	a.html = appendAssets(a.html, "web/html/index-top.html")
	a.js = appendAssets(a.js, "web/js/ui.js", "web/js/template.js",
		"web/js/popup.js", "web/js/datasets.js",
		"web/js/config.js", "web/js/app.js", "web/js/state.js")
	a.css = appendAssets(a.css, "web/css/style.css", "web/css/color.css")
	if err = a.registerPlugin(&base.Base, renderer); err != nil {
		panic(err)
	}
	a.html = appendAssets(a.html, "web/html/index-bottom.html")
	a.js = appendAssets(a.js, "web/js/init.js")

	a.persistence, a.communication = a.config.LoadPersisted(a)
	a.loadModuleResources()
	if err := a.display.Init(
		a, events, fullscreen, port, window, renderer); err != nil {
		panic(err)
	}
}

func (a *app) DataDir(subdirs ...string) string {
	return filepath.Join(append([]string{a.dataDir}, subdirs...)...)
}

func (a *app) ModuleAt(index api.ModuleIndex) api.Module {
	return a.modules[index]
}

func (a *app) NumModules() api.ModuleIndex {
	return api.ModuleIndex(len(a.modules))
}

func (a *app) DefaultBorderWidth() int32 {
	return a.defaultBorderWidth
}

func (a *app) FontCatalog() []api.FontFamily {
	return a.fonts
}

func (a *app) NumPlugins() int {
	return len(a.plugins)
}

func (a *app) Plugin(index int) *api.Plugin {
	return a.plugins[index]
}

func appendDir(resources []resourceFile, path string, group int, system int) []resourceFile {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return resources
	}
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
		if a.config.System(i).ID() != "" {
			resources = appendDir(
				resources, a.DataDir("systems",
					a.config.System(i).ID(), id, selector.Subdirectory), -1, i)
		}
	}
	for i := 0; i < a.config.NumGroups(); i++ {
		group := a.config.Group(i)
		if group.ID() != "" {
			resources = appendDir(resources, a.DataDir("groups",
				group.ID(), id, selector.Subdirectory), i, -1)
		}
	}
	return resources
}

func (a *app) registerModule(descr *api.ModuleDescriptor,
	renderer *sdl.Renderer) error {
	module, err := descr.CreateModule(renderer, a)
	if err != nil {
		a.resourceCollections = a.resourceCollections[:len(a.resourceCollections)-1]
		return err
	}
	a.modules = append(a.modules, module)
	return nil
}

func (a *app) registerPlugin(plugin *api.Plugin, renderer *sdl.Renderer) error {
	println("Loading plugin", plugin.Name)
	if js := plugin.AdditionalJS; js != nil {
		a.js = append(a.js, '\n')
		a.js = append(a.js, js...)
	}
	if html := plugin.AdditionalHTML; html != nil {
		a.html = append(a.html, '\n')
		a.html = append(a.html, html...)
	}
	if css := plugin.AdditionalCSS; css != nil {
		a.css = append(a.css, '\n')
		a.css = append(a.css, css...)
	}
	modules := plugin.Modules
	for i := range modules {
		if err := a.registerModule(modules[i], renderer); err != nil {
			return err
		}
	}
	a.plugins = append(a.plugins, plugin)
	return nil
}

func (a *app) loadModuleResources() {
	for i := range a.modules {
		descr := a.modules[i].Descriptor()
		collections := make([][]resourceFile, 0, 32)
		selectors := descr.ResourceCollections
		for i := range selectors {
			collections = append(collections, a.listFiles(descr.ID, selectors[i]))
		}
		a.resourceCollections = append(a.resourceCollections, collections)
	}
}

func (a *app) activeGroup() data.Group {
	if a.activeGroupIndex == -1 {
		return nil
	}
	return a.config.Group(a.activeGroupIndex)
}

type emptyHeroList struct{}

func (emptyHeroList) Item(index int) api.Hero {
	panic("out of range!")
}

func (emptyHeroList) Length() int {
	return 0
}

func (a *app) Heroes() api.HeroView {
	g := a.activeGroup()
	if g == nil {
		return nil
	}
	return g.ViewHeroes()
}

// Font returns the font face of the selected font.
func (a *app) Font(
	fontFamily int, style api.FontStyle, size api.FontSize) *ttf.Font {
	return a.fonts[fontFamily].Styled(style).Font(size)
}

// GetResources filters resources by current group and system.
func (a *app) GetResources(
	moduleIndex api.ModuleIndex, index api.ResourceCollectionIndex) []api.Resource {
	complete := a.resourceCollections[moduleIndex][index]
	ret := make([]api.Resource, 0, len(complete))
	for i := range complete {
		if (complete[i].group == -1 || complete[i].group == a.activeGroupIndex) &&
			(complete[i].system == -1 || complete[i].system == a.activeSystemIndex) {
			ret = append(ret, &complete[i])
		}
	}
	return ret
}

func (a *app) pathToState() string {
	return filepath.Join(a.dataDir, "groups",
		a.activeGroup().ID(), "state.yaml")
}

// SetActiveGroup changes the active group to the group at the given index.
// it loads the state of that group into all modules.
//
// Returns the index of the currently active scene inside the group
func (a *app) setActiveGroup(index int) (int, error) {
	if index < 0 || index >= a.config.NumGroups() {
		return -1, errors.New("index out of range")
	}
	a.activeGroupIndex = index
	group := a.activeGroup()
	a.activeSystemIndex = group.SystemIndex()
	groupState, err := data.LoadPersistedGroupState(a, group, a.pathToState())
	if err != nil {
		return -1, err
	}
	a.groupState = groupState
	return groupState.ActiveScene(), nil
}

func (a *app) destroy() {
	a.display.Destroy()
}
