package module

import (
	"github.com/veandco/go-sdl2/sdl"
)

type SceneCommon struct {
	SharedData
	Renderer           *sdl.Renderer
	Window             *sdl.Window
	Fonts              []LoadedFont
	DefaultBorderWidth int32
}
