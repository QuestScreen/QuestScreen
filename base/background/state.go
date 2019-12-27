package background

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/flyx/pnpscreen/api"
	"gopkg.in/yaml.v3"
)

type state struct {
	moduleIndex api.ModuleIndex
	curIndex    int
	resources   []api.Resource
}

// LoadFrom loads the stored selection, defaults to no item being selected.
func newState(input *yaml.Node, env api.Environment, index api.ModuleIndex) (api.ModuleState, error) {
	s := new(state)
	s.resources = env.GetResources(index, 0)
	s.curIndex = -1
	if input != nil {
		if input.Kind != yaml.ScalarNode {
			return nil, errors.New("unexpected YAML for Background state: not a string")
		}
		for i := range s.resources {
			if s.resources[i].Name() == input.Value {
				s.curIndex = i
				break
			}
		}
		if s.curIndex == -1 {
			log.Println("Didn't find resource \"" + input.Value + "\"")
		}
	}
	return s, nil
}

func (s *state) CreateModuleData() interface{} {
	if s.curIndex == -1 {
		return &request{file: nil}
	}
	return &request{file: s.resources[s.curIndex]}
}

type webState struct {
	CurIndex int      `json:"curIndex"`
	Items    []string `json:"items"`
}

// SerializableView returns
// - the name of the currently selected resource (nil if none) for Persisted
// - the list of all available resources plus the current index for Web
func (s *state) SerializableView(
	env api.Environment, layout api.DataLayout) interface{} {
	if layout == api.Persisted {
		if s.curIndex == -1 {
			return nil
		}
		return s.resources[s.curIndex].Name()
	}
	return webState{CurIndex: s.curIndex, Items: api.ResourceNames(s.resources)}
}

func (s *state) HandleAction(index int, payload []byte) (interface{}, interface{}, error) {
	if index != 0 {
		panic("Index out of bounds!")
	}
	var value int
	if err := json.Unmarshal(payload, &value); err != nil {
		return nil, nil, err
	}
	if value < -1 || value >= len(s.resources) {
		return nil, nil, fmt.Errorf("value %d out of bounds -1..%d",
			value, len(s.resources)-1)
	}
	s.curIndex = value
	req := &request{}
	if value != -1 {
		req.file = s.resources[s.curIndex]
	}
	return value, req, nil
}
