package overlays

import (
	"encoding/json"

	"github.com/QuestScreen/QuestScreen/web/controls"
	"github.com/QuestScreen/api/web/groups"
	"github.com/QuestScreen/api/web/modules"
	"github.com/QuestScreen/api/web/server"

	"github.com/QuestScreen/QuestScreen/plugins/base/shared"
	"github.com/flyx/askew/runtime"
)

// State implements web.ModuleState
type State struct {
	server.State
	data shared.OverlayState
}

// NewState implements modules.Constructor.
func NewState(data json.RawMessage, srv server.State, group groups.Group) (modules.State, error) {
	ret := &State{State: srv}
	return ret, json.Unmarshal(data, &ret.data)
}

// UI creates a dropdown as UI for the overlays module.
func (s *State) UI(srv server.State) runtime.Component {
	ret := controls.NewDropdown(controls.SelectMultiple, controls.VisibilityIndicator)
	for _, item := range s.data {
		ret.AddItem(item.Name, item.Selected)
	}
	ret.Controller = s
	return ret
}

// ItemClicked implements the Dropdown's controller.
func (s *State) ItemClicked(index int) bool {
	s.Fetch(server.Post, "", struct {
		resourceIndex int
		visible       bool
	}{index, s.data[index].Selected},
		&s.data[index].Selected)
	return s.data[index].Selected
}
