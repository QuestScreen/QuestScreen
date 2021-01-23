package web

import "syscall/js"

var smartphoneMode = js.Global().Get("window").Call(
	"matchMedia", "screen and (max-width: 35.5em)")

// InSmartphoneMode returns whether the css selector used for enabling
// smartphone UI currently matches.
func InSmartphoneMode() bool {
	return smartphoneMode.Get("matches").Bool()
}
