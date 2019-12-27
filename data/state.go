package data

import (
	"errors"
	"io/ioutil"
	"log"
	"sync"

	"github.com/flyx/pnpscreen/api"
	"github.com/flyx/pnpscreen/app"
)

// GroupState holds the complete state for the currently active group.
// whenever the current group changes, state must be reloaded from the
// state.yaml of the new group.
type GroupState struct {
	activeScene int
	scenes      [][]api.ModuleState
	path        string
	writeMutex  sync.Mutex
	a           app.App
	group       Group
}

// SetScene sets the scene index.
func (gs *GroupState) SetScene(index int) error {
	if index < 0 || index >= len(gs.scenes) {
		return errors.New("index out of range")
	}
	gs.activeScene = index
	return nil
}

// ActiveScene returns the index of the currently active scene
func (gs *GroupState) ActiveScene() int {
	return gs.activeScene
}

// State returns the module state for the given module in the active scene
func (gs *GroupState) State(moduleIndex api.ModuleIndex) api.ModuleState {
	return gs.scenes[gs.activeScene][moduleIndex]
}

// Persist writes the group state to its YAML file.
// The actual writing operation is done asynchronous.
func (gs *GroupState) Persist() {
	raw, err := gs.buildYaml()
	if err != nil {
		log.Printf("%s[w]: %s", gs.path, err)
	} else {
		go func(content []byte, gs *GroupState) {
			gs.writeMutex.Lock()
			defer gs.writeMutex.Unlock()
			ioutil.WriteFile(gs.path, content, 0644)
		}(raw, gs)
	}
}
