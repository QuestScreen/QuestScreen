package api

import (
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

// the interfaces declared in this file are implemented by the QuestScreen core.

// Resource describes a selectable resource (typically a file).
type Resource interface {
	// Name of the file as it should be presented to the user.
	Name() string
	// Absolute path to the file.
	Path() string
}

// Hero describes a hero (player character).
type Hero interface {
	// Name of the hero
	Name() string
	// ID of the hero
	ID() string
	// Short description (e.g. class/race/etc)
	Description() string
}

// HeroList describes the list of heroes.
type HeroList interface {
	Hero(index int) Hero
	NumHeroes() int
}

// ResourceProvider is the interface to files on the file system that have been
// selected by a module's ResourceSelectors. Resources are read-only and
// available on both server and display thread.
type ResourceProvider interface {
	// GetResources queries the list of available resources of the given
	// resource collection index.
	//
	// The resources are filtered by the currently active system, group and scene.
	// Each Resource object is read-only and may be freely shared between threads.
	GetResources(index ResourceCollectionIndex) []Resource
}

// ServerContext gives access to data available in the server thread.
// This is a read-only view of data required for serialization and state
// initialization.
//
// Details on Fonts and Heroes are available in the display thread via
// [Extended]RenderContext.
type ServerContext interface {
	ResourceProvider
	NumFontFamilies() int
	FontFamilyName(index int) string
	NumHeroes() int
	HeroID(index int) string
}

// RenderContext is the context given to all rendering funcs of a module
type RenderContext interface {
	ResourceProvider
	Renderer() *sdl.Renderer
	// Font returns the font face of the selected font.
	Font(fontFamily int, style FontStyle, size FontSize) *ttf.Font
	// DefaultBorderWidth returns the default size (in pixels) of a border line.
	DefaultBorderWidth() int32
}

// ExtendedRenderContext is the context used for rebuilding the whole module
// and may contain additional data depending on the module's description.
type ExtendedRenderContext interface {
	RenderContext
	// Heroes returns a non-null list iff the module's description has UseHeroes set.
	Heroes() HeroList
}

// ResourceNames generates a list of resource names from a list of resources.
func ResourceNames(resources []Resource) []string {
	ret := make([]string, len(resources))
	for i := range resources {
		ret[i] = resources[i].Name()
	}
	return ret
}
