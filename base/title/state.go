package title

import (
	"encoding/json"

	"github.com/flyx/pnpscreen/api"
	"gopkg.in/yaml.v3"
)

type state struct {
	shared    *sharedData
	caption   string
	resources []api.Resource
}

func newState(input *yaml.Node, env api.Environment,
	shared *sharedData) (*state, error) {
	s := new(state)
	s.resources = env.GetResources(shared.moduleIndex, 0)
	s.shared = shared

	if input == nil {
		s.caption = ""
	} else {
		if err := input.Decode(&s.caption); err != nil {
			return nil, err
		}
	}

	return s, nil
}

func (s *state) SendToModule() {
	s.shared.mutex.Lock()
	s.shared.kind = stateRequest
	s.shared.caption = s.caption
	if len(s.resources) > 0 {
		s.shared.mask = s.resources[0]
	} else {
		s.shared.mask = nil
	}
	s.shared.mutex.Unlock()
}

// SerializableView returns the current caption of the title as string.
func (s *state) SerializableView(
	env api.Environment, layout api.DataLayout) interface{} {
	return s.caption
}

func (s *state) HandleAction(index int, payload []byte) (interface{}, error) {
	if index != 0 {
		panic("Index out of range")
	}
	var value string
	if err := json.Unmarshal(payload, &value); err != nil {
		return nil, err
	}
	s.caption = value
	s.shared.mutex.Lock()
	s.shared.kind = changeRequest
	s.shared.caption = value
	s.shared.mutex.Unlock()
	return value, nil
}
