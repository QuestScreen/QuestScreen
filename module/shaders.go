package module

import (
	gl "github.com/remogatto/opengles2"
	"log"
	"reflect"
	"unicode"
	"unicode/utf8"
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

// paramsTarget shall be a struct like this:
// struct {
//   AttributeIds struct {
//     attrName uint32
//     otherAttrName uint32
//     ...
//   }
//   UniformIds struct {
//     unifName uint32
//     ...
//   }
//   ...
// }
// fields of AttributeIds and UniformIds are to have the same name as the
// parameters in the shaders.
func CreateProgram(vertexShader, fragmentShader string, paramsTarget interface{}) uint32 {
	program := gl.CreateProgram()
	vShaderId := compileShader(gl.VERTEX_SHADER, vertexShader)
	fShaderId := compileShader(gl.FRAGMENT_SHADER, fragmentShader)
	gl.AttachShader(program, vShaderId)
	gl.AttachShader(program, fShaderId)
	gl.LinkProgram(program)
	checkProgramLinkStatus(program)
	loadProgramParameters(paramsTarget, program)
	return program
}

func toCamelCase(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToLower(r)) + s[size:]
}

func loadProgramParameters(v interface{}, program uint32) {
	target := reflect.ValueOf(v).Elem()
	for i := 0; i < target.NumField(); i++ {
		typeValue := target.Type().Field(i)
		if typeValue.Name == "AttributeIds" {
			attributesTarget := target.Field(i).Type()
			for j := 0; j < attributesTarget.NumField(); j++ {
				name := toCamelCase(attributesTarget.Field(j).Name)
				id := gl.GetAttribLocation(program, name)
				if id == 1<<32 - 1 { panic("unknown attribute: " + name) }
				target.FieldByIndex([]int{i,j}).SetUint(uint64(id))
			}
		} else if typeValue.Name == "UniformIds" {
			uniformsTarget := target.Field(i).Type()
			for j := 0; j < uniformsTarget.NumField(); j++ {
				name := toCamelCase(uniformsTarget.Field(j).Name)
				id := gl.GetUniformLocation(program, name)
				if id == 1<<32 - 1 { panic("unknown uniform: " + name) }
				target.FieldByIndex([]int{i,j}).SetUint(uint64(id))
			}
		}
	}
}
