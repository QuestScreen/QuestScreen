package display

import "github.com/veandco/go-sdl2/sdl"

// Events lists the event IDs used with SDL's UserEvent
type Events struct {
	ModuleUpdateID uint32
	ModuleConfigID uint32
	SceneChangeID  uint32
}

// GenEvents generates a set of event IDs. Only call this once!
func GenEvents() Events {
	var ret Events
	ret.ModuleUpdateID = sdl.RegisterEvents(3)
	ret.ModuleConfigID = ret.ModuleUpdateID + 1
	ret.SceneChangeID = ret.ModuleUpdateID + 2
	return ret
}
