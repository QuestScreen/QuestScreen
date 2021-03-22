package session

import "github.com/QuestScreen/api/web/modules"

type namedState struct {
	name  string
	state modules.State
}
