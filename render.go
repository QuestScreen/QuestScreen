package main

import (
	"github.com/flyx/mobile/gl"
	"github.com/go-gl/mathgl/mgl32"
	"image"
	"image/png"
	"log"
	"os"
	"unsafe"
)

const (
	SizeOfFloat = 4
	TexCoordMax = 1
)

type BufferByte struct {
	data []byte
	gl.Buffer
}

func (b *BufferByte) Len() int {
	return len(b.data)
}

func (b *BufferByte) Data() []byte {
	return b.data
}

type BufferFloat struct {
	data []float32
	gl.Buffer
}

func (b *BufferFloat) Len() int {
	return len(b.data)
}

func (b *BufferFloat) Data() []float32 {
	return b.data
}

func NewBufferByte(data []byte, glctx gl.Context) *BufferByte {
	b := new(BufferByte)
	b.data = data
	b.Buffer = glctx.CreateBuffer()
	glctx.BindBuffer(gl.ARRAY_BUFFER, b.Buffer)
	glctx.BufferData(gl.ARRAY_BUFFER, b.data, gl.STATIC_DRAW)
	return b
}

func asByteSlice(floats []float32) []byte {
	lf := 4 * len(floats)

	// step by step
	pf := &(floats[0])                        // To pointer to the first byte of b
	up := unsafe.Pointer(pf)                  // To *special* unsafe.Pointer, it can be converted to any pointer
	pi := (*[1]byte)(up)                      // To pointer as byte array
	buf := (*pi)[:]                           // Creates slice to our array of 1 byte
	address := unsafe.Pointer(&buf)           // Capture the address to the slice structure
	lenAddr := uintptr(address) + uintptr(8)  // Capture the address where the length and cap size is stored
	capAddr := uintptr(address) + uintptr(16) // WARNING: This is fragile, depending on a go-internal structure.
	lenPtr := (*int)(unsafe.Pointer(lenAddr)) // Create pointers to the length and cap size
	capPtr := (*int)(unsafe.Pointer(capAddr)) //
	*lenPtr = lf                              // Assign the actual slice size and cap
	*capPtr = lf                              //

	return buf
}

func NewBufferFloat(data []float32, glctx gl.Context) *BufferFloat {
	b := new(BufferFloat)
	b.data = data
	b.Buffer = glctx.CreateBuffer()
	glctx.BindBuffer(gl.ARRAY_BUFFER, b.Buffer)
	glctx.BufferData(gl.ARRAY_BUFFER, asByteSlice(b.data), gl.STATIC_DRAW)
	return b
}

type VertexShader string
type FragmentShader string

func checkShaderCompileStatus(shader gl.Shader, glctx gl.Context) {
	stat := glctx.GetShaderi(shader, gl.COMPILE_STATUS)
	if stat == 0 {
		infoLog := glctx.GetShaderInfoLog(shader)
		log.Fatalf("Compile error in shader %s: \"%s\"\n", shader.String(), infoLog)
	}
}

func checkProgramLinkStatus(program gl.Program, glctx gl.Context) {
	stat := glctx.GetProgrami(program, gl.LINK_STATUS)
	if stat == 0 {
		infoLog := glctx.GetProgramInfoLog(program)
		log.Fatalf("Link error in program %s: \"%s\"\n", program.String(), infoLog)
	}
}

func compileShader(typeOfShader gl.Enum, source string, glctx gl.Context) gl.Shader {
	shader := glctx.CreateShader(typeOfShader)
	if shader.Value != 0 {
		glctx.ShaderSource(shader, source)
		glctx.CompileShader(shader)
		checkShaderCompileStatus(shader, glctx)
	}
	return shader
}

func (s VertexShader) Compile(glctx gl.Context) gl.Shader {
	shaderId := compileShader(gl.VERTEX_SHADER, (string)(s), glctx)
	return shaderId
}

func (s FragmentShader) Compile(glctx gl.Context) gl.Shader {
	shaderId := compileShader(gl.FRAGMENT_SHADER, (string)(s), glctx)
	return shaderId
}

func CreateProgram(fsh, vsh gl.Shader, glctx gl.Context) gl.Program {
	program := glctx.CreateProgram()
	glctx.AttachShader(program, fsh)
	glctx.AttachShader(program, vsh)
	glctx.LinkProgram(program)
	checkProgramLinkStatus(program, glctx)
	return program
}

