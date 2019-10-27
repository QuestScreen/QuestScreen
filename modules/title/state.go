package title

import (
	"encoding/json"
	"errors"

	"github.com/flyx/rpscreen/data"
)

type state struct {
	owner   *Title
	caption string
}

func (s *state) LoadFrom(yamlSubtree interface{}, store *data.Store) error {
	if yamlSubtree == nil {
		s.caption = ""
	} else {
		var ok bool
		s.caption, ok = yamlSubtree.(string)
		if !ok {
			return errors.New("Title caption is not a string")
		}
	}

	s.owner.requests.mutex.Lock()
	s.owner.requests.kind = stateRequest
	s.owner.requests.caption = s.caption
	s.owner.requests.mutex.Unlock()

	return nil
}

func (s *state) ToYAML(store *data.Store) interface{} {
	return s.caption
}

func (s *state) ToJSON() interface{} {
	return s.caption
}

func (*state) Actions() []string {
	return []string{"set"}
}

func (s *state) HandleAction(index int, payload []byte, store *data.Store) error {
	if index != 0 {
		panic("Index out of range")
	}
	var value string
	if err := json.Unmarshal(payload, &value); err != nil {
		return err
	}
	s.caption = value
	s.owner.requests.mutex.Lock()
	s.owner.requests.kind = changeRequest
	s.owner.requests.caption = value
	s.owner.requests.mutex.Unlock()
	return nil
}
