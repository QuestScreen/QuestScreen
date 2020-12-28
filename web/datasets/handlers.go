package datasets

import (
	"github.com/QuestScreen/QuestScreen/shared"
	"github.com/QuestScreen/QuestScreen/web"
)

func (o *EditableText) setEdited() {
	o.Edited.Set(true)
}

func (o *ListItem) clicked(index int) {
}

func (o *Base) init(data *shared.Data) {
	for index, group := range data.Groups {
		o.Groups.Append(NewListItem(group.Name, true, index))
	}
	for index, system := range data.Systems {
		o.Systems.Append(NewListItem(system.Name,
			index >= web.StaticData.NumPluginSystems, index))
	}
}

func (o *Base) onInclude() {
	web.Page.SetTitle("Dataset Base", "", web.BackButtonBack)
}

func (o *Base) addSystem() {

}

func (o *Base) addGroup() {

}