type Scene struct {
	Vertices                      *BufferFloat
	Program                       gl.Program
	indices                       *BufferByte
	textureBuffer                 gl.Texture
	attrPos, attrColor, attrTexIn gl.Attrib
	uniformTexture                gl.Uniform
	uniformModel                  gl.Uniform
	uniformProjectionView         gl.Uniform
	model, projectionView         mgl32.Mat4
}

func NewScene(glctx gl.Context) *Scene {
	scene := new(Scene)
	scene.model = mgl32.Ident4()

	scene.Vertices = NewBufferFloat([]float32{
		// rectangle
		1, -1, 1, 1, TexCoordMax, 0,
		1, 1, 1, 1, TexCoordMax, TexCoordMax,
		-1, 1, 1, 1, 0, TexCoordMax,
		-1, -1, 1, 1, 0, 0,
	}, glctx)
	scene.indices = NewBufferByte([]byte{
		// rectangle
		0, 1, 2,
		2, 3, 0,
	}, glctx)

	fragmentShader := (FragmentShader)(`
			#version 300 es
			uniform sampler2D tx;
			in vec2 texOut;
			out vec4 fragColor;
			void main() {
				fragColor = texture(tx, texOut);
			}
        `)
	vertexShader := (VertexShader)(`
 				#version 300 es
				precision mediump float;
        uniform mat4 model;
        uniform mat4 projection_view;
        layout (location = 0) in vec4 pos;
        layout (location = 0) in vec2 texIn;
        out vec2 texOut;
        void main() {
          gl_Position = projection_view*model*pos;
          texOut = texIn;
        }
        `)

	fsh := fragmentShader.Compile(glctx)
	vsh := vertexShader.Compile(glctx)
	scene.Program = CreateProgram(fsh, vsh, glctx)

	glctx.UseProgram(scene.Program)
	scene.attrPos = glctx.GetAttribLocation(scene.Program, "pos")
	scene.attrColor = glctx.GetAttribLocation(scene.Program, "color")
	scene.attrTexIn = glctx.GetAttribLocation(scene.Program, "texIn")

	scene.uniformTexture = glctx.GetUniformLocation(scene.Program, "texture")
	scene.uniformModel = glctx.GetUniformLocation(scene.Program, "model")
	scene.uniformProjectionView = glctx.GetUniformLocation(scene.Program, "projection_view")

	glctx.EnableVertexAttribArray(scene.attrPos)
	glctx.EnableVertexAttribArray(scene.attrColor)
	glctx.EnableVertexAttribArray(scene.attrTexIn)

	return scene
}

func (s *Scene) AttachTextureFromFile(filename string, glctx gl.Context) error {
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
	s.AttachTexture(img, glctx)
	return nil
}

func (s *Scene) AttachTexture(img image.Image, glctx gl.Context) {
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
	s.textureBuffer = glctx.CreateTexture()
	glctx.BindTexture(gl.TEXTURE_2D, s.textureBuffer)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	glctx.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, width, height, gl.RGBA, gl.UNSIGNED_BYTE, buffer)
}

func (s *Scene) AttachTextureFromBuffer(buffer []byte, width, height int, glctx gl.Context) {
	s.textureBuffer = glctx.CreateTexture()
	glctx.BindTexture(gl.TEXTURE_2D, s.textureBuffer)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	glctx.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, width, height, gl.RGBA, gl.UNSIGNED_BYTE, buffer)
}

func (s *Scene) Render(glctx gl.Context) {
	glctx.UseProgram(s.Program)
	glctx.BindBuffer(gl.ARRAY_BUFFER, s.Vertices.Buffer)
	glctx.VertexAttribPointer(s.attrPos, 4, gl.FLOAT, false, SizeOfFloat*6, 0)
	glctx.VertexAttribPointer(s.attrTexIn, 2, gl.FLOAT, false, 6*SizeOfFloat, 4*SizeOfFloat)
	//glctx.VertexAttribPointer(c.attrColor, 4, gl.FLOAT, false, SizeOfFloat*8, SizeOfFloat*4)

	glctx.UniformMatrix4fv(s.uniformModel, (&s.model)[:])
	glctx.UniformMatrix4fv(s.uniformProjectionView, (&s.projectionView)[:])

	glctx.ActiveTexture(gl.TEXTURE0)
	glctx.BindTexture(gl.TEXTURE_2D, s.textureBuffer)
	glctx.Uniform1i(s.uniformTexture, 0)

	glctx.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, s.indices.Buffer)
	glctx.DrawElements(gl.TRIANGLES, s.indices.Len(), gl.UNSIGNED_BYTE, 0)
	glctx.Flush()
	glctx.Finish()
}
