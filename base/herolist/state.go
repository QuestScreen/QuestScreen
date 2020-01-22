package herolist

import (
	"log"

	"github.com/flyx/pnpscreen/api"
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

func newState(input *yaml.Node, env api.Environment,
	index api.ModuleIndex) (api.ModuleState, error) {
	heroes := env.Heroes()
	defer heroes.Close()
	s := &state{heroVisible: make([]bool, heroes.NumHeroes()),
		heroIDToIndex: make(map[string]int)}
	for i := 0; i < heroes.NumHeroes(); i++ {
		s.heroVisible[i] = true
		s.heroIDToIndex[heroes.Hero(i).ID()] = i
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
			for j := 0; j < heroes.NumHeroes(); j++ {
				if tmp.HeroVisible[i] == heroes.Hero(j).Name() {
					found = true
					s.heroVisible[j] = true
					break
				}
			}
			if !found {
				log.Println("Unknown hero: \"" + tmp.HeroVisible[i] + "\"")
			}
		}
	}

	return s, nil
}

func (s *state) CreateModuleData() interface{} {
	states := make([]bool, len(s.heroVisible))
	copy(states, s.heroVisible)
	return &fullRequest{heroes: states, global: s.globalVisible}
}

func (s *state) visibleHeroesList(env api.Environment) []string {
	ret := make([]string, 0, len(s.heroVisible))
	heroes := env.Heroes()
	defer heroes.Close()
	for i := range s.heroVisible {
		if s.heroVisible[i] {
			ret = append(ret, heroes.Hero(i).Name())
		}
	}
	return ret
}

type webState struct {
	Global bool   `json:"global"`
	Heroes []bool `json:"heroes"`
}

// WebView returns a structure containing the global flag and a list containing
// boolean flags for each hero
func (s *state) WebView(env api.Environment) interface{} {
	return webState{Global: s.globalVisible, Heroes: s.heroVisible}
}

// PersistingView returns a structure containing the `global` flag and a list
// containing each visible hero as ID
func (s *state) PersistingView(env api.Environment) interface{} {
	return persistedState{GlobalVisible: s.globalVisible,
		HeroVisible: s.visibleHeroesList(env)}
}

func (s *state) PureEndpoint(index int) api.ModulePureEndpoint {
	if index != 0 {
		panic("Endpoint index out of range")
	}
	return globalEndpoint{s}
}

func (s *state) IDEndpoint(index int) api.ModuleIDEndpoint {
	if index != 1 {
		panic("Endpoint index out of range")
	}
	return heroEndpoint{s}
}

func (e globalEndpoint) Put(payload []byte) (interface{}, interface{},
	api.SendableError) {
	var value bool
	if err := api.ReceiveData(payload, &value); err != nil {
		return nil, nil, err
	}
	e.globalVisible = value
	return value, &globalRequest{visible: e.globalVisible}, nil
}

func (e heroEndpoint) Put(id string, payload []byte) (interface{}, interface{},
	api.SendableError) {
	hIndex, ok := e.heroIDToIndex[id]
	if !ok {
		return nil, nil, &api.NotFound{Name: id}
	}
	var value bool
	if err := api.ReceiveData(payload, &value); err != nil {
		return nil, nil, err
	}
	e.heroVisible[hIndex] = value
	return value, &heroRequest{index: int32(hIndex), visible: value}, nil
}
