package main

import (
	"fmt"

	"github.com/gopherjs/gopherjs/js"
)

// App is the single-page application managing the web interface.
type App struct {
	headerHeight int
}

// BackButtonClicked implements the respective controller method for the title bar.
func (a *App) BackButtonClicked() {

}

// HeaderToggleClicked implemnts the respective controller method for the title bar.
func (a *App) HeaderToggleClicked(target *js.Object) {
	if a.headerHeight > 0 {
		TitleContent.ToggleOrientation.Set(1)
		Header.Self.Get().Call("addEventListener", func() {
			Header.height.Set("")
			Header.paddingBottom.Set("")
			Header.overflow.Set("")
		}, struct{ once bool }{true})
		Header.height.Set(fmt.Sprintf("%vpx", a.headerHeight))
		a.headerHeight = 0
	} else {
		TitleContent.ToggleOrientation.Set(2)
		a.headerHeight = Header.offsetHeight.Get()
		// no transition since height was 'auto' before
		Header.height.Set(fmt.Sprintf("%vpx", a.headerHeight))
		Header.paddingBottom.Set("0")
		Header.overflow.Set("hidden")
		Header.offsetWidth.Get() // forces repaint
		Header.height.Set("0")
	}
}
