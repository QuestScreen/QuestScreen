package background

import (
	"encoding/json"

	"github.com/QuestScreen/QuestScreen/plugins/base/shared"
	"github.com/QuestScreen/QuestScreen/web/controls"
	"github.com/QuestScreen/api/web/groups"
	"github.com/QuestScreen/api/web/modules"
	"github.com/QuestScreen/api/web/server"
	"github.com/flyx/askew/runtime"
)

// State implements api.web.ModuleState
type State struct {
	server.State
	data shared.BackgroundState
}

// NewState implements modules.Constructor.
func NewState(data json.RawMessage, srv server.State, group groups.Group) (modules.State, error) {
	ret := &State{State: srv}
	return ret, json.Unmarshal(data, &ret.data)
}

// UI returns a dropdown widget.
func (s *State) UI(srv server.State) runtime.Component {
	ret := controls.NewDropdown(controls.SelectOne, controls.SelectionIndicator)
	for index, item := range s.data.Items {
		ret.AddItem(item, s.data.CurIndex == index)
	}
	ret.Controller = s
	return ret
}

// ItemClicked handles a click by switching to the clicked background and
// returning true.
func (s *State) ItemClicked(index int) bool {
	s.Fetch(server.Post, "", index, nil)
	return true
}
