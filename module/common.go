package module

import (
	"github.com/veandco/go-sdl2/sdl"
)

type SceneCommon struct {
	Width, Height int32
	DataDir  string
	Renderer *sdl.Renderer
	Window   *sdl.Window
	Fonts    []LoadedFont
}
