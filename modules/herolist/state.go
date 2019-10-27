package herolist

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/flyx/rpscreen/data"
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
			return errors.New("unexpected state for ShowableHeroes: not a mapping")
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
				sequence, ok := v.([]string)
				if !ok {
					return errors.New(
						"unexpected value for heroesVisible: not a list of strings")
				}
				for i := range sequence {
					found := false
					for j := 0; j < store.NumHeroes(store.GetActiveGroup()); j++ {
						if sequence[i] == store.HeroName(store.GetActiveGroup(), j) {
							found = true
							s.heroVisible[j] = true
							break
						}
					}
					if !found {
						log.Println("Unknown hero: \"" + sequence[i] + "\"")
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

// ToJSON returns the item itself since it can be serialized as-is.
func (s *state) ToJSON() interface{} {
	return s
}

func (*state) Actions() []string {
	return []string{"switchGlobal", "switchHero"}
}

func (s *state) HandleAction(index int, payload []byte, store *data.Store) error {
	s.owner.requests.mutex.Lock()
	defer s.owner.requests.mutex.Unlock()
	if s.owner.requests.kind != noRequest {
		return errors.New("Too many requests")
	}

	switch index {
	case 0:
		s.globalVisible = !s.globalVisible
		s.owner.requests.kind = globalRequest
		s.owner.requests.globalVisible = s.globalVisible
	case 1:
		var index int
		if err := json.Unmarshal(payload, &index); err != nil {
			return err
		}
		if index < 0 || index >= len(s.heroVisible) {
			return fmt.Errorf("Index %d out of range 0..%d",
				index, len(s.heroVisible)-1)
		}
		s.heroVisible[index] = !s.heroVisible[index]
		s.owner.requests.heroIndex = int32(index)
		s.owner.requests.heroVisible = s.heroVisible[index]
	default:
		panic("Index out of range")
	}
	return nil
}
