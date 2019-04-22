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

	screen, err := newScreen(eglState, ctrl)
	if err != nil {
		panic(err)
	}

	server := startServer(screen)

Outer:
	for {
		screen.Render()
		egl.SwapBuffers(eglState.Display, eglState.Surface)
		select {
		case curUpdate := <-ctrl.ModuleUpdate:
			break
		case <-ctrl.Exit:
			_ = server.Close()
			egl.DestroySurface(eglState.Display, eglState.Surface)
			egl.DestroyContext(eglState.Display, eglState.Context)
			egl.Terminate(eglState.Display)
			break Outer
		}
	}
}
