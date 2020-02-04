package app

import (
	"net/http"

	"github.com/QuestScreen/QuestScreen/api"
	"github.com/veandco/go-sdl2/ttf"
)

// ModuleIndex identifies the module internally.
type ModuleIndex int

// FirstModule is the index of the first module
const FirstModule ModuleIndex = 0

// HeroView is a exclusive view on the heroes of a group.
// it must be closed via Close() after acquiring it to release the mutex.
type HeroView interface {
	api.HeroList
	HeroByID(id string) (index int, h api.Hero)
	Close()
}

// App is the interface to the application for the data and display modules.
type App interface {
	DataDir(subdirs ...string) string
	NumModules() ModuleIndex
	ModuleAt(index ModuleIndex) api.Module
	ModulePluginIndex(index ModuleIndex) int
	NumPlugins() int
	Plugin(index int) *api.Plugin
	// ServerContext builds an api.ServerContext with the given moduleIndex and
	// hero list. The list may be queried by ViewHeroes (and closed afterwards).
	//
	// The list needs to be given separately so that a ServerContext can also be
	// created for a not currently active group (by querying the hero list from
	// the Group object).
	ServerContext(moduleIndex ModuleIndex, heroes api.HeroList) api.ServerContext
	GetResources(moduleIndex ModuleIndex,
		index api.ResourceCollectionIndex) []api.Resource
	GetTextures() []api.Resource
	Font(fontFamily int, style api.FontStyle, size api.FontSize) *ttf.Font
	FontNames() []string
	ViewHeroes() HeroView
}

// TooManyRequests is an error that is issued if the server receives more data
// than the rendering thread can process.
type TooManyRequests struct {
}

// Error returns "Too many requests"
func (TooManyRequests) Error() string {
	return "Too many requests"
}

// StatusCode returns 503
func (TooManyRequests) StatusCode() int {
	return http.StatusTooManyRequests
}
