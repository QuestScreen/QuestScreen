package herolist

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/flyx/pnpscreen/api"
	"gopkg.in/yaml.v3"
)

type state struct {
	globalVisible bool
	heroVisible   []bool
}

type persistedState struct {
	GlobalVisible bool
	HeroVisible   []string
}

func newState(input *yaml.Node, env api.Environment,
	index api.ModuleIndex) (api.ModuleState, error) {
	heroes := env.Heroes()
	defer heroes.Close()
	s := &state{heroVisible: make([]bool, heroes.NumHeroes())}
	for i := 0; i < heroes.NumHeroes(); i++ {
		s.heroVisible[i] = true
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

// Serializable view returns a structure containing the global flag and
// - a list containing each visible hero as ID for Persisted
// - a list containing boolean flags for each hero for Web
func (s *state) SerializableView(
	env api.Environment, layout api.DataLayout) interface{} {
	if layout == api.Persisted {
		return persistedState{GlobalVisible: s.globalVisible,
			HeroVisible: s.visibleHeroesList(env)}
	}
	return webState{
		Global: s.globalVisible, Heroes: s.heroVisible,
	}
}

func (s *state) HandleAction(index int, payload []byte) (interface{}, interface{}, error) {
	var ret bool
	var data interface{}
	switch index {
	case 0:
		s.globalVisible = !s.globalVisible
		ret = s.globalVisible
		data = &globalRequest{visible: s.globalVisible}
	case 1:
		var value int
		if err := json.Unmarshal(payload, &value); err != nil {
			return nil, nil, err
		}
		if value < 0 || value >= len(s.heroVisible) {
			return nil, nil, fmt.Errorf("index %d out of range 0..%d",
				value, len(s.heroVisible)-1)
		}
		s.heroVisible[value] = !s.heroVisible[value]
		ret = s.heroVisible[value]
		data = &heroRequest{index: int32(value), visible: s.heroVisible[value]}
	default:
		panic("Index out of range")
	}
	return ret, data, nil
}
