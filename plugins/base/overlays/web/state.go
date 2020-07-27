package web

import (
	"encoding/json"

	"github.com/QuestScreen/QuestScreen/web/controls"

	"github.com/QuestScreen/QuestScreen/plugins/base/shared"
	"github.com/QuestScreen/api/web"
	"github.com/flyx/askew/runtime"
)

// State implements web.ModuleState
type State struct {
	web.ServerState
	data shared.OverlayState
}

// NewState creates a new overlays state.
func NewState(data json.RawMessage, server web.ServerState, group web.GroupData) (web.ModuleState, error) {
	ret := &State{ServerState: server}
	return ret, json.Unmarshal(data, &ret.data)
}

// UI creates a dropdown as UI for the overlays module.
func (s *State) UI() runtime.Component {
	ret := controls.NewDropdown(controls.SelectMultiple)
	for index, item := range s.data {
		w := controls.NewDropdownItem(true, item.Name, index)
		w.Selected.Set(item.Selected)
		ret.Items.Append(w)
	}
	ret.Controller = s
	return ret
}

// ItemClicked implements the Dropdown's controller.
func (s *State) ItemClicked(index int) bool {
	s.Fetch(web.Post, "", struct {
		resourceIndex int
		visible       bool
	}{index, s.data[index].Selected},
		&s.data[index].Selected)
	return s.data[index].Selected
}
