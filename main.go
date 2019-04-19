package main

/*
#cgo LDFLAGS: -lGLESv2
*/
import "C"
import (
	gl "github.com/remogatto/opengles2"
	"github.com/flyx/egl"
	"github.com/flyx/egl/platform"
	"runtime"
	"fmt"
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
		fmt.Println("panic during init!")
		panic(err)
	}

	go userAppCode(ctrl)
	fmt.Println("main loop")
	Outer: for {
		state.scene.Render()
		select {
		case <-ctrl.draw: break
		case <-ctrl.exit:
			break Outer
		}
	}
	fmt.Println("/main loop")
}

func userAppCode(ctrl *controlCh) {
	fmt.Println("userAppCode init")
	// TODO
	fmt.Println("/userAppCode main loop")
}

// A render state includes information about the 3d world and the EGL
// state (rendering surfaces, etc.)
type renderState struct {
	scene *Scene
}

func (state *renderState) init(eglState *platform.EGLState) error {
	fmt.Println("init()")
	defer fmt.Println("/init()")

	if ok := egl.MakeCurrent(eglState.Display, eglState.Surface, eglState.Surface, eglState.Context); !ok {
		return egl.NewError(egl.GetError())
	}

	fmt.Println("Vendor: %s", gl.GetString(gl.VENDOR))
	fmt.Println("Renderer: %s", gl.GetString(gl.RENDERER))
	fmt.Println("Version: %s", gl.GetString(gl.VERSION))

	state.scene = NewScene()

	if err := state.scene.AttachTextureFromFile("Kerker.png"); err != nil {
		fmt.Println("could not attach texture file")
		return err
	}

	return nil
}
