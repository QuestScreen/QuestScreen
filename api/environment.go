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

// HeroList describes a list of heroes.
// this type is necessary since a []HeroImpl object cannot be casted to []Hero.
type HeroList interface {
	Item(index int) Hero
	Length() int
}

// Environment describes the global environment.
type Environment interface {
	// GetResources queries the list of available resources of the given
	// resource collection index.
	//
	// The resources are filtered by the currently active system and group.
	GetResources(
		moduleIndex ModuleIndex, index ResourceCollectionIndex) []Resource
	// Heroes returns the list of available heroes. This list depends on the
	// currently selected group and may be nil.
	Heroes() HeroList
	// FontCatalog returns the list of loaded font families.
	FontCatalog() []FontFamily
	// Font is a shorthand to select a specific font from the catalog.
	Font(familyIndex int, style FontStyle, size FontSize) *ttf.Font
	// Returns the default size (in pixels) of a border line.
	DefaultBorderWidth() int32
}

// ResourceNames generates a list of resource names from a list of resources.
func ResourceNames(resources []Resource) []string {
	ret := make([]string, len(resources))
	for i := range resources {
		ret[i] = resources[i].Name()
	}
	return ret
}
