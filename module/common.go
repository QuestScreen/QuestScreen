package module

import (
	"github.com/go-gl/mathgl/mgl32"
)

type SceneCommon struct {
	Ratio  float32
	Square struct {
		Vertices FloatBuffer
		Indices  ByteBuffer
	}
	DataDir string
}

var OrthoMatrix = mgl32.Ortho2D(-1.0, 1.0, -1.0, 1.0)
