package module

import (
	"github.com/go-gl/mathgl/mgl32"
	gl "github.com/remogatto/opengles2"
)

type SceneCommon struct {
	Ratio                float32
	TextureRenderProgram struct {
		GlId       uint32
		UniformIds struct {
			Texture, Model, ProjectionView uint32
		}
		AttributeIds struct {
			Pos, Color, TexIn uint32
		}
	}
	Square struct {
		Vertices FloatBuffer
		Indices  ByteBuffer
	}
}

var OrthoMatrix = mgl32.Ortho2D(-1.0, 1.0, -1.0, 1.0)

func (me *SceneCommon) DrawSquare(textureId uint32, model mgl32.Mat4) {
	gl.UseProgram(me.TextureRenderProgram.GlId)
	gl.EnableVertexAttribArray(me.TextureRenderProgram.AttributeIds.Pos)
	defer gl.DisableVertexAttribArray(me.TextureRenderProgram.AttributeIds.Pos)
	gl.EnableVertexAttribArray(me.TextureRenderProgram.AttributeIds.Color)
	defer gl.DisableVertexAttribArray(me.TextureRenderProgram.AttributeIds.Color)
	gl.EnableVertexAttribArray(me.TextureRenderProgram.AttributeIds.TexIn)
	defer gl.DisableVertexAttribArray(me.TextureRenderProgram.AttributeIds.TexIn)

	gl.BindBuffer(gl.ARRAY_BUFFER, me.Square.Vertices.GlId)
	gl.VertexAttribPointer(me.TextureRenderProgram.AttributeIds.Pos, 4, gl.FLOAT, false, SizeOfFloat*6, gl.Void(nil))
	gl.VertexAttribPointer(me.TextureRenderProgram.AttributeIds.TexIn, 2, gl.FLOAT, false, 6*SizeOfFloat, gl.Void(uintptr(4*SizeOfFloat)))
	//gl.VertexAttribPointer(c.attrColor, 4, gl.FLOAT, false, SizeOfFloat*8, SizeOfFloat*4)

	gl.UniformMatrix4fv(int32(me.TextureRenderProgram.UniformIds.Model), 1, false, (*float32)(&model[0]))
	gl.UniformMatrix4fv(int32(me.TextureRenderProgram.UniformIds.ProjectionView), 1, false, (*float32)(&OrthoMatrix[0]))

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, textureId)
	gl.Uniform1i(int32(textureId), 0)

	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, me.Square.Indices.GlId)
	gl.DrawElements(gl.TRIANGLES, gl.Sizei(me.Square.Indices.ByteLen), gl.UNSIGNED_BYTE, gl.Void(nil))
}
