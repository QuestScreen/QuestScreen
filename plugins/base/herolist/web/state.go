package web

import (
	"encoding/json"

	"github.com/QuestScreen/QuestScreen/web/controls"

	"github.com/flyx/askew/runtime"

	"github.com/QuestScreen/QuestScreen/plugins/base/shared"
	"github.com/QuestScreen/api/web"
)

// State implements api.ModuleState
type State struct {
	web.ServerState
	web.GroupData
	data shared.HerolistState
}

// NewState creates a new herolist state
func NewState(data json.RawMessage, server web.ServerState, group web.GroupData) (web.ModuleState, error) {
	ret := &State{ServerState: server, GroupData: group}
	return ret, json.Unmarshal(data, &ret.data)
}

// UI generates the herolist widget.
func (s *State) UI() runtime.Component {
	ret := NewWidget(s.data.Global)
	for i := 0; i < s.NumHeroes(); i++ {
		item := controls.NewDropdownItem(false, s.HeroName(i), i)
		item.Selected.Set(s.data.Heroes[i])
		ret.Heroes.Items.Append(item)
	}
	ret.Controller = s
	return ret
}

func (s *State) switchAll() bool {
	s.Fetch(web.Post, "", !s.data.Global, &s.data.Global)
	return s.data.Global
}

func (s *State) switchHero(index int) bool {
	s.Fetch(web.Post, s.HeroID(index), s.data.Heroes[index], &s.data.Heroes[index])
	return s.data.Heroes[index]
}
