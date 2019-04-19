package main

import (
"github.com/BurntSushi/xgb/xproto"
"github.com/BurntSushi/xgbutil"
"github.com/BurntSushi/xgbutil/keybind"
"github.com/BurntSushi/xgbutil/mousebind"
"github.com/BurntSushi/xgbutil/xevent"
"github.com/BurntSushi/xgbutil/xwindow"
"github.com/flyx/egl"
"github.com/flyx/egl/platform"
"github.com/flyx/egl/platform/xorg"
"unsafe"
"fmt"
)

func newWindow(controlCh *controlCh, X *xgbutil.XUtil, width, height int) *xwindow.Window {
	fmt.Println("newWindow()")
	defer fmt.Println("/newWindow()")
	var (
		err error
		win *xwindow.Window
	)

	win, err = xwindow.Generate(X)
	if err != nil {
		panic(err)
	}

	win.Create(X.RootWin(), 0, 0, width, height,
		xproto.CwBackPixel|xproto.CwEventMask,
		0, xproto.EventMaskButtonRelease)

	// Xorg application exits when the window is closed.
	win.WMGracefulClose(
		func(w *xwindow.Window) {
			xevent.Detach(w.X, w.Id)
			mousebind.Detach(w.X, w.Id)
			w.Destroy()
			xevent.Quit(X)
			controlCh.exit <- struct{}{}
		})

	// In order to get ConfigureNotify events, we must listen to the window
	// using the 'StructureNotify' mask.
	err = win.Listen(xproto.EventMaskButtonPress |
			xproto.EventMaskButtonRelease |
			xproto.EventMaskKeyPress |
			xproto.EventMaskKeyRelease |
			xproto.EventMaskStructureNotify)
	if err != nil {
		panic(err)
	}
	win.Map()
	return win
}

func initEGL(controlCh *controlCh, width, height int) *platform.EGLState {
	fmt.Println("initEgl()")
	defer fmt.Println("/initEgl()")
	X, err := xgbutil.NewConn()
	if err != nil {
		panic(err)
	}
	mousebind.Initialize(X)
	keybind.Initialize(X)
	xWindow := newWindow(controlCh, X, width, height)
	go xevent.Main(X)
	return xorg.Initialize(
		egl.NativeWindowType(unsafe.Pointer(uintptr(xWindow.Id))),
		xorg.DefaultConfigAttributes,
		xorg.DefaultContextAttributes,
	)
}
