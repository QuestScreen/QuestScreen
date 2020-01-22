package data

import (
	"sync"

	"github.com/flyx/pnpscreen/api"
	"github.com/flyx/pnpscreen/app"
)

// State holds the complete state for the currently active group.
// whenever the current group changes, state must be reloaded from the
// state.yaml of the new group.
type State struct {
	activeScene int
	scenes      [][]api.ModuleState
	path        string
	writeMutex  sync.Mutex
	a           app.App
	group       Group
}

// SetScene sets the scene index.
func (s *State) SetScene(index int) api.SendableError {
	if index < 0 || index >= len(s.scenes) {
		return &api.BadRequest{Message: "index out of range"}
	}
	s.activeScene = index
	return nil
}

// ActiveScene returns the index of the currently active scene
func (s *State) ActiveScene() int {
	return s.activeScene
}

// StateOf returns the module state for the given module in the active scene
func (s *State) StateOf(moduleIndex api.ModuleIndex) api.ModuleState {
	return s.scenes[s.activeScene][moduleIndex]
}
