package web

import (
	"encoding/json"

	"github.com/QuestScreen/api/web"
	"github.com/flyx/askew/runtime"
)

// State implements web.ModuleState
type State struct {
	web.ServerState
	caption string
}

// NewState creates a new title module state.
func NewState(data json.RawMessage, server web.ServerState, group web.GroupData) (web.ModuleState, error) {
	ret := &State{ServerState: server}
	return ret, json.Unmarshal(data, &ret.caption)
}

// UI generates a new widget for the title state.
func (s *State) UI() runtime.Component {
	ret := NewWidget(s.caption)
	ret.Controller = s
	return ret
}

func (s *State) update(caption string) string {
	s.Fetch(web.Post, "", caption, &s.caption)
	return s.caption
}

func (w *Widget) submit(caption string) {
	if w.Controller != nil {
		w.Caption.Set(w.Controller.update(caption))
	}
}
