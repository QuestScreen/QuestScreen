package module

import (
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
)

func LoadTextureFromFile(filename string, renderer *sdl.Renderer) (*sdl.Texture, error) {
	tex, err := img.LoadTexture(renderer, filename)
	if err != nil {
		return nil, err
	} else {
		return tex, nil
	}
}
