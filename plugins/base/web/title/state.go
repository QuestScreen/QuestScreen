package title

import (
	"encoding/json"

	"github.com/QuestScreen/api/web"
	"github.com/QuestScreen/api/web/modules"
)

// NewState implements modules.Constructor.
func NewState(data json.RawMessage, srv web.Server) (modules.State, error) {
	ret := &State{srv: srv}
	if err := json.Unmarshal(data, &ret.caption); err != nil {
		return nil, err
	}
	ret.askewInit(ret.caption)
	return ret, nil
}

func (s *State) submit(caption string) {
	s.srv.Fetch(web.Post, "", caption, &s.caption)
	s.Caption.Set(s.caption)
}
