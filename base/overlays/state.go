package overlays

import (
	"log"

	"github.com/QuestScreen/QuestScreen/api"
	"gopkg.in/yaml.v3"
)

type state struct {
	visible   []bool
	resources []api.Resource
}

type endpoint struct {
	*state
}

type persistentState []string

func newState(input *yaml.Node, ctx api.ServerContext) (api.ModuleState, error) {
	s := &state{resources: ctx.GetResources(0)}
	s.visible = make([]bool, len(s.resources))
	if input != nil {
		var tmp persistentState
		if err := input.Decode(&tmp); err != nil {
			return nil, err
		}
		for i := range tmp {
			found := false
			for j := range s.resources {
				if s.resources[j].Name() == tmp[i] {
					found = true
					s.visible[j] = true
					break
				}
			}
			if !found {
				log.Println("Did not find resource \"" + tmp[i] + "\"")
			}
		}
	}

	return s, nil
}

func (s *state) CreateModuleData() interface{} {
	visible := make([]bool, len(s.visible))
	copy(visible, s.visible)
	resources := make([]api.Resource, len(s.resources))
	copy(resources, s.resources)
	return &fullRequest{visible: visible, resources: resources}
}

type webStateItem struct {
	Name     string `json:"name"`
	Selected bool   `json:"selected"`
}

type webState []webStateItem

// WebView returns a list of resources descriptors (name & visible)
func (s *state) WebView(ctx api.ServerContext) interface{} {
	ret := make(webState, len(s.resources))
	for i := range s.resources {
		ret[i].Name = s.resources[i].Name()
		ret[i].Selected = s.visible[i]
	}
	return ret
}

// PersistingView returns a list of selected resource names
func (s *state) PersistingView(ctx api.ServerContext) interface{} {
	ret := make([]string, 0, len(s.resources))
	for i := range s.visible {
		if s.visible[i] {
			ret = append(ret, s.resources[i].Name())
		}
	}
	return ret
}

func (s *state) PureEndpoint(index int) api.ModulePureEndpoint {
	if index != 0 {
		panic("Endpoint index out of range")
	}
	return endpoint{s}
}

func (e endpoint) Put(payload []byte) (interface{},
	interface{}, api.SendableError) {
	value := struct {
		ResourceIndex api.ValidatedInt `json:"resourceIndex"`
		Visible       bool             `json:"visible"`
	}{ResourceIndex: api.ValidatedInt{Min: 0, Max: len(e.resources) - 1}}
	if err := api.ReceiveData(payload,
		&api.ValidatedStruct{Value: &value}); err != nil {
		return nil, nil, err
	}
	e.visible[value.ResourceIndex.Value] = value.Visible
	return value.Visible, &itemRequest{index: value.ResourceIndex.Value,
		visible: value.Visible}, nil
}
