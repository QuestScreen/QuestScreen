package persons

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/flyx/pnpscreen/data"
)

type state struct {
	owner     *Persons
	visible   []bool
	resources []data.Resource
}

func (s *state) LoadFrom(yamlSubtree interface{}, store *data.Store) error {
	s.resources = store.ListFiles(s.owner, "")
	s.visible = make([]bool, len(s.resources))
	if yamlSubtree != nil {
		names, ok := yamlSubtree.([]interface{})
		if !ok {
			return errors.New("value of Persons is not a sequence")
		}
		for i := range names {
			name, ok := names[i].(string)
			if !ok {
				return errors.New("item in Persons is not a string")
			}
			found := false
			for j := range s.resources {
				if s.resources[j].Name == name {
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
	visible := make([]bool, len(s.visible))
	copy(visible, s.visible)
	s.owner.requests.mutex.Lock()
	s.owner.requests.kind = stateRequest
	s.owner.requests.state = visible
	s.owner.requests.mutex.Unlock()
	return nil
}

// ToYAML returns a slice containing the names of all visible items.
func (s *state) ToYAML(store *data.Store) interface{} {
	ret := make([]string, 0, len(s.resources))
	for i := range s.visible {
		if s.visible[i] {
			ret = append(ret, s.resources[i].Name)
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
		ret[i].Name = s.resources[i].Name
		ret[i].Selected = s.visible[i]
	}
	return ret
}

func (*state) Actions() []string {
	return []string{"switch"}
}

func (s *state) HandleAction(index int, payload []byte, store *data.Store) ([]byte, error) {
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
	s.owner.requests.mutex.Lock()
	defer s.owner.requests.mutex.Unlock()
	if s.owner.requests.kind != noRequest {
		return nil, errors.New("Too many requests")
	}
	s.visible[value] = !s.visible[value]
	s.owner.requests.kind = itemRequest
	s.owner.requests.itemIndex = value
	s.owner.requests.itemShown = s.visible[value]
	return json.Marshal(s.visible[value])
}
