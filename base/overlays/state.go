package overlays

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/flyx/pnpscreen/api"
	"gopkg.in/yaml.v3"
)

type state struct {
	visible   []bool
	resources []api.Resource
}

type persistentState []string

func newState(input *yaml.Node, env api.Environment,
	index api.ModuleIndex) (api.ModuleState, error) {
	s := &state{resources: env.GetResources(index, 0)}
	s.visible = make([]bool, len(s.resources))
	if input != nil {
		var tmp persistentState
		if err := input.Decode(&tmp); err != nil {
			return nil, err
		}
		for i := range tmp {
			found := false
			for j := range s.resources {
				if s.resources[j].Name() == tmp[i] {
					found = true
					s.visible[j] = true
					break
				}
			}
			if !found {
				log.Println("Did not find resource \"" + tmp[i] + "\"")
			}
		}
	}

	return s, nil
}

func (s *state) CreateModuleData() interface{} {
	visible := make([]bool, len(s.visible))
	copy(visible, s.visible)
	resources := make([]api.Resource, len(s.resources))
	copy(resources, s.resources)
	return &fullRequest{visible: visible, resources: resources}
}

type webStateItem struct {
	Name     string `json:"name"`
	Selected bool   `json:"selected"`
}

type webState []webStateItem

// SerializableView returns
// - a list of selected resource names for Persisted
// - a list of resources descriptors (name & visible) for Web
func (s *state) SerializableView(
	env api.Environment, layout api.DataLayout) interface{} {
	if layout == api.Persisted {
		ret := make([]string, 0, len(s.resources))
		for i := range s.visible {
			if s.visible[i] {
				ret = append(ret, s.resources[i].Name())
			}
		}
		return ret
	}
	ret := make(webState, len(s.resources))
	for i := range s.resources {
		ret[i].Name = s.resources[i].Name()
		ret[i].Selected = s.visible[i]
	}
	return ret
}

func (*state) Actions() []string {
	return []string{"switch"}
}

func (s *state) HandleAction(index int, payload []byte) (interface{}, interface{}, error) {
	if index != 0 {
		panic("Index out of range")
	}
	var value int
	if err := json.Unmarshal(payload, &value); err != nil {
		return nil, nil, err
	}
	if value < 0 || value >= len(s.resources) {
		return nil, nil, fmt.Errorf("Index %d not in range 0..%d",
			value, len(s.resources)-1)
	}
	s.visible[value] = !s.visible[value]
	return s.visible[value], &itemRequest{index: value, visible: s.visible[value]}, nil
}
