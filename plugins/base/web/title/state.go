package title

import (
	"encoding/json"

	"github.com/QuestScreen/api/web/groups"
	"github.com/QuestScreen/api/web/modules"
	"github.com/QuestScreen/api/web/server"
	"github.com/flyx/askew/runtime"
)

// State implements web.ModuleState
type State struct {
	server.State
	caption string
}

// NewState implements modules.Constructor.
func NewState(data json.RawMessage, srv server.State, group groups.Group) (modules.State, error) {
	ret := &State{State: srv}
	return ret, json.Unmarshal(data, &ret.caption)
}

// UI generates a new widget for the title state.
func (s *State) UI(srv server.State) runtime.Component {
	ret := NewWidget(s.caption)
	ret.Controller = s
	return ret
}

func (s *State) update(caption string) string {
	s.Fetch(server.Post, "", caption, &s.caption)
	return s.caption
}

func (w *Widget) submit(caption string) {
	if w.Controller != nil {
		w.Caption.Set(w.Controller.update(caption))
	}
}
