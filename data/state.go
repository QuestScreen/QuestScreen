package data

import (
	"sync"

	"github.com/QuestScreen/QuestScreen/app"
	"github.com/QuestScreen/QuestScreen/shared"
	"github.com/QuestScreen/api/modules"
	"github.com/QuestScreen/api/server"
)

// State holds the complete state for the currently active group.
// whenever the current group changes, state must be reloaded from the
// state.yaml of the new group.
type State struct {
	activeScene int
	scenes      [][]modules.State
	path        string
	writeMutex  sync.Mutex
	a           app.App
	group       Group
}

// SetScene sets the scene index.
func (s *State) SetScene(index int) server.Error {
	if index < 0 || index >= len(s.scenes) {
		return &server.BadRequest{Message: "index out of range"}
	}
	s.activeScene = index
	return nil
}

// ActiveScene returns the index of the currently active scene
func (s *State) ActiveScene() int {
	return s.activeScene
}

// StateOf returns the module state for the given module in the active scene
func (s *State) StateOf(moduleIndex shared.ModuleIndex) modules.State {
	return s.scenes[s.activeScene][moduleIndex]
}

// StateOfScene returns the module state for the given module in the given
// scene
func (s *State) StateOfScene(
	sceneIndex int, moduleIndex shared.ModuleIndex) modules.State {
	return s.scenes[sceneIndex][moduleIndex]
}
