package main

import (
	gl "github.com/remogatto/opengles2"
	"github.com/go-gl/mathgl/mgl32"
	"image"
	"image/png"
	"log"
	"os"
)

const (
	SizeOfFloat = 4
	TexCoordMax = 1
)

type BufferByte struct {
	data []byte
	buffer uint32
}

func (b *BufferByte) Len() int {
	return len(b.data)
}

func (b *BufferByte) Data() []byte {
	return b.data
}

type BufferFloat struct {
	data []float32
	buffer uint32
}

func (b *BufferFloat) Len() int {
	return len(b.data)
}

func (b *BufferFloat) Data() []float32 {
	return b.data
}

func NewBufferByte(data []byte) *BufferByte {
	b := new(BufferByte)
	b.data = data
	gl.GenBuffers(1, &b.buffer)
	gl.BindBuffer(gl.ARRAY_BUFFER, b.buffer)
	gl.BufferData(gl.ARRAY_BUFFER, gl.SizeiPtr(len(b.data)), gl.Void(&b.data[0]), gl.STATIC_DRAW)
	return b
}

func NewBufferFloat(data []float32) *BufferFloat {
	b := new(BufferFloat)
	b.data = data
	gl.GenBuffers(1, &b.buffer)
	gl.BindBuffer(gl.ARRAY_BUFFER, b.buffer)
	gl.BufferData(gl.ARRAY_BUFFER, gl.SizeiPtr(len(b.data)*SizeOfFloat), gl.Void(&b.data[0]), gl.STATIC_DRAW)
	return b
}

type VertexShader string
type FragmentShader string

func checkShaderCompileStatus(shader uint32) {
	var stat int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &stat)
	if stat == 0 {
		var length int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &length)
		infoLog := gl.GetShaderInfoLog(shader, gl.Sizei(length), nil)
		log.Fatalf("Compile error in shader %d: \"%s\"\n", shader, infoLog[:len(infoLog) - 1])
	}
}

func checkProgramLinkStatus(program uint32) {
	var stat int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &stat)
	if stat == 0 {
		var length int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &length)
		infoLog := gl.GetProgramInfoLog(program, gl.Sizei(length), nil)
		log.Fatalf("Link error in program %d: \"%s\"\n", program, infoLog[:len(infoLog)-1])
	}
}

func compileShader(typeOfShader gl.Enum, source string) uint32 {
	shader := gl.CreateShader(typeOfShader)
	if shader != 0 {
		gl.ShaderSource(shader, 1, &source, nil)
		gl.CompileShader(shader)
		checkShaderCompileStatus(shader)
	}
	return shader
}

func (s VertexShader) Compile() uint32 {
	shaderId := compileShader(gl.VERTEX_SHADER, (string)(s))
	return shaderId
}

func (s FragmentShader) Compile() uint32 {
	shaderId := compileShader(gl.FRAGMENT_SHADER, (string)(s))
	return shaderId
}

func CreateProgram(fsh, vsh uint32) uint32 {
	program := gl.CreateProgram()
	gl.AttachShader(program, fsh)
	gl.AttachShader(program, vsh)
	gl.LinkProgram(program)
	checkProgramLinkStatus(program)
	return program
}

type Scene struct {
	Vertices                      *BufferFloat
	Program                       uint32
	indices                       *BufferByte
	textureBuffer                 uint32
	attrPos, attrColor, attrTexIn uint32
	uniformTexture                uint32
	uniformModel                  uint32
	uniformProjectionView         uint32
	model, projectionView         mgl32.Mat4
}

