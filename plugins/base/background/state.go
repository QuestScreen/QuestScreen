package background

import (
	"errors"

	"github.com/QuestScreen/QuestScreen/plugins/base/shared"
	"github.com/QuestScreen/api/modules"
	"github.com/QuestScreen/api/resources"
	"github.com/QuestScreen/api/server"
	"gopkg.in/yaml.v3"
)

type state struct {
	curIndex  int
	resources []resources.Resource
}

type endpoint struct {
	*state
}

// LoadFrom loads the stored selection, defaults to no item being selected.
func newState(input *yaml.Node, ctx server.Context,
	ms server.MessageSender) (modules.State, error) {
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

func (s *state) CreateRendererData(ctx server.Context) interface{} {
	if s.curIndex == -1 {
		return &request{file: nil}
	}
	return &request{file: s.resources[s.curIndex]}
}

// WebView returns the list of all available resources plus the current index
func (s *state) WebView(env server.Context) interface{} {
	return shared.BackgroundState{CurIndex: s.curIndex, Items: resources.Names(s.resources)}
}

// PersistingView returns the name of the currently selected resource
//(nil if none)
func (s *state) PersistingView(env server.Context) interface{} {
	if s.curIndex == -1 {
		return nil
	}
	return s.resources[s.curIndex].Name()
}

func (s *state) PureEndpoint(index int) modules.PureEndpoint {
	if index != 0 {
		panic("Endpoint index out of bounds")
	}
	return endpoint{s}
}

func (e endpoint) Post(payload []byte) (interface{}, interface{},
	server.Error) {
	value := server.ValidatedInt{Min: -1, Max: len(e.resources) - 1}
	if err := server.ReceiveData(payload, &value); err != nil {
		return nil, nil, err
	}
	e.curIndex = value.Value
	req := &request{}
	if e.curIndex != -1 {
		req.file = e.resources[e.curIndex]
	}
	return value, req, nil
}
