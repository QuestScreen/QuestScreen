package overlays

import (
	"github.com/QuestScreen/QuestScreen/plugins/base/shared"
	"github.com/QuestScreen/api/modules"
	"github.com/QuestScreen/api/resources"
	"github.com/QuestScreen/api/server"
	"gopkg.in/yaml.v3"
)

type state struct {
	visible []bool
	items   []resources.Resource
}

type endpoint struct {
	*state
}

type persistentState []string

func newState(input *yaml.Node, ctx server.Context,
	ms server.MessageSender) (modules.State, error) {
	s := &state{items: ctx.GetResources(0)}
	s.visible = make([]bool, len(s.items))
	if input != nil {
		var tmp persistentState
		if err := input.Decode(&tmp); err != nil {
			return nil, err
		}
		for i := range tmp {
			found := false
			for j := range s.items {
				if s.items[j].Name() == tmp[i] {
					found = true
					s.visible[j] = true
					break
				}
			}
			if !found {
				ms.Warning("Did not find resource \"" + tmp[i] + "\"")
			}
		}
	}

	return s, nil
}

func (s *state) CreateRendererData(ctx server.Context) interface{} {
	resources := make([]showRequest, 0, len(s.items))
	for i := range s.items {
		if s.visible[i] {
			resources = append(resources, showRequest{s.items[i], i})
		}
	}
	return &fullRequest{resources: resources}
}

// WebView returns a list of resources descriptors (name & visible)
func (s *state) WebView(ctx server.Context) interface{} {
	ret := make(shared.OverlayState, len(s.items))
	for i := range s.items {
		ret[i].Name = s.items[i].Name()
		ret[i].Selected = s.visible[i]
	}
	return ret
}

// PersistingView returns a list of selected resource names
func (s *state) PersistingView(ctx server.Context) interface{} {
	ret := make([]string, 0, len(s.items))
	for i := range s.visible {
		if s.visible[i] {
			ret = append(ret, s.items[i].Name())
		}
	}
	return ret
}

func (s *state) PureEndpoint(index int) modules.PureEndpoint {
	if index != 0 {
		panic("Endpoint index out of range")
	}
	return endpoint{s}
}

func (e endpoint) Post(payload []byte) (interface{},
	interface{}, server.Error) {
	value := struct {
		ResourceIndex server.ValidatedInt `json:"resourceIndex"`
		Visible       bool                `json:"visible"`
	}{ResourceIndex: server.ValidatedInt{Min: 0, Max: len(e.items) - 1}}
	if err := server.ReceiveData(payload,
		&server.ValidatedStruct{Value: &value}); err != nil {
		return nil, nil, err
	}
	e.visible[value.ResourceIndex.Value] = value.Visible
	if value.Visible {
		return value.Visible, &showRequest{
			resource:      e.items[value.ResourceIndex.Value],
			resourceIndex: value.ResourceIndex.Value}, nil
	}
	return value.Visible, &hideRequest{resourceIndex: value.ResourceIndex.Value}, nil
}