func NewScene() *Scene {
	scene := new(Scene)
	scene.model = mgl32.Ident4()
	scene.projectionView = mgl32.Ortho2D(-1.0, 1.0, -1.0, 1.0)

	scene.Vertices = NewBufferFloat([]float32{
		// rectangle
		1, -1, 1, 1, TexCoordMax, 0,
		1, 1, 1, 1, TexCoordMax, TexCoordMax,
		-1, 1, 1, 1, 0, TexCoordMax,
		-1, -1, 1, 1, 0, 0,
	})
	scene.indices = NewBufferByte([]byte{
		// rectangle
		0, 1, 2,
		2, 3, 0,
	})

	fragmentShader := (FragmentShader)(`
			#version 101
			precision mediump float;
			uniform sampler2D tx;
			varying vec2 texOut;
			void main() {
				gl_FragColor = texture2D(tx, texOut);
				//gl_FragColor = vec4(1,0,0,1);
			}
        `)
	vertexShader := (VertexShader)(`
 				#version 101
				precision mediump float;
        uniform mat4 model;
        uniform mat4 projection_view;
        attribute vec4 pos;
        attribute vec2 texIn;
        varying vec2 texOut;
        void main() {
          gl_Position = projection_view*model*pos;
          texOut = texIn;
        }
        `)

	fsh := fragmentShader.Compile()
	vsh := vertexShader.Compile()
	scene.Program = CreateProgram(fsh, vsh)

	gl.UseProgram(scene.Program)
	scene.attrPos = gl.GetAttribLocation(scene.Program, "pos")
	scene.attrColor = gl.GetAttribLocation(scene.Program, "color")
	scene.attrTexIn = gl.GetAttribLocation(scene.Program, "texIn")

	scene.uniformTexture = gl.GetUniformLocation(scene.Program, "texture")
	scene.uniformModel = gl.GetUniformLocation(scene.Program, "model")
	scene.uniformProjectionView = gl.GetUniformLocation(scene.Program, "projection_view")

	gl.EnableVertexAttribArray(scene.attrPos)
	gl.EnableVertexAttribArray(scene.attrColor)
	gl.EnableVertexAttribArray(scene.attrTexIn)

	return scene
}

func (s *Scene) AttachTextureFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	// Decode the image.
	img, err := png.Decode(file)
	if err != nil {
		return err
	}
	s.AttachTexture(img)
	return nil
}

func (s *Scene) AttachTexture(img image.Image) {
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
	gl.GenTextures(1, &s.textureBuffer)
	gl.BindTexture(gl.TEXTURE_2D, s.textureBuffer)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.Sizei(width), gl.Sizei(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Void(&buffer[0]))
}

func (s *Scene) AttachTextureFromBuffer(buffer []byte, width, height int) {
	gl.GenTextures(1, &s.textureBuffer)
	gl.BindTexture(gl.TEXTURE_2D, s.textureBuffer)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.Sizei(width), gl.Sizei(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Void(&buffer[0]))
}

func (s *Scene) Render() {
	gl.UseProgram(s.Program)
	gl.BindBuffer(gl.ARRAY_BUFFER, s.Vertices.buffer)
	gl.VertexAttribPointer(s.attrPos, 4, gl.FLOAT, false, SizeOfFloat*6, gl.Void(nil))
	gl.VertexAttribPointer(s.attrTexIn, 2, gl.FLOAT, false, 6*SizeOfFloat, gl.Void(uintptr(4*SizeOfFloat)))
	//gl.VertexAttribPointer(c.attrColor, 4, gl.FLOAT, false, SizeOfFloat*8, SizeOfFloat*4)

	gl.UniformMatrix4fv(int32(s.uniformModel), 1, false, (*float32)(&s.model[0]))
	gl.UniformMatrix4fv(int32(s.uniformProjectionView), 1, false, (*float32)(&s.projectionView[0]))

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, s.textureBuffer)
	gl.Uniform1i(int32(s.uniformTexture), 0)

	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, s.indices.buffer)
	gl.DrawElements(gl.TRIANGLES, gl.Sizei(s.indices.Len()), gl.UNSIGNED_BYTE, gl.Void(nil))
	gl.Flush()
	gl.Finish()
}
