package data

import (
	"errors"

	"github.com/flyx/pnpscreen/api"
)

// GroupState holds the complete state for the currently active group.
// whenever the current group changes, state must be reloaded from the
// state.yaml of the new group.
type GroupState struct {
	activeScene int
	scenes      [][]api.ModuleState
}

// SetScene sets the scene index.
func (g *GroupState) SetScene(index int) error {
	if index < 0 || index >= len(g.scenes) {
		return errors.New("index out of range")
	}
	g.activeScene = index
	for j := range g.scenes[index] {
		state := g.scenes[index][j]
		if state != nil {
			state.SendToModule()
		}
	}
	return nil
}

// ActiveScene returns the index of the currently active scene
func (g *GroupState) ActiveScene() int {
	return g.activeScene
}

// State returns the module state for the given module in the active scene
func (g *GroupState) State(moduleIndex api.ModuleIndex) api.ModuleState {
	return g.scenes[g.activeScene][moduleIndex]
}
