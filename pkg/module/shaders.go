package module

import (
	gl "github.com/remogatto/opengles2"
	"log"
)

type VertexShader string
type FragmentShader string

func checkShaderCompileStatus(shader uint32) {
	var stat int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &stat)
	if stat == 0 {
		var length int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &length)
		infoLog := gl.GetShaderInfoLog(shader, gl.Sizei(length), nil)
		log.Fatalf("Compile error in shader %d: \"%s\"\n", shader, infoLog[:len(infoLog)-1])
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
