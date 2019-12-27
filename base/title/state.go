package title

import (
	"encoding/json"

	"github.com/flyx/pnpscreen/api"
	"gopkg.in/yaml.v3"
)

type state struct {
	caption   string
	resources []api.Resource
}

func newState(input *yaml.Node, env api.Environment,
	index api.ModuleIndex) (api.ModuleState, error) {
	s := &state{resources: env.GetResources(index, 0)}

	if input == nil {
		s.caption = ""
	} else {
		if err := input.Decode(&s.caption); err != nil {
			return nil, err
		}
	}

	return s, nil
}

func (s *state) CreateModuleData() interface{} {
	ret := &fullRequest{caption: s.caption}
	if len(s.resources) > 0 {
		ret.mask = s.resources[0]
	}
	return ret
}

// SerializableView returns the current caption of the title as string.
func (s *state) SerializableView(
	env api.Environment, layout api.DataLayout) interface{} {
	return s.caption
}

func (s *state) HandleAction(index int,
	payload []byte) (interface{}, interface{}, error) {
	if index != 0 {
		panic("Index out of range")
	}
	var value string
	if err := json.Unmarshal(payload, &value); err != nil {
		return nil, nil, err
	}
	s.caption = value
	return value, &changeRequest{caption: value}, nil
}
