package api

import "github.com/veandco/go-sdl2/ttf"

// the interfaces declared in this file are implemented by the pnpscreen core.

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
	// Short description (e.g. class/race/etc)
	Description() string
}

// HeroView describes a view on a list of heroes.
// This view is exclusive and must be closed via Close() after requesting it!
// The exclusiveness ensures that no data races happen between the threads since
// both server and render thread may access the heroes.
type HeroView interface {
	Hero(index int) Hero
	NumHeroes() int
	Close()
}

// StaticEnvironment describes the part of the environment that does not depend
// on its modules or the selected group or system.
type StaticEnvironment interface {
	// FontCatalog returns the list of loaded font families. Safe for the access
	// to the *ttf.Font objects, this list is read-only after app startup and
	// therefore may be safely used in any thread. The *ttf.Font objects are only
	// to be used in the OpenGL thread.
	FontCatalog() []FontFamily
	// Returns the default size (in pixels) of a border line.
	// This is read-only after app startup and may be safely used in any thread.
	DefaultBorderWidth() int32
}

// Environment describes the global environment.
type Environment interface {
	StaticEnvironment
	// GetResources queries the list of available resources of the given
	// resource collection index.
	//
	// The resources are filtered by the currently active system and group.
	// Each Resource object is read-only and may be freely shared between threads.
	GetResources(
		moduleIndex ModuleIndex, index ResourceCollectionIndex) []Resource
	// Heroes returns a view of available heroes. This viewdepends on the
	// currently selected group and may be empty.
	// Requesting a HeroView may be blocking and the returned view must be closed
	// after usage to unblock other threads from accessing the heroes.
	Heroes() HeroView
	// Font is a shorthand to select a specific font from the catalog.
	// This func may only be called in the OpenGL thread.
	Font(familyIndex int, style FontStyle, size FontSize) *ttf.Font
}

// ResourceNames generates a list of resource names from a list of resources.
func ResourceNames(resources []Resource) []string {
	ret := make([]string, len(resources))
	for i := range resources {
		ret[i] = resources[i].Name()
	}
	return ret
}
