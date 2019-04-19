package main

import (
  "github.com/go-gl/glfw/v3.2/glfw"
  "golang.org/x/mobile/gl"
  "runtime"
)

type controlCh struct {
  exit     chan struct{}
  draw     chan struct{}
}

func newControlCh() *controlCh {
  return &controlCh{
    exit:     make(chan struct {}),
    draw:     make(chan struct {}),
  }
}

func init() {
  runtime.LockOSThread()
}

func main() {
  if err := glfw.Init(); err != nil {
    panic(err)
  }
  defer glfw.Terminate()

  glfw.WindowHint(glfw.ClientAPI, glfw.OpenGLESAPI)
  glfw.WindowHint(glfw.ContextCreationAPI, glfw.NativeContextAPI)

  window, err := glfw.CreateWindow(640, 480, "Testing", nil, nil)
  if err != nil {
    panic(err)
  }

  window.MakeContextCurrent()
  glctx, worker := gl.NewContext()
  ctrl := newControlCh()

  workAvailable := worker.WorkAvailable()
  go userAppCode(glctx, ctrl, window)
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

func userAppCode(glctx gl.Context, ctrl *controlCh, window *glfw.Window) {
  var state renderState
  if err := state.init(window, glctx); err != nil {
    panic(err)
  }

  var exit = false

  window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
    if key == glfw.KeyEscape {
      exit = true
    }
  })

  for !exit {
    state.scene.Render(glctx)
    glfw.WaitEvents()
  }
  ctrl.exit<- struct{}{}
}

// A render state includes information about the 3d world and the EGL
// state (rendering surfaces, etc.)
type renderState struct {
  scene *Scene
}

func (state *renderState) init(window *glfw.Window, glctx gl.Context) error {
  //width, height := window.GetSize()

  state.scene = NewScene(glctx)

  if err := state.scene.AttachTextureFromFile("Kerker.png", glctx); err != nil {
    return err
  }

  return nil
}
