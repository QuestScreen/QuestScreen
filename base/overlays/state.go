package overlays

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/flyx/pnpscreen/api"
)

type state struct {
	shared    *sharedData
	visible   []bool
	resources []api.Resource
}

func newState(yamlSubtree interface{}, env api.Environment,
	shared *sharedData) (*state, error) {
	s := new(state)
	s.resources = env.GetResources(shared.moduleIndex, 0)
	s.visible = make([]bool, len(s.resources))
	s.shared = shared
	if yamlSubtree != nil {
		names, ok := yamlSubtree.([]interface{})
		if !ok {
			return nil, errors.New("value of Persons is not a sequence")
		}
		for i := range names {
			name, ok := names[i].(string)
			if !ok {
				return nil, errors.New("item in Persons is not a string")
			}
			found := false
			for j := range s.resources {
				if s.resources[j].Name() == name {
					found = true
					s.visible[j] = true
					break
				}
			}
			if !found {
				log.Println("Did not find resource \"" + name + "\"")
			}
		}
	}

	return s, nil
}

func (s *state) SendToModule() {
	visible := make([]bool, len(s.visible))
	copy(visible, s.visible)
	resources := make([]api.Resource, len(s.resources))
	copy(resources, s.resources)
	s.shared.mutex.Lock()
	s.shared.kind = stateRequest
	s.shared.state = visible
	s.shared.resources = resources
	s.shared.mutex.Unlock()
}

// ToYAML returns a slice containing the names of all visible items.
func (s *state) ToYAML(env api.Environment) interface{} {
	ret := make([]string, 0, len(s.resources))
	for i := range s.visible {
		if s.visible[i] {
			ret = append(ret, s.resources[i].Name())
		}
	}
	return ret
}

type jsonStateItem struct {
	Name     string `json:"name"`
	Selected bool   `json:"selected"`
}

type jsonState []jsonStateItem

// ToJSON returns a list of resource names together with a flag telling whether
// the resource is visible
func (s *state) ToJSON() interface{} {
	ret := make(jsonState, len(s.resources))
	for i := range s.resources {
		ret[i].Name = s.resources[i].Name()
		ret[i].Selected = s.visible[i]
	}
	return ret
}

func (*state) Actions() []string {
	return []string{"switch"}
}

func (s *state) HandleAction(index int, payload []byte) (interface{}, error) {
	if index != 0 {
		panic("Index out of range")
	}
	var value int
	if err := json.Unmarshal(payload, &value); err != nil {
		return nil, err
	}
	if value < 0 || value >= len(s.resources) {
		return nil, fmt.Errorf("Index %d not in range 0..%d",
			value, len(s.resources)-1)
	}
	s.shared.mutex.Lock()
	defer s.shared.mutex.Unlock()
	if s.shared.kind != noRequest {
		return nil, errors.New("Too many requests")
	}
	s.visible[value] = !s.visible[value]
	s.shared.kind = itemRequest
	s.shared.itemIndex = value
	s.shared.itemShown = s.visible[value]
	return s.visible[value], nil
}
