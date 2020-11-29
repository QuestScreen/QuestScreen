package title

import (
	"github.com/QuestScreen/api/comms"
	"github.com/QuestScreen/api/modules"
	"github.com/QuestScreen/api/server"
	"gopkg.in/yaml.v3"
)

type state struct {
	caption string
}

type endpoint struct {
	*state
}

func newState(input *yaml.Node, ctx server.Context,
	ms server.MessageSender) (modules.State, error) {
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

func (s *state) CreateRendererData(ctx server.Context) interface{} {
	ret := &changeRequest{caption: s.caption}
	return ret
}

// WebView returns the current caption of the title as string.
func (s *state) WebView(ctx server.Context) interface{} {
	return s.caption
}

// PersistingView returns the current caption of the title as string.
func (s *state) PersistingView(ctx server.Context) interface{} {
	return s.caption
}

func (s *state) PureEndpoint(index int) modules.PureEndpoint {
	if index != 0 {
		panic("Endpoint index out of range")
	}
	return endpoint{s}
}

func (e endpoint) Post(payload []byte) (interface{}, interface{},
	server.Error) {
	var value string
	if err := comms.ReceiveData(payload, &value); err != nil {
		return nil, nil, &server.BadRequest{Inner: err, Message: "received invalid data"}
	}
	e.caption = value
	return value, &changeRequest{caption: value}, nil
}
