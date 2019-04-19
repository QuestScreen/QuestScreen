package main

/*
#cgo LDFLAGS: -lGLESv2
*/
import "C"
import (
	"github.com/flyx/egl"
	"github.com/flyx/egl/platform"
	"runtime"
)

type controlCh struct {
	exit        chan struct{}
	draw        chan struct{}
}

func newControlCh() *controlCh {
	return &controlCh{
		exit:        make(chan struct{}),
		draw:        make(chan struct{}),
	}
}

func init() {
	runtime.LockOSThread()
}

func main() {
	ctrl := newControlCh()
	eglState := initEGL(ctrl, 800, 600)

	var state renderState
	if err := state.init(eglState); err != nil {
		panic(err)
	}

	go userAppCode(ctrl)
	Outer: for {
		state.scene.Render()
		egl.SwapBuffers(eglState.Display, eglState.Surface)
		select {
		case <-ctrl.draw: break
		case <-ctrl.exit:
			egl.DestroySurface(eglState.Display, eglState.Surface)
			egl.DestroyContext(eglState.Display, eglState.Context)
			egl.Terminate(eglState.Display)
			break Outer
		}
	}
}

func userAppCode(ctrl *controlCh) {
	// TODO
}

// A render state includes information about the 3d world and the EGL
// state (rendering surfaces, etc.)
type renderState struct {
	scene *Scene
}

func (state *renderState) init(eglState *platform.EGLState) error {
	if ok := egl.MakeCurrent(eglState.Display, eglState.Surface, eglState.Surface, eglState.Context); !ok {
		return egl.NewError(egl.GetError())
	}

	state.scene = NewScene()

	if err := state.scene.AttachTextureFromFile("Kerker.png"); err != nil {
		return err
	}

	return nil
}
