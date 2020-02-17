package title

import (
	"github.com/QuestScreen/QuestScreen/api"
	"gopkg.in/yaml.v3"
)

type state struct {
	caption string
}

type endpoint struct {
	*state
}

func newState(input *yaml.Node, ctx api.ServerContext) (api.ModuleState, error) {
	s := &state{}

	if input == nil {
		s.caption = ""
	} else {
		if err := input.Decode(&s.caption); err != nil {
			return nil, err
		}
	}

	return s, nil
}

func (s *state) CreateRendererData() interface{} {
	ret := &changeRequest{caption: s.caption}
	return ret
}

// WebView returns the current caption of the title as string.
func (s *state) WebView(ctx api.ServerContext) interface{} {
	return s.caption
}

// PersistingView returns the current caption of the title as string.
func (s *state) PersistingView(ctx api.ServerContext) interface{} {
	return s.caption
}

func (s *state) PureEndpoint(index int) api.ModulePureEndpoint {
	if index != 0 {
		panic("Endpoint index out of range")
	}
	return endpoint{s}
}

func (e endpoint) Put(payload []byte) (interface{}, interface{},
	api.SendableError) {
	var value string
	if err := api.ReceiveData(payload, &value); err != nil {
		return nil, nil, err
	}
	e.caption = value
	return value, &changeRequest{caption: value}, nil
}
