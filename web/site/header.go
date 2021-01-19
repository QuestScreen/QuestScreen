package site

import (
	"fmt"
	"syscall/js"
)

var headerHeight int

// this function implements hiding and showing the header.
// To properly animate it, it retrieves its height, sets that as defined height
// and then uses CSS transitioning to shrink it to zero, and when showing, grow
// it back to its original size.
func (tb *topBar) headerToggleClicked(target js.Value) {
	if headerHeight > 0 {
		tb.ToggleOrientation.Set(1)
		var f js.Func
		f = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			tb.height.Set("")
			tb.paddingBottom.Set("")
			tb.overflow.Set("")
			f.Release()
			return nil
		})
		tb.Self.Get().Call("addEventListener", "transitionend", f,
			js.Global().Get("JSON").Call("parse", `{"once":true}`))
		tb.height.Set(fmt.Sprintf("%vpx", headerHeight))
		headerHeight = 0
	} else {
		tb.ToggleOrientation.Set(2)
		headerHeight = tb.offsetHeight.Get()
		// no transition since height was 'auto' before
		tb.height.Set(fmt.Sprintf("%vpx", headerHeight))
		tb.paddingBottom.Set("0")
		tb.overflow.Set("hidden")
		tb.offsetWidth.Get() // forces repaint
		tb.height.Set("0")
	}
}
