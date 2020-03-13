package display

import "github.com/veandco/go-sdl2/sdl"

// Events lists the event IDs used with SDL's UserEvent
type Events struct {
	// issued for updates to the module state via transitioning
	ModuleUpdateID uint32
	// issued for updates of module configuration
	ModuleConfigID uint32
	// issued when changing the current scene
	SceneChangeID uint32
	// issued when the list of heroes is edited
	HeroesChangedID uint32
	// issued when the user leaves the current group
	// (resets display to welcome screen)
	LeaveGroupID uint32
}

// GenEvents generates a set of event IDs. Only call this once!
func GenEvents() Events {
	var ret Events
	ret.ModuleUpdateID = sdl.RegisterEvents(5)
	ret.ModuleConfigID = ret.ModuleUpdateID + 1
	ret.SceneChangeID = ret.ModuleUpdateID + 2
	ret.HeroesChangedID = ret.ModuleUpdateID + 3
	ret.LeaveGroupID = ret.ModuleUpdateID + 4
	return ret
}
