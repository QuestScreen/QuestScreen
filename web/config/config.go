package config

import "github.com/QuestScreen/api/web/config"

func newItem(ui config.Widget, wasEnabled bool) *item {
	ret := new(item)
	ret.init(ui, wasEnabled)
	return ret
}

func (o *item) init(ui config.Widget, wasEnabled bool) {
	o.askewInit(ui, wasEnabled)
	o.editIndicator.Set("hidden")
	o.content.SetEnabled(wasEnabled)
}

func (o *item) Edited() {
	o.valuesEdited = true
	o.editIndicator.Set("visible")
}

func (o *item) editedEnabled() {
	newVal := o.enabled.Get()
	o.content.SetEnabled(newVal)
	if (newVal == o.wasEnabled) && !(newVal && o.valuesEdited) {
		o.editIndicator.Set("hidden")
	} else {
		o.editIndicator.Set("visible")
	}
}
