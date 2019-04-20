package module

import (
	gl "github.com/remogatto/opengles2"
)

const (
	SizeOfFloat = 4
)

type (
	BufferData struct {
		ByteLen int
		GlId    uint32
	}
	ByteBuffer  BufferData
	FloatBuffer BufferData
)

func CreateByteBuffer(data []byte) ByteBuffer {
	b := ByteBuffer{ByteLen: len(data)}
	gl.GenBuffers(1, &b.GlId)
	gl.BindBuffer(gl.ARRAY_BUFFER, b.GlId)
	gl.BufferData(gl.ARRAY_BUFFER, gl.SizeiPtr(b.ByteLen), gl.Void(&data[0]), gl.STATIC_DRAW)
	return b
}

func CreateFloatBuffer(data []float32) FloatBuffer {
	b := FloatBuffer{ByteLen: len(data) * SizeOfFloat}
	gl.GenBuffers(1, &b.GlId)
	gl.BindBuffer(gl.ARRAY_BUFFER, b.GlId)
	gl.BufferData(gl.ARRAY_BUFFER, gl.SizeiPtr(b.ByteLen), gl.Void(&data[0]), gl.STATIC_DRAW)
	return b
}
