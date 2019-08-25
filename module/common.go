package module

import (
	"github.com/flyx/rpscreen/data"
	"github.com/veandco/go-sdl2/sdl"
)

// SceneCommon describes the current scene.
type SceneCommon struct {
	data.Store
	Renderer *sdl.Renderer
	Window   *sdl.Window
}
