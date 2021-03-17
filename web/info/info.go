package info

import "github.com/QuestScreen/QuestScreen/web/session"

func (o *ChooseableGroup) click() {
	go session.StartSession(o.index)
}
