package main

/*
#cgo LDFLAGS: -lGLESv2
*/
import "C"
import (
	"github.com/flyx/egl"
	"github.com/flyx/rpscreen/internal/app/rpscreen"
	"runtime"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	ctrl := rpscreen.NewControlCh()
	eglState := rpscreen.InitEGL(ctrl, 800, 600)

	state, err := rpscreen.NewRenderState(eglState)
	if err != nil {
		panic(err)
	}

	//go userAppCode(ctrl)
Outer:
	for {
		state.Render()
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
