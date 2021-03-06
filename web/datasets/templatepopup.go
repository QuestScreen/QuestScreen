package datasets

import (
	"fmt"

	"github.com/QuestScreen/QuestScreen/shared"
	"github.com/QuestScreen/QuestScreen/web"
	"github.com/QuestScreen/api/web/controls"
)

func (o *popupFromTemplate) Confirm() {
	go o.cb(o.selectedPlugin, o.selectedTemplate, o.Name.Get())
}

func (o *popupFromTemplate) Cancel() {
}

func (o *popupFromTemplate) NeedsDoShow() bool {
	return true
}

func (o *popupFromTemplate) DoShow() {
	o.Expanded.Set(true)
	for index := 0; index < o.Templates.Len(); index++ {
		item := o.Templates.Item(index)
		// calculate and explicitly set the height of the item based on the height
		// of the container which can vary due to its variable content.
		// this is required to make our expand/collapse animation work.
		//
		// the 1em is the summed .5em vertical padding around the container.
		item.Height.Set(fmt.Sprintf("calc(%vpx + 1em)", item.OffsetHeight.Get()))
	}
	// select first item
	o.Templates.Item(0).click()
	o.Expanded.Set(false)
}

func (o *popupSelectableTemplate) click() {
	o.Controller.choose(o.pluginIndex, o.templateIndex, o.Active.Get())
	o.Active.Set(true)
}

func (o *popupFromTemplate) choose(pluginIndex int, templateIndex int, active bool) {
	if active {
		o.Expanded.Set(!o.Expanded.Get())
	} else {
		for index := 0; index < o.Templates.Len(); index++ {
			o.Templates.Item(index).Active.Set(false)
		}
		o.Expanded.Set(false)
		o.selectedPlugin = pluginIndex
		o.selectedTemplate = templateIndex
	}
}

// TemplateKind defines what entity a template can create.
type TemplateKind int

const (
	// SystemTemplate creates a system.
	SystemTemplate TemplateKind = iota
	// GroupTemplate creates a group
	GroupTemplate
	// SceneTemplate creates a scene (but doesn't make a scene about it)
	SceneTemplate
)

// TemplateSelect shows the popup and lets the user enter a name and select a
// template out of the available templates for the given kind.
func TemplateSelect(pb *controls.PopupBase, kind TemplateKind, cb func(pluginIndex int, tmplIndex int, name string)) {
	pft := newPopupFromTemplate(cb)

	for pIndex, p := range web.StaticData.Plugins {
		var templates []shared.TemplateDescr
		switch kind {
		case SystemTemplate:
			templates = p.SystemTemplates
		case GroupTemplate:
			templates = p.GroupTemplates
		case SceneTemplate:
			templates = p.SceneTemplates
		}

		for tIndex, t := range templates {
			item := newPopupSelectableTemplate(p.Name, t.Name, t.Description)
			item.pluginIndex = pIndex
			item.templateIndex = tIndex
			pft.Templates.Append(item)
		}
	}

	switch kind {
	case SystemTemplate:
		pb.Show("Create System", pft, "OK", "Cancel")
	case GroupTemplate:
		pb.Show("Create Group", pft, "OK", "Cancel")
	case SceneTemplate:
		pb.Show("Create Scene", pft, "OK", "Cancel")
	}
}
