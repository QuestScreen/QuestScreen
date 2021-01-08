package controls

import (
	"syscall/js"

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
	pb.CancelCaption.Set(cancelCaption)
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
