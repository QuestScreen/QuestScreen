package controls

import (
	"syscall/js"

	"github.com/QuestScreen/QuestScreen/web"
	"github.com/flyx/askew/runtime"
)

type popupController interface {
	confirm()
	cancel()
	needsDoShow() bool
	doShow()
}

func (pb *PopupBase) show(title string, content runtime.Component, confirmCaption, cancelCaption string) {
	pb.Title.Set(title)
	pb.Content.Set(content)
	pb.ConfirmCaption.Set(confirmCaption)
	if cancelCaption == "" {
		pb.cancelVisible.Set("hidden")
	} else {
		pb.cancelVisible.Set("visible")
		pb.CancelCaption.Set(cancelCaption)
	}
	if pb.controller != nil && pb.controller.needsDoShow() {
		pb.Visibility.Set("hidden")
		pb.Display.Set("flex")
		pb.controller.doShow()
		// this is required to avoid flickering. I have no idea why.
		// it doesn't work if the timeout simply removes style.visibility.
		pb.Display.Set("none")
		pb.Visibility.Set("")
		var f js.Func
		f = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			pb.Display.Set("flex")
			f.Release()
			return nil
		})
		js.Global().Call("setTimeout", f, 10)
	} else {
		pb.Display.Set("flex")
	}
}

func (pb *PopupBase) confirm() {
	if pb.controller != nil {
		pb.controller.confirm()
	}
	pb.cleanup()
}

func (pb *PopupBase) cancel() {
	if pb.controller != nil {
		pb.controller.cancel()
	}
	pb.cleanup()
}

func (pb *PopupBase) cleanup() {
	pb.Display.Set("")
	pb.Content.Set(nil)
	pb.controller = nil
}

// ErrorMsg shows the popup containing the given text titled as 'Error'.
// Does not block.
func (pb *PopupBase) ErrorMsg(text string) {
	pb.show("Error", newPopupText(text), "OK", "")
}

type confirmController struct {
	val chan bool
}

func (cc *confirmController) confirm() {
	cc.val <- true
}

func (cc *confirmController) cancel() {
	cc.val <- false
}

func (cc *confirmController) needsDoShow() bool {
	return false
}

func (cc *confirmController) doShow() {}

// Confirm shows the popup and returns true if the user clicks OK, false if
// Cancel. Blocking, must be called from a goroutine.
func (pb *PopupBase) Confirm(text string) bool {
	c := &confirmController{make(chan bool, 1)}
	pb.controller = c
	pb.show("Confirm", newPopupText(text), "OK", "Cancel")
	ret := <-c.val
	pb.controller = nil
	return ret
}

type textInputController struct {
	val   chan *string
	input *popupInput
}

func (tic *textInputController) confirm() {
	str := tic.input.Value.Get()
	tic.val <- &str
}

func (tic *textInputController) cancel() {
	tic.val <- nil
}

func (tic *textInputController) needsDoShow() bool {
	return false
}

func (tic *textInputController) doShow() {}

// TextInput shows the popup and returns the entered string if the user clicks
// OK, nil if Cancel. Blocking, must be called from a goroutine.
func (pb *PopupBase) TextInput(title, label string) *string {
	tic := &textInputController{make(chan *string, 1), newPopupInput(label)}
	pb.controller = tic
	pb.show(title, tic.input, "OK", "Cancel")
	ret := <-tic.val
	pb.controller = nil
	return ret
}

type templateSelectController struct {
	*popupFromTemplate
	val chan bool
}

func (tsc *templateSelectController) confirm() {
	tsc.val <- true
}

func (tsc *templateSelectController) cancel() {
	tsc.val <- false
}

func (tsc *templateSelectController) needsDoShow() bool {
	return true
}

func (tsc *templateSelectController) doShow() {
	tsc.Expanded.Set(true)
	for index := 0; index < tsc.Templates.Len(); index++ {
		item := tsc.Templates.Item(index)
		// calculate and explicitly set the height of the item based on the height
		// of the container which can vary due to its variable content.
		// this is required to make our expand/collapse animation work.
		//
		// the 1em is the summed .5em vertical padding around the container.
		item.Height.Set("calc(" + item.OffsetHeight.Get() + "px + 1em)")
	}
	// select first item
	tsc.Templates.Item(0).click()
	tsc.Expanded.Set(false)
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
func (pb *PopupBase) TemplateSelect(kind TemplateKind) (pluginIndex int, tmplIndex int, name string) {
	c := &templateSelectController{newPopupFromTemplate(), make(chan bool, 1)}
	for pIndex, p := range web.StaticData.Plugins {
		switch kind {
		case SystemTemplate:
			for tIndex, t := range p.SystemTemplates {
				item := newPopupSelectableTemplate(p.Name, t.Name, t.Description)
				item.pluginIndex = pIndex
				item.templateIndex = tIndex
				c.Templates.Append(item)
			}
		case GroupTemplate:
			for tIndex, t := range p.GroupTemplates {
				item := newPopupSelectableTemplate(p.Name, t.Name, t.Description)
				item.pluginIndex = pIndex
				item.templateIndex = tIndex
				c.Templates.Append(item)
			}
		case SceneTemplate:
			for tIndex, t := range p.SceneTemplates {
				item := newPopupSelectableTemplate(p.Name, t.Name, t.Description)
				item.pluginIndex = pIndex
				item.templateIndex = tIndex
				c.Templates.Append(item)
			}
		}
	}
	pb.controller = c
	switch kind {
	case SystemTemplate:
		pb.show("Create System", c.popupFromTemplate, "OK", "Cancel")
	case GroupTemplate:
		pb.show("Create Group", c.popupFromTemplate, "OK", "Cancel")
	case SceneTemplate:
		pb.show("Create Scene", c.popupFromTemplate, "OK", "Cancel")
	}
	if <-c.val {
		pluginIndex = c.selectedPlugin
		tmplIndex = c.selectedTemplate
		name = c.Name.Get()
	} else {
		pluginIndex = -1
		tmplIndex = -1
	}
	pb.controller = nil
	return
}
