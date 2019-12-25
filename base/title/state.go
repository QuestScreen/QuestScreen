package title

import (
	"encoding/json"
	"errors"

	"github.com/flyx/pnpscreen/api"
)

type state struct {
	shared    *sharedData
	caption   string
	resources []api.Resource
}

func newState(yamlSubtree interface{}, env api.Environment,
	shared *sharedData) (*state, error) {
	s := new(state)
	s.resources = env.GetResources(shared.moduleIndex, 0)
	s.shared = shared

	if yamlSubtree == nil {
		s.caption = ""
	} else {
		var ok bool
		s.caption, ok = yamlSubtree.(string)
		if !ok {
			return nil, errors.New("title caption is not a string")
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

func (s *state) ToYAML(env api.Environment) interface{} {
	return s.caption
}

func (s *state) ToJSON() interface{} {
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
