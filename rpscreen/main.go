package main

import (
	"github.com/flyx/egl"
	"runtime"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	ctrl := newControlCh()
	eglState := InitEGL(ctrl, 800, 600)

	screen, err := newScreen(eglState)
	if err != nil {
		panic(err)
	}

	startServer(screen)
Outer:
	for {
		screen.Render()
		egl.SwapBuffers(eglState.Display, eglState.Surface)
		select {
		case <-ctrl.Draw:
			break
		case <-ctrl.Exit:
			egl.DestroySurface(eglState.Display, eglState.Surface)
			egl.DestroyContext(eglState.Display, eglState.Context)
			egl.Terminate(eglState.Display)
			break Outer
		}
	}
}
