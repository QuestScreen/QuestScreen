package module

import (
	"github.com/veandco/go-sdl2/sdl"
)

type SceneCommon struct {
	SharedData
	Width, Height int32
	Renderer      *sdl.Renderer
	Window        *sdl.Window
	Fonts         []LoadedFont
}
