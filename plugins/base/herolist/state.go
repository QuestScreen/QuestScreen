package herolist

import (
	"github.com/QuestScreen/QuestScreen/plugins/base/shared"
	"github.com/QuestScreen/api/comms"
	"github.com/QuestScreen/api/groups"
	"github.com/QuestScreen/api/modules"
	"github.com/QuestScreen/api/server"
	"gopkg.in/yaml.v3"
)

type state struct {
	globalVisible bool
	heroVisible   []bool
	heroIDToIndex map[string]int
}

type persistedState struct {
	GlobalVisible bool
	HeroVisible   []string
}

type globalEndpoint struct {
	*state
}

type heroEndpoint struct {
	*state
}

func newState(input *yaml.Node, ctx server.Context,
	ms server.MessageSender) (modules.State, error) {
	h := ctx.ActiveGroup().Heroes()
	s := &state{heroVisible: make([]bool, h.NumHeroes()),
		heroIDToIndex: make(map[string]int)}
	for i := 0; i < h.NumHeroes(); i++ {
		s.heroVisible[i] = true
		s.heroIDToIndex[h.Hero(i).ID()] = i
	}
	if input == nil {
		s.globalVisible = true
	} else {
		var tmp persistedState
		if err := input.Decode(&tmp); err != nil {
			return nil, err
		}
		s.globalVisible = tmp.GlobalVisible
		for i := range s.heroVisible {
			s.heroVisible[i] = false
		}
		for i := range tmp.HeroVisible {
			found := false
			for j := 0; j < h.NumHeroes(); j++ {
				if tmp.HeroVisible[i] == h.Hero(j).ID() {
					found = true
					s.heroVisible[j] = true
					break
				}
			}
			if !found {
				ms.Warning("Unknown hero: \"" + tmp.HeroVisible[i] + "\"")
			}
		}
	}

	return s, nil
}

func (s *state) CreateRendererData(ctx server.Context) interface{} {
	hl := ctx.ActiveGroup().Heroes()
	states := make([]heroData, len(s.heroVisible))
	for i := range states {
		h := hl.Hero(i)
		states[i] = heroData{name: h.Name(), desc: h.Description(),
			visible: s.heroVisible[i]}
	}
	return &fullRequest{heroes: states, global: s.globalVisible}
}

func (s *state) HeroListChanged(ctx server.Context, action groups.HeroChangeAction, heroIndex int) {
	switch action {
	case groups.HeroAdded:
		s.heroVisible = append(s.heroVisible, true)
	case groups.HeroModified:
		break
	case groups.HeroDeleted:
		copy(s.heroVisible[heroIndex:], s.heroVisible[heroIndex+1:])
		s.heroVisible = s.heroVisible[:len(s.heroVisible)-1]
	}
}

func (s *state) visibleHeroesList(ctx server.Context) []string {
	ret := make([]string, 0, len(s.heroVisible))
	hl := ctx.ActiveGroup().Heroes()
	for i := range s.heroVisible {
		if s.heroVisible[i] {
			ret = append(ret, hl.Hero(i).ID())
		}
	}
	return ret
}

// Send returns a structure containing the global flag and a list containing
// boolean flags for each hero
func (s *state) Send(ctx server.Context) interface{} {
	return shared.HerolistState{Global: s.globalVisible, Heroes: s.heroVisible}
}

// Persist returns a structure containing the `global` flag and a list
// containing each visible hero as ID
func (s *state) Persist(ctx server.Context) interface{} {
	return persistedState{GlobalVisible: s.globalVisible,
		HeroVisible: s.visibleHeroesList(ctx)}
}

func (s *state) PureEndpoint(index int) modules.PureEndpoint {
	if index != 0 {
		panic("Endpoint index out of range")
	}
	return globalEndpoint{s}
}

func (s *state) IDEndpoint(index int) modules.IDEndpoint {
	if index != 1 {
		panic("Endpoint index out of range")
	}
	return heroEndpoint{s}
}

func (e globalEndpoint) Post(payload []byte) (interface{}, interface{},
	server.Error) {
	var value bool
	if err := comms.ReceiveData(payload, &value); err != nil {
		return nil, nil, &server.BadRequest{Inner: err, Message: "received invalid data"}
	}
	e.globalVisible = value
	return value, &globalRequest{visible: e.globalVisible}, nil
}

func (e heroEndpoint) Post(id string, payload []byte) (interface{}, interface{},
	server.Error) {
	hIndex, ok := e.heroIDToIndex[id]
	if !ok {
		return nil, nil, &server.NotFound{Name: id}
	}
	var value bool
	if err := comms.ReceiveData(payload, &value); err != nil {
		return nil, nil, &server.BadRequest{Inner: err, Message: "received invalid data"}
	}
	e.heroVisible[hIndex] = value
	return value, &heroRequest{index: int32(hIndex), visible: value}, nil
}
