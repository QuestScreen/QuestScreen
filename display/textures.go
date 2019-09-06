package display

import (
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
)

// LoadTextureFromFile loads an image file to a texture.
// The list of supported formats depends on what has been compiled into SDL_img.
func LoadTextureFromFile(filename string, renderer *sdl.Renderer) (*sdl.Texture, error) {
	tex, err := img.LoadTexture(renderer, filename)
	if err != nil {
		return nil, err
	}
	return tex, nil
}
