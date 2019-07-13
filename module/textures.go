package module

import (
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
)

type Texture struct {
	Tex   *sdl.Texture
	Ratio float32
}

func LoadTextureFromFile(filename string, renderer *sdl.Renderer) (Texture, error) {
	res := Texture{}
	var err error
	res.Tex, err = img.LoadTexture(renderer, filename)
	if err != nil {
		return res, err
	}
	_, _, width, height, err := res.Tex.Query()
	if err != nil {
		return res, err
	}
	res.Ratio = float32(width) / float32(height)
	return res, nil
}
