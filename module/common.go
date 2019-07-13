package module

import (
	"github.com/veandco/go-sdl2/sdl"
)

type SceneCommon struct {
	Ratio    float32
	DataDir  string
	Renderer *sdl.Renderer
	Window   *sdl.Window
	Fonts    []LoadedFont
}
