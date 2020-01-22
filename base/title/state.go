package title

import (
	"github.com/flyx/pnpscreen/api"
	"gopkg.in/yaml.v3"
)

type state struct {
	caption   string
	resources []api.Resource
}

type endpoint struct {
	*state
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

// WebView returns the current caption of the title as string.
func (s *state) WebView(env api.Environment) interface{} {
	return s.caption
}

// PersistingView returns the current caption of the title as string.
func (s *state) PersistingView(env api.Environment) interface{} {
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
