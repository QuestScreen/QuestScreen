package module

// #include<freetype.h>
import "C"

import (
	"errors"
	gl "github.com/remogatto/opengles2"
	"unsafe"
)

var lib C.FT_Library = nil

type TextRenderProgram struct {
	id           uint32
	AttributeIds struct {
		Vertex uint32
	}
	UniformIds struct {
		CharacterInfo, Transformation, TextSampler uint32
	}
}

type LoadedCharacter struct {
	width, yMin, yMax, advance, left int
	textureId uint32
}

type TextRenderer struct {
	program TextRenderProgram
	loadedCharacters map[rune]LoadedCharacter
	fontFace C.FT_Face
}

func (t *TextRenderer) Init(common *SceneCommon, fontPath string) error {
	if lib == nil {
		if C.FT_Init_FreeType(&lib) {
			return errors.New("failed to initialize FreeType library")
		}
	}
	cpath := C.CString(fontPath)
	defer C.free(unsafe.Pointer(cpath))

	if err := C.FT_New_Face(lib, cpath, 0, &t.fontFace); err != 0 {
		if err == C.FT_Err_Unknown_File_Format {
			return errors.New("not a font file: " + fontPath)
		} else {
			return errors.New("could not load file: " + fontPath)
		}
	}

	t.program.id = CreateProgram(`
			#version 101
			precision mediump float;
			attribute vec2 vertex;
			uniform vec4 characterInfo;
			uniform mat4 transformation;
			varying vec2 textureCoords;
			void main() {
				vec2 translated = vec2(
					characterInfo.z * vertex.x + characterInfo.x,
					characterInfo.w * vertex.y + characterInfo.y);
				gl_Position = transformation * vec4(translated, 0.0, 1.0);
				textureCoords = vec2(vertex.x, 1.0 - vertex.y);
			}`,`
			#version 101
			precision mediump float;
			varying vec2 textureCoords;
			uniform sampler2D textSampler;
			void main() {
				float alpha = texture(textSampler, textureCoords).r;
				gl_FragColor.a = alpha;
			}`, &t.program)
	return nil
}

func (t *TextRenderer) characterData(codePoint rune) LoadedCharacter {
	if val, ok := t.loadedCharacters[codePoint]; ok {
		return val
	}
	charIndex := C.FT_Get_Char_Index (t.fontFace, codePoint)
	if charIndex == 0 {
		charIndex = C.FT_Get_Char_Index (t.fontFace, uint64('?'))
	}
	C.FT_Load_Glyph(t.fontFace, charIndex, C.FT_LOAD_RENDER)
	glyphSlot := t.fontFace.glyph
	C.FT_Render_Glyph(glyphSlot, C.FT_RENDER_MODE_MONO)
	var ret LoadedCharacter
	ret.width = glyphSlot.bitmap.width
	ret.yMin = glyphSlot.bitmap_top - glyphSlot.bitmap.rows
	ret.yMax = glyphSlot.bitmap_top
	ret.advance = glyphSlot.advance.x / 64
	ret.left = glyphSlot.bitmap_left
	gl.GenTextures(1, &ret.textureId)
	gl.BindTexture(gl.TEXTURE_2D, ret.textureId)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	var oldUnpackAlignment int32
	gl.GetIntegerv(gl.UNPACK_ALIGNMENT, &oldUnpackAlignment)
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 1)
	defer gl.PixelStorei(gl.UNPACK_ALIGNMENT, oldUnpackAlignment)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.ALPHA, glyphSlot.bitmap.width,
		glyphSlot.bitmap.rows, 0, gl.ALPHA, gl.UNSIGNED_BYTE,
		glyphSlot.bitmap.buffer)
	t.loadedCharacters[codePoint] = ret
	return ret
}

func (t *TextRenderer) CalculateDimensions(str string) (width int, yMin int, yMax int) {
	width = 0
	yMin = 0
	yMax = 0
	for _, codePoint := range str {
		data := t.characterData(codePoint)
		width += data.advance
		if data.yMin < yMin { yMin = data.yMin }
		if data.yMax > yMax { yMax = data.yMax }
	}
	return
}

func (t *TextRenderer) ToSizedTexture(str string, width, yMin, yMax int) uint32 {
	xOffset := 0
	height := yMax - yMin
	var frameBuf, target uint32
	gl.GenFramebuffers(1, &frameBuf)
	gl.GenTextures(1, &target)
	gl.BindFramebuffer(gl.FRAMEBUFFER, frameBuf)
	gl.BindTexture(gl.TEXTURE_2D, target)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.ALPHA, gl.Sizei(width), gl.Sizei(height), 0,
		gl.ALPHA, gl.UNSIGNED_BYTE, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	var oldViewport [4]int32
	gl.GetIntegerv(gl.VIEWPORT, &oldViewport[0])
	defer gl.Viewport(oldViewport[0], oldViewport[1], gl.Sizei(oldViewport[2]), gl.Sizei(oldViewport[3]))
	gl.Viewport(0, 0, gl.Sizei(width), gl.Sizei(height))
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, target, 0)
	defer gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, 0, 0)
	gl.ActiveTexture(0)
	gl.UseProgram(t.program.id)
	gl.EnableVertexAttribArray(0)
	defer gl.DisableVertexAttribArray(0)
	gl.Uniform1i(int32(t.program.UniformIds.TextSampler), 0)
	gl.Uniform1f(int32(t.program.UniformIds.Transformation), /* TODO: matrix */)
	// square_array.bind (?)
	// array_buffer.bind (?)
	// TODO: clear color?
	gl.Clear(gl.COLOR_BUFFER_BIT)
	for _, codePoint := range str {
		data := t.characterData(codePoint)
		gl.Uniform4f(int32(t.program.UniformIds.CharacterInfo), float32(xOffset + data.left),
			float32(data.yMin - yMin), float32(data.width), float32(data.yMax - data.yMin))
		gl.BindTexture(gl.TEXTURE_2D, data.textureId)
		// TODO: draw arrays
		xOffset += data.advance
	}
	gl.Flush()
	gl.Finish()

	return target
}

func (t *TextRenderer) ToTexture(str string) uint32 {
	width, yMin, yMax := t.CalculateDimensions(str)
	return t.ToSizedTexture(str, width, yMin, yMax)
}