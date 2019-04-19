package main

import (
  "github.com/remogatto/egl/platform"
  "golang.org/x/mobile/gl"
  "runtime"
)

type controlCh struct {
  eglState chan *platform.EGLState
  exitUserApp chan struct{}
  exit     chan struct{}
  draw     chan struct{}
}

func newControlCh() *controlCh {
  return &controlCh{
    eglState: make(chan *platform.EGLState),
    exitUserApp: make(chan struct{}),
    exit:     make(chan struct {}),
    draw:     make(chan struct {}),
  }
}

func init() {
  runtime.LockOSThread()
}

func main() {
  glctx, worker := gl.NewContext()
  ctrl := newControlCh()
  initEGL(ctrl, 800, 600)

  workAvailable := worker.WorkAvailable()
  go userAppCode(glctx, ctrl)
  for {
    select {
    case <-workAvailable:
      worker.DoWork()
    case <-ctrl.draw:
      // ... platform-specific cgo call to draw screen
    case <-ctrl.exit:
      break
    }
  }
}

func userAppCode(glctx gl.Context, ctrl *controlCh) {
  var state renderState
  if err := state.init(glctx); err != nil {
    panic(err)
  }

  Outer: for {
    state.scene.Render(glctx)
    select {
    case <-ctrl.draw: break
    case <-ctrl.exitUserApp:
      break Outer
    }
  }
  ctrl.exit<- struct{}{}
}

// A render state includes information about the 3d world and the EGL
// state (rendering surfaces, etc.)
type renderState struct {
  scene *Scene
}

func (state *renderState) init(glctx gl.Context) error {
  state.scene = NewScene(glctx)

  if err := state.scene.AttachTextureFromFile("Kerker.png", glctx); err != nil {
    return err
  }

  return nil
}
