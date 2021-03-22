package config

import "github.com/QuestScreen/api/web/config"

func newItem(ui config.Widget, name string, wasEnabled bool, p *Page) *item {
	ret := new(item)
	ret.init(ui, name, wasEnabled, p)
	return ret
}

func (o *item) init(ui config.Widget, name string, wasEnabled bool, p *Page) {
	o.askewInit(ui, name, wasEnabled, p)
	o.editIndicator.Set("hidden")
	o.content.SetEnabled(wasEnabled)
}

func (o *item) Edited() {
	o.valuesEdited = true
	o.editIndicator.Set("visible")
	o.p.updateEdited(true)
}

func (o *item) editedEnabled() {
	newVal := o.enabled.Get()
	o.content.SetEnabled(newVal)
	if (newVal == o.wasEnabled) && !(newVal && o.valuesEdited) {
		o.editIndicator.Set("hidden")
	} else {
		o.editIndicator.Set("visible")
	}
	o.p.updateEdited(false)
}
