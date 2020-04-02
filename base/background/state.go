package background

import (
	"errors"

	"github.com/QuestScreen/api"
	"gopkg.in/yaml.v3"
)

type state struct {
	curIndex  int
	resources []api.Resource
}

type endpoint struct {
	*state
}

// LoadFrom loads the stored selection, defaults to no item being selected.
func newState(input *yaml.Node, ctx api.ServerContext,
	ms api.MessageSender) (api.ModuleState, error) {
	s := new(state)
	s.resources = ctx.GetResources(0)
	s.curIndex = -1
	if input != nil {
		if input.Kind != yaml.ScalarNode {
			return nil, errors.New("unexpected YAML for Background state: not a string")
		}
		if input.Tag != "!!null" {
			for i := range s.resources {
				if s.resources[i].Name() == input.Value {
					s.curIndex = i
					break
				}
			}
			if s.curIndex == -1 {
				ms.Warning("Didn't find resource \"" + input.Value + "\"")
			}
		}
	}
	return s, nil
}

func (s *state) CreateRendererData() interface{} {
	if s.curIndex == -1 {
		return &request{file: nil}
	}
	return &request{file: s.resources[s.curIndex]}
}

type webState struct {
	CurIndex int      `json:"curIndex"`
	Items    []string `json:"items"`
}

// WebView returns the list of all available resources plus the current index
func (s *state) WebView(env api.ServerContext) interface{} {
	return webState{CurIndex: s.curIndex, Items: api.ResourceNames(s.resources)}
}

// PersistingView returns the name of the currently selected resource
//(nil if none)
func (s *state) PersistingView(env api.ServerContext) interface{} {
	if s.curIndex == -1 {
		return nil
	}
	return s.resources[s.curIndex].Name()
}

func (s *state) PureEndpoint(index int) api.ModulePureEndpoint {
	if index != 0 {
		panic("Endpoint index out of bounds")
	}
	return endpoint{s}
}

func (e endpoint) Post(payload []byte) (interface{}, interface{},
	api.SendableError) {
	value := api.ValidatedInt{Min: -1, Max: len(e.resources) - 1}
	if err := api.ReceiveData(payload, &value); err != nil {
		return nil, nil, err
	}
	e.curIndex = value.Value
	req := &request{}
	if e.curIndex != -1 {
		req.file = e.resources[e.curIndex]
	}
	return value, req, nil
}
