package web

import (
	"encoding/json"

	"github.com/QuestScreen/QuestScreen/plugins/base/shared"
	"github.com/QuestScreen/QuestScreen/web/controls"
	"github.com/QuestScreen/api/web"
	"github.com/flyx/askew/runtime"
)

// State implements api.web.ModuleState
type State struct {
	web.ServerState
	data shared.BackgroundState
}

// NewState creates a new background state.
func NewState(data json.RawMessage, server web.ServerState, group web.GroupData) (web.ModuleState, error) {
	ret := &State{ServerState: server}
	return ret, json.Unmarshal(data, &ret.data)
}

// UI returns a dropdown widget.
func (s *State) UI() runtime.Component {
	ret := controls.NewDropdown(controls.SelectSingle)
	for index, item := range s.data.Items {
		ret.Items.Append(controls.NewDropdownItem(true, item, index))
	}
	ret.Controller = s
	return ret
}

// ItemClicked handles a click by switching to the clicked background and
// returning true.
func (s *State) ItemClicked(index int) bool {
	s.Fetch(web.Post, "", index, nil)
	return true
}
