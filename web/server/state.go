package server

import (
	"strings"

	api "github.com/QuestScreen/api/web/server"
)

// State implements web.ServerState.
type State struct {
	base string
}

// Fetch implements web.ServerState (see there for description)
func (s *State) Fetch(method api.RequestMethod, subpath string, payload interface{}, target interface{}) {
	var urlBuilder strings.Builder
	urlBuilder.WriteString(s.base)
	if subpath != "" {
		if subpath[0] == '/' {
			panic("Fetch @ " + s.base + ": subpath '" + subpath + "' begins with a slash")
		}
		urlBuilder.WriteByte('/')
		urlBuilder.WriteString(subpath)
	}
	if err := Fetch(method, urlBuilder.String(), payload, target); err != nil {
		panic(err)
	}
}
