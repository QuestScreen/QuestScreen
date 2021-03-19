package background

import (
	"encoding/json"
	"syscall/js"

	"github.com/QuestScreen/QuestScreen/plugins/base/shared"
	"github.com/QuestScreen/api/web"
	"github.com/QuestScreen/api/web/controls"
	"github.com/QuestScreen/api/web/modules"
)

// State implements api.web.ModuleState
type State struct {
	controls.Dropdown
	srv  web.Server
	data shared.BackgroundState
}

// NewState implements modules.Constructor.
func NewState(data json.RawMessage, srv web.Server) (modules.State, error) {
	ret := &State{srv: srv}
	ret.Dropdown.Init(controls.SelectAtMostOne, controls.SelectionIndicator, "")
	for index, item := range srv.GetResources(0) {
		ret.Dropdown.AddItem(item.Name, ret.data.CurIndex == index)
	}
	ret.Dropdown.Controller = ret
	if err := json.Unmarshal(data, &ret.data); err != nil {
		return nil, err
	}
	ret.Dropdown.SetItem(ret.data.CurIndex, true)
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

// ItemClicked handles a click by switching to the clicked background and
// returning true.
func (s *State) ItemClicked(index int) bool {
	s.srv.Fetch(web.Post, "", index, nil)
	return true
}
