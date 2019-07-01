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
	ClampColors [16]float32 // top | right | bottom | left
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

type colorsum struct {
	r, g, b, a uint64
	count uint64
}

func (c *colorsum) add(r, g, b, a byte) {
	c.r, c.g, c.b, c.a = uint64(r) + c.r, uint64(g) + c.g, uint64(b) + c.b, uint64(a) + c.a
	c.count++
}

func (c *colorsum) average() [4]float32 {
	var ret [4]float32
	ret[0] = float32(c.r / c.count) / 255.0
	ret[1] = float32(c.g / c.count) / 255.0
	ret[2] = float32(c.b / c.count) / 255.0
	ret[3] = float32(c.a / c.count) / 255.0
	return ret
}

func LoadTexture(img image.Image) Texture {
	bounds := img.Bounds()
	width, height := bounds.Size().X, bounds.Size().Y
	buffer := make([]byte, width*height*4)
	index := 0
	var leftSum, rightSum, topSum, bottomSum colorsum
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rawR, rawG, rawB, rawA := img.At(x, y).RGBA()
			r, g, b, a := byte(rawR / 256), byte(rawG / 256), byte(rawB / 256), byte(rawA / 256)
			buffer[index] = r
			buffer[index+1] = g
			buffer[index+2] = b
			buffer[index+3] = a
			index += 4
			if x == bounds.Min.X {
				leftSum.add(r, g, b, a)
			} else if x == bounds.Max.X - 1 {
				rightSum.add(r, g, b, a)
			}
			if y == bounds.Min.Y {
				bottomSum.add(r, g, b, a)
			} else if y == bounds.Max.Y - 1 {
				topSum.add(r, g, b, a)
			}
		}
	}
	res := Texture{Ratio: float32(width) / float32(height)}
	gl.GenTextures(1, &res.GlId)
	gl.BindTexture(gl.TEXTURE_2D, res.GlId)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.Sizei(width), gl.Sizei(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Void(&buffer[0]))
	tmp := topSum.average()
	copy(res.ClampColors[0:3], tmp[:])
	tmp = rightSum.average()
	copy(res.ClampColors[4:7], tmp[:])
	tmp = bottomSum.average()
	copy(res.ClampColors[8:11], tmp[:])
	tmp = leftSum.average()
	copy(res.ClampColors[12:15], tmp[:])
	return res
}

func LoadTextureFromBuffer(buffer []byte, width, height int) Texture {
	res := Texture{Ratio: float32(width) / float32(height)}
	gl.GenTextures(1, &res.GlId)
	gl.BindTexture(gl.TEXTURE_2D, res.GlId)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.Sizei(width), gl.Sizei(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Void(&buffer[0]))
	return res
}
