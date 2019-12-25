package background

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/flyx/pnpscreen/api"
)

type state struct {
	shared    *sharedData
	curIndex  int
	resources []api.Resource
}

// LoadFrom loads the stored selection, defaults to no item being selected.
func newState(yamlSubtree interface{}, env api.Environment,
	shared *sharedData) (*state, error) {
	s := new(state)
	s.shared = shared
	s.resources = env.GetResources(shared.moduleIndex, 0)
	s.curIndex = -1
	if yamlSubtree != nil {
		scalar, ok := yamlSubtree.(string)
		if !ok {
			return nil, errors.New("unexpected YAML for Background state: not a string")
		}
		for i := range s.resources {
			if s.resources[i].Name() == scalar {
				s.curIndex = i
				break
			}
		}
		if s.curIndex == -1 {
			log.Println("Didn't find resource \"" + scalar + "\"")
		}
	}
	return s, nil
}

func (s *state) SendToModule() {
	s.shared.mutex.Lock()
	s.shared.activeRequest = true
	if s.curIndex == -1 {
		s.shared.file = nil
	} else {
		s.shared.file = s.resources[s.curIndex]
	}
	s.shared.mutex.Unlock()
}

// ToYAML returns the name of the currently selected resource, or nil if none
func (s *state) ToYAML(env api.Environment) interface{} {
	if s.curIndex == -1 {
		return nil
	}
	return s.resources[s.curIndex].Name()
}

type jsonState struct {
	CurIndex int      `json:"curIndex"`
	Items    []string `json:"items"`
}

// ToJSON returns the index of the current item (-1 if none)
func (s *state) ToJSON() interface{} {
	return jsonState{CurIndex: s.curIndex, Items: api.ResourceNames(s.resources)}
}

func (s *state) HandleAction(index int, payload []byte) (interface{}, error) {
	if index != 0 {
		panic("Index out of bounds!")
	}
	var value int
	if err := json.Unmarshal(payload, &value); err != nil {
		return nil, err
	}
	if value < -1 || value >= len(s.resources) {
		return nil, fmt.Errorf("value %d out of bounds -1..%d",
			value, len(s.resources)-1)
	}
	s.curIndex = value
	s.shared.mutex.Lock()
	defer s.shared.mutex.Unlock()
	if s.shared.activeRequest {
		return nil, errors.New("too many requests")
	}
	s.shared.activeRequest = true
	if value == -1 {
		s.shared.file = nil
	} else {
		s.shared.file = s.resources[s.curIndex]
	}
	return value, nil
}
