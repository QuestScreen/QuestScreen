package web

import (
	"encoding/json"

	"github.com/QuestScreen/QuestScreen/web/controls"

	"github.com/flyx/askew/runtime"

	"github.com/QuestScreen/QuestScreen/plugins/base/shared"
	"github.com/QuestScreen/api/web/groups"
	"github.com/QuestScreen/api/web/modules"
	"github.com/QuestScreen/api/web/server"
)

// State implements api.ModuleState
type State struct {
	server.State
	groups.Group
	data shared.HerolistState
}

// NewState implements modules.Constructor.
func NewState(data json.RawMessage, srv server.State, group groups.Group) (modules.State, error) {
	ret := &State{State: srv, Group: group}
	return ret, json.Unmarshal(data, &ret.data)
}

// UI generates the herolist widget.
func (s *State) UI(srv server.State) runtime.Component {
	ret := NewWidget(s.data.Global)
	heroes := s.Heroes()
	for i := 0; i < heroes.NumHeroes(); i++ {
		hero := heroes.Hero(i)
		item := controls.NewDropdownItem(false, hero.Name(), i)
		item.Selected.Set(s.data.Heroes[i])
		ret.Heroes.Items.Append(item)
	}
	ret.Controller = s
	return ret
}

func (s *State) switchAll() bool {
	s.Fetch(server.Post, "", !s.data.Global, &s.data.Global)
	return s.data.Global
}

func (s *State) switchHero(index int) bool {
	s.Fetch(server.Post, s.Heroes().Hero(index).ID(), s.data.Heroes[index], &s.data.Heroes[index])
	return s.data.Heroes[index]
}
