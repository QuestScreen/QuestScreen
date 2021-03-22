package herolist

import (
	"encoding/json"

	"github.com/QuestScreen/api/web"
	"github.com/QuestScreen/api/web/modules"
)

// NewState implements modules.Constructor.
func NewState(data json.RawMessage, srv web.Server) (modules.State, error) {
	ret := &State{srv: srv}

	if err := json.Unmarshal(data, &ret.data); err != nil {
		return nil, err
	}

	ret.askewInit(ret.data.Global)
	heroes := srv.ActiveGroup().Heroes()
	for i := 0; i < heroes.NumHeroes(); i++ {
		hero := heroes.Hero(i)
		ret.Heroes.AddItem(hero.Name(), ret.data.Heroes[i])
	}
	return ret, nil
}

func (s *State) switchAll() bool {
	s.srv.Fetch(web.Post, "", !s.data.Global, &s.data.Global)
	return s.data.Global
}

func (s *State) switchHero(index int) bool {
	s.srv.Fetch(web.Post, s.srv.ActiveGroup().Heroes().Hero(index).ID(),
		!s.data.Heroes[index], &s.data.Heroes[index])
	return s.data.Heroes[index]
}

func (s *State) allClicked() {
	go func() {
		s.allState.Set(s.switchAll())
	}()
}

// ItemClicked implements the controller of controls.Dropdown
func (s *State) ItemClicked(index int) bool {
	return s.switchHero(index)
}
