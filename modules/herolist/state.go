package herolist

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"

	"github.com/flyx/pnpscreen/data"
)

type state struct {
	owner         *HeroList
	globalVisible bool
	heroVisible   []bool
}

func (s *state) LoadFrom(yamlSubtree interface{}, store *data.Store) error {
	s.heroVisible = make([]bool, store.NumHeroes(store.GetActiveGroup()))
	for i := 0; i < store.NumHeroes(store.GetActiveGroup()); i++ {
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
				for i := 0; i < store.NumHeroes(store.GetActiveGroup()); i++ {
					s.heroVisible[i] = false
				}
				sequence, ok := v.([]interface{})
				if !ok {
					return errors.New(
						"unexpected value for heroesVisible: not a list of strings")
				}
				for i := range sequence {
					found := false
					for j := 0; j < store.NumHeroes(store.GetActiveGroup()); j++ {
						if sequence[i].(string) == store.HeroName(store.GetActiveGroup(), j) {
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

	heroes := make([]bool, len(s.heroVisible))
	copy(heroes, s.heroVisible)
	s.owner.requests.mutex.Lock()
	defer s.owner.requests.mutex.Unlock()
	s.owner.requests.globalVisible = s.globalVisible
	s.owner.requests.heroes = heroes
	s.owner.requests.kind = stateRequest
	return nil
}

func (s *state) visibleHeroesList(store *data.Store) []string {
	ret := make([]string, 0, len(s.heroVisible))
	for i := range s.heroVisible {
		if s.heroVisible[i] {
			ret = append(ret, store.HeroName(store.GetActiveGroup(), i))
		}
	}
	return ret
}

// ToYAML returns a mapping representing the state, where each
// hero is identified by its name.
func (s *state) ToYAML(store *data.Store) interface{} {
	return map[string]interface{}{
		"globalVisible": s.globalVisible,
		"heroVisible":   s.visibleHeroesList(store),
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

func (s *state) HandleAction(index int, payload []byte, store *data.Store) ([]byte, error) {
	s.owner.requests.mutex.Lock()
	defer s.owner.requests.mutex.Unlock()
	if s.owner.requests.kind != noRequest {
		return nil, errors.New("Too many requests")
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
			return nil, fmt.Errorf("Index %d out of range 0..%d",
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
