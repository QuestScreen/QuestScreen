package site

import (
	"fmt"
	"syscall/js"
)

var headerHeight int

func (tb *TitleBar) headerToggleClicked(target js.Value) {
	if headerHeight > 0 {
		tb.ToggleOrientation.Set(1)
		var f js.Func
		f = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			Header.height.Set("")
			Header.paddingBottom.Set("")
			Header.overflow.Set("")
			f.Release()
			return nil
		})
		Header.Self.Get().Call("addEventListener", "transitionend", f,
			js.Global().Get("JSON").Call("parse", `{"once":true}`))
		Header.height.Set(fmt.Sprintf("%vpx", headerHeight))
		headerHeight = 0
	} else {
		tb.ToggleOrientation.Set(2)
		headerHeight = Header.offsetHeight.Get()
		// no transition since height was 'auto' before
		Header.height.Set(fmt.Sprintf("%vpx", headerHeight))
		Header.paddingBottom.Set("0")
		Header.overflow.Set("hidden")
		Header.offsetWidth.Get() // forces repaint
		Header.height.Set("0")
	}
}

func (o *PageMenuEntry) clicked() {

}

func (o *GroupMenuItem) clicked() {

}
