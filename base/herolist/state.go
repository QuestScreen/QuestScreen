package herolist

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"

	"github.com/flyx/pnpscreen/api"
)

type state struct {
	shared        *sharedData
	globalVisible bool
	heroVisible   []bool
}

func newState(yamlSubtree interface{}, env api.Environment,
	shared *sharedData) (*state, error) {
	heroes := env.Heroes()
	defer heroes.Close()
	s := new(state)
	s.heroVisible = make([]bool, heroes.NumHeroes())
	s.shared = shared
	for i := 0; i < heroes.NumHeroes(); i++ {
		s.heroVisible[i] = true
	}
	if yamlSubtree == nil {
		s.globalVisible = true
	} else {
		mapping, ok := yamlSubtree.(map[string]interface{})
		if !ok {
			return nil, errors.New(
				"unexpected state for ShowableHeroes: not a mapping, but a " +
					reflect.TypeOf(yamlSubtree).Name())
		}
		for k, v := range mapping {
			switch k {
			case "globalVisible":
				boolean, ok := v.(bool)
				if !ok {
					return nil,
						errors.New("unexpected value for globalVisible: not a bool")
				}
				s.globalVisible = boolean
			case "heroVisible":
				for i := 0; i < heroes.NumHeroes(); i++ {
					s.heroVisible[i] = false
				}
				sequence, ok := v.([]interface{})
				if !ok {
					return nil, errors.New(
						"unexpected value for heroesVisible: not a list of strings")
				}
				for i := range sequence {
					found := false
					for j := 0; j < heroes.NumHeroes(); j++ {
						if sequence[i].(string) == heroes.Hero(j).Name() {
							found = true
							s.heroVisible[j] = true
							break
						}
					}
					if !found {
						log.Println("Unknown hero: \"" + sequence[i].(string) + "\"")
					}
				}
			}
		}
	}

	return s, nil
}

func (s *state) SendToModule() {
	states := make([]bool, len(s.heroVisible))
	copy(states, s.heroVisible)
	s.shared.mutex.Lock()
	defer s.shared.mutex.Unlock()
	s.shared.globalVisible = s.globalVisible
	s.shared.heroes = states
	s.shared.kind = stateRequest
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

// ToYAML returns a mapping representing the state, where each
// hero is identified by its name.
func (s *state) ToYAML(env api.Environment) interface{} {
	return map[string]interface{}{
		"globalVisible": s.globalVisible,
		"heroVisible":   s.visibleHeroesList(env),
	}
}

type jsonState struct {
	Global bool   `json:"global"`
	Heroes []bool `json:"heroes"`
}

// ToJSON returns an object with global and hero visibility.
func (s *state) ToJSON() interface{} {
	return jsonState{
		Global: s.globalVisible, Heroes: s.heroVisible,
	}
}

func (s *state) HandleAction(index int, payload []byte) (interface{}, error) {
	s.shared.mutex.Lock()
	defer s.shared.mutex.Unlock()
	if s.shared.kind != noRequest {
		return nil, errors.New("too many requests")
	}

	var ret bool
	switch index {
	case 0:
		s.globalVisible = !s.globalVisible
		ret = s.globalVisible
		s.shared.kind = globalRequest
		s.shared.globalVisible = s.globalVisible
	case 1:
		var value int
		if err := json.Unmarshal(payload, &value); err != nil {
			return nil, err
		}
		if value < 0 || value >= len(s.heroVisible) {
			return nil, fmt.Errorf("index %d out of range 0..%d",
				value, len(s.heroVisible)-1)
		}
		s.heroVisible[value] = !s.heroVisible[value]
		ret = s.heroVisible[value]
		s.shared.kind = heroRequest
		s.shared.heroIndex = int32(value)
		s.shared.heroVisible = s.heroVisible[value]
	default:
		panic("Index out of range")
	}
	return ret, nil
}
