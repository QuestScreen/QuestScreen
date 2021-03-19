package overlays

import (
	"encoding/json"
	"syscall/js"

	"github.com/QuestScreen/api/web"
	"github.com/QuestScreen/api/web/controls"
	"github.com/QuestScreen/api/web/modules"

	"github.com/QuestScreen/QuestScreen/plugins/base/shared"
)

// State implements web.ModuleState
type State struct {
	controls.Dropdown
	srv  web.Server
	data shared.OverlayState
}

// NewState implements modules.Constructor.
func NewState(data json.RawMessage, srv web.Server) (modules.State, error) {
	ret := &State{srv: srv}
	if err := json.Unmarshal(data, &ret.data); err != nil {
		return nil, err
	}
	ret.Dropdown.Init(controls.SelectMultiple, controls.VisibilityIndicator, "Items")
	for _, item := range ret.data {
		ret.Dropdown.AddItem(item.Name, item.Selected)
	}
	ret.Dropdown.Controller = ret
	return ret, nil
}

// Destroy calls the dropdown's Destroy.
func (s *State) Destroy() {
	s.Dropdown.Destroy()
}

// Extract calls the dropdown's Extract.
func (s *State) Extract() {
	s.Dropdown.Extract()
}

// FirstNode calls the dropdown's FirstNode
func (s *State) FirstNode() js.Value {
	return s.Dropdown.FirstNode()
}

// InsertInto calls the dropdown's InsertInto
func (s *State) InsertInto(parent js.Value, before js.Value) {
	s.Dropdown.InsertInto(parent, before)
}

// ItemClicked implements the Dropdown's controller.
func (s *State) ItemClicked(index int) bool {
	s.srv.Fetch(web.Post, "", struct {
		resourceIndex int
		visible       bool
	}{index, s.data[index].Selected},
		&s.data[index].Selected)
	return s.data[index].Selected
}
