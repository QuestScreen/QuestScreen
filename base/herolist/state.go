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
	owner         *HeroList
	globalVisible bool
	heroVisible   []bool
}

func (s *state) LoadFrom(yamlSubtree interface{}, env api.Environment) error {
	heroes := env.Heroes()
	s.heroVisible = make([]bool, heroes.Length())
	for i := 0; i < heroes.Length(); i++ {
		s.heroVisible[i] = true
	}
	if yamlSubtree == nil {
		s.globalVisible = true
	} else {
		mapping, ok := yamlSubtree.(map[string]interface{})
		if !ok {
			return errors.New("unexpected state for ShowableHeroes: not a mapping, but a " +
				reflect.TypeOf(yamlSubtree).Name())
		}
		for k, v := range mapping {
			switch k {
			case "globalVisible":
				boolean, ok := v.(bool)
				if !ok {
					return errors.New("unexpected value for globalVisible: not a bool")
				}
				s.globalVisible = boolean
			case "heroVisible":
				for i := 0; i < heroes.Length(); i++ {
					s.heroVisible[i] = false
				}
				sequence, ok := v.([]interface{})
				if !ok {
					return errors.New(
						"unexpected value for heroesVisible: not a list of strings")
				}
				for i := range sequence {
					found := false
					for j := 0; j < heroes.Length(); j++ {
						if sequence[i].(string) == heroes.Item(j).Name() {
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

	states := make([]bool, len(s.heroVisible))
	copy(states, s.heroVisible)
	s.owner.requests.mutex.Lock()
	defer s.owner.requests.mutex.Unlock()
	s.owner.requests.globalVisible = s.globalVisible
	s.owner.requests.heroes = states
	s.owner.requests.kind = stateRequest
	return nil
}

func (s *state) visibleHeroesList(env api.Environment) []string {
	ret := make([]string, 0, len(s.heroVisible))
	for i := range s.heroVisible {
		if s.heroVisible[i] {
			ret = append(ret, env.Heroes().Item(i).Name())
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

func (*state) Actions() []string {
	return []string{"switchGlobal", "switchHero"}
}

func (s *state) HandleAction(index int, payload []byte) ([]byte, error) {
	s.owner.requests.mutex.Lock()
	defer s.owner.requests.mutex.Unlock()
	if s.owner.requests.kind != noRequest {
		return nil, errors.New("too many requests")
	}

	var ret bool
	switch index {
	case 0:
		s.globalVisible = !s.globalVisible
		ret = s.globalVisible
		s.owner.requests.kind = globalRequest
		s.owner.requests.globalVisible = s.globalVisible
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
		s.owner.requests.kind = heroRequest
		s.owner.requests.heroIndex = int32(value)
		s.owner.requests.heroVisible = s.heroVisible[value]
	default:
		panic("Index out of range")
	}
	return json.Marshal(ret)
}
