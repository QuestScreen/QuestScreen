package module

import (
	"github.com/veandco/go-sdl2/sdl"
)

// SceneCommon describes the current scene.
type SceneCommon struct {
	SharedData
	Renderer *sdl.Renderer
	Window   *sdl.Window
}
