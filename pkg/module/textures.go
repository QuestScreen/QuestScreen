package module

import (
	gl "github.com/remogatto/opengles2"
	"image"
	"image/png"
	"os"
)

type Texture struct {
	GlId  uint32
	Ratio float32
}

func LoadTextureFromFile(filename string) (Texture, error) {
	file, err := os.Open(filename)
	if err != nil {
		return Texture{}, err
	}
	defer file.Close()
	// Decode the image.
	img, err := png.Decode(file)
	if err != nil {
		return Texture{}, err
	}
	return LoadTexture(img), nil
}

func LoadTexture(img image.Image) Texture {
	bounds := img.Bounds()
	width, height := bounds.Size().X, bounds.Size().Y
	buffer := make([]byte, width*height*4)
	index := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			buffer[index] = byte(r)
			buffer[index+1] = byte(g)
			buffer[index+2] = byte(b)
			buffer[index+3] = byte(a)
			index += 4
		}
	}
	res := Texture{Ratio: float32(width) / float32(height)}
	gl.GenTextures(1, &res.GlId)
	gl.BindTexture(gl.TEXTURE_2D, res.GlId)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.Sizei(width), gl.Sizei(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Void(&buffer[0]))
	return res
}

func LoadTextureFromBuffer(buffer []byte, width, height int) Texture {
	res := Texture{Ratio: float32(width) / float32(height)}
	gl.GenTextures(1, &res.GlId)
	gl.BindTexture(gl.TEXTURE_2D, res.GlId)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.Sizei(width), gl.Sizei(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Void(&buffer[0]))
	return res
}
