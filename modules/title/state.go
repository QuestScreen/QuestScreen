package title

import (
	"encoding/json"
	"errors"
	"github.com/flyx/pnpscreen/api"
)

type state struct {
	owner   *Title
	caption string
	resources []api.Resource
}

func (s *state) LoadFrom(yamlSubtree interface{}, env api.Environment) error {
	s.resources = env.GetResources(s.owner.moduleIndex, 0)

	if yamlSubtree == nil {
		s.caption = ""
	} else {
		var ok bool
		s.caption, ok = yamlSubtree.(string)
		if !ok {
			return errors.New("title caption is not a string")
		}
	}

	s.owner.requests.mutex.Lock()
	s.owner.requests.kind = stateRequest
	s.owner.requests.caption = s.caption
	s.owner.requests.mutex.Unlock()

	return nil
}

func (s *state) ToYAML(env api.Environment) interface{} {
	return s.caption
}

func (s *state) ToJSON() interface{} {
	return s.caption
}

func (*state) Actions() []string {
	return []string{"set"}
}

func (s *state) HandleAction(index int, payload []byte) ([]byte, error) {
	if index != 0 {
		panic("Index out of range")
	}
	var value string
	if err := json.Unmarshal(payload, &value); err != nil {
		return nil, err
	}
	s.caption = value
	s.owner.requests.mutex.Lock()
	s.owner.requests.kind = changeRequest
	s.owner.requests.caption = value
	s.owner.requests.mutex.Unlock()
	return nil, nil
}
