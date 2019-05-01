package background

import (
	"github.com/flyx/rpscreen/module"
	gl "github.com/remogatto/opengles2"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type BackgroundProgram struct {
	id           uint32
	AttributeIds struct {
		Pos, TexIn uint32
	}
	UniformIds struct {
		ProjectionView, OldTex , NewTex, OldScale, NewScale, OldClampColors, NewClampColors, XCut uint32
	}
}

type Background struct {
	texture, newTexture, empty       module.Texture
	reqTextureIndex, curTextureIndex int
	images                           []string
	curTextureSplit                  float32
	program                          BackgroundProgram
}

func (me *Background) Init(common *module.SceneCommon) error {
	files, err := ioutil.ReadDir(common.DataDir + "/background")
	if err == nil {
		me.images = make([]string, 0, 64)
		for _, file := range files {
			if !file.IsDir() {
				me.images = append(me.images, file.Name())
			}
		}
	} else {
		log.Println(err)
	}
	me.empty = module.LoadTextureFromBuffer([]byte{0, 0, 0, 1}, 1, 1)
	me.empty.Ratio = common.Ratio

	me.texture = module.Texture{Ratio: 1}
	me.newTexture = module.Texture{Ratio: 1}

	me.reqTextureIndex = -1
	me.curTextureIndex = len(me.images)
	me.curTextureSplit = -0.1
	me.program.id = module.CreateProgram(`
			#version 101
			precision mediump float;
			uniform mat4 projectionView;
			uniform vec2 oldScale;
			uniform vec2 newScale;
			attribute vec4 pos;
			attribute vec2 texIn;
			varying vec2 oldTexOut;
			varying vec2 newTexOut;
			varying float xPos;
			void main() {
				gl_Position = projectionView*pos;
				oldTexOut = vec2(texIn.x * oldScale.x + (1.0 - oldScale.x) / 2.0, texIn.y * oldScale.y + (1.0 - oldScale.y) / 2.0);
				newTexOut = vec2(texIn.x * newScale.x + (1.0 - newScale.x) / 2.0, texIn.y * newScale.y + (1.0 - newScale.y) / 2.0);
				xPos = texIn.x;
			}`, `
			#version 101
			precision mediump float;
			uniform sampler2D oldTex;
			uniform vec4 oldClampColors[4];
			uniform sampler2D newTex;
			uniform vec4 newClampColors[4];
			uniform float xCut;
			varying vec2 oldTexOut;
			varying vec2 newTexOut;
			varying float xPos;

			vec4 texColor(sampler2D tex, vec4 clampColors[4], vec2 texPos) {
				if (texPos.x < 0.0) {
					return clampColors[3];
				} else if (texPos.x > 1.0) {
					return clampColors[1];
				} else if (texPos.y < 0.0) {
					return clampColors[2];
				} else if (texPos.y > 1.0) {
					return clampColors[0];
				} else {
					return texture2D(tex, texPos);
				}
			}

			void main() {
				float diff = ((xCut + 0.01) - xPos) * 50.0;
				if (diff <= 0.0) {
					gl_FragColor = texColor(oldTex, oldClampColors, oldTexOut);
				} else if (diff < 1.0) {
					gl_FragColor = diff * texColor(newTex, newClampColors, newTexOut) +
							(1.0 - diff) * texColor(oldTex, oldClampColors, oldTexOut);
				} else {
					gl_FragColor = texColor(newTex, newClampColors, newTexOut);
				}
			}`, &me.program)
	return err
}

func (*Background) Name() string {
	return "Background Image"
}

func (me *Background) UI() template.HTML {
	var builder strings.Builder
	shownIndex := me.reqTextureIndex
	if shownIndex == -1 {
		shownIndex = me.curTextureIndex
	}
	builder.WriteString(`<form class="pure-form" action="/background/image" method="post">
  <fieldset>
    <legend>Select Image</legend>
    <input type="hidden" name="redirect" value="1"/>
    <select id="image" name="value">
      <option value=""`)
	if shownIndex == len(me.images) {
		builder.WriteString(` selected="selected"`)
	}
	builder.WriteString(`>None</option>`)
	for index, name := range me.images {
		builder.WriteString(`<option value="`)
		builder.WriteString(strconv.Itoa(index))
		if shownIndex == index {
			builder.WriteString(`" selected="selected">`)
		} else {
			builder.WriteString(`">`)
		}
		builder.WriteString(name)
		builder.WriteString(`</option>`)
	}
	builder.WriteString(`</select>
    <button type="submit" class="pure-button pure-button-primary">Update</button>
  </fieldset>
</form>`)

	return template.HTML(builder.String())
}

func (*Background) EndpointPath() string {
	return "/background/"
}

func (me *Background) EndpointHandler(suffix string, value string, w http.ResponseWriter, returnPartial bool) bool {
	if suffix == "image" {
		if value == "" {
			me.reqTextureIndex = len(me.images)
		} else {
			index, err := strconv.Atoi(value)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return false
			}
			if index < 0 || index >= len(me.images) {
				http.Error(w, "image index out of range", http.StatusBadRequest)
				return false
			}
			me.reqTextureIndex = index
		}
		var returns module.EndpointReturn
		if returnPartial {
			returns = module.EndpointReturnEmpty
		} else {
			returns = module.EndpointReturnRedirect
		}
		module.WriteEndpointHeader(w, returns)
		return true
	} else {
		http.Error(w, "404 not found: "+suffix, http.StatusNotFound)
		return false
	}
}

func (me *Background) InitTransition(common *module.SceneCommon) time.Duration {
	var ret time.Duration = -1
	if me.reqTextureIndex != -1 {
		if me.reqTextureIndex != me.curTextureIndex {
			if me.reqTextureIndex < len(me.images) {
				name := me.images[me.reqTextureIndex]
				var err error
				me.newTexture, err = module.LoadTextureFromFile(common.DataDir + "/background/" + name)
				if err != nil {
					log.Println(err)
				}
			}
			me.curTextureIndex = me.reqTextureIndex
			me.curTextureSplit = -0.1
			ret = time.Second
		}
		me.reqTextureIndex = -1
	}
	return ret
}

func (me *Background) TransitionStep(common *module.SceneCommon, elapsed time.Duration) {
	me.curTextureSplit = (float32(elapsed) / float32(time.Second)) * 1.2 - 0.1
}

func (me *Background) FinishTransition(common *module.SceneCommon) {
	if me.texture.GlId != 0 {
		gl.DeleteTextures(1, &me.texture.GlId)
		me.texture.GlId = 0
	}
	me.texture = me.newTexture
	me.curTextureSplit = -0.1
	me.newTexture = module.Texture{Ratio: 1}
}

func setTexture(tex module.Texture, index int32, texUniformId uint32, clampUniformId uint32) {
	gl.ActiveTexture(gl.TEXTURE0 + gl.Enum(index))
	gl.BindTexture(gl.TEXTURE_2D, tex.GlId)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.Uniform1i(int32(texUniformId), index)
	gl.Uniform4fv(int32(clampUniformId), 4, (*float32)(&tex.ClampColors[0]))
}

func (me *Background) Render(common *module.SceneCommon) {
	if err := gl.GetError(); err != 0 {
		panic("GetError() not 0 at begin of render")
	}
	if me.texture.GlId != 0 || me.curTextureSplit != 0 {
		var oldScale, newScale [2]float32
		if me.texture.Ratio > common.Ratio {
			oldScale[0] = 1
			oldScale[1] = me.texture.Ratio / common.Ratio
		} else {
			oldScale[0] = common.Ratio / me.texture.Ratio
			oldScale[1] = 1
		}
		if me.newTexture.Ratio > common.Ratio {
			newScale[0] = 1
			newScale[1] = me.newTexture.Ratio / common.Ratio
		} else {
			newScale[0] = common.Ratio / me.texture.Ratio
			newScale[1] = 1
		}

		gl.UseProgram(me.program.id)
		gl.EnableVertexAttribArray(me.program.AttributeIds.Pos)
		defer gl.DisableVertexAttribArray(me.program.AttributeIds.Pos)
		gl.EnableVertexAttribArray(me.program.AttributeIds.TexIn)
		defer gl.DisableVertexAttribArray(me.program.AttributeIds.TexIn)

		gl.BindBuffer(gl.ARRAY_BUFFER, common.Square.Vertices.GlId)
		gl.VertexAttribPointer(me.program.AttributeIds.Pos, 4, gl.FLOAT, false,
			module.SizeOfFloat*6, gl.Void(nil))
		gl.VertexAttribPointer(me.program.AttributeIds.TexIn, 2, gl.FLOAT, false,
			6*module.SizeOfFloat, gl.Void(uintptr(4*module.SizeOfFloat)))
		if err := gl.GetError(); err != 0 { panic(uint32(err)) }

		gl.UniformMatrix4fv(int32(me.program.UniformIds.ProjectionView), 1, false,
			(*float32)(&module.OrthoMatrix[0]))
		gl.Uniform2fv(int32(me.program.UniformIds.OldScale), 1, (*float32)(&oldScale[0]))
		gl.Uniform2fv(int32(me.program.UniformIds.NewScale), 1, (*float32)(&newScale[0]))

		var tex = me.texture
		if tex.GlId == 0 { tex = me.empty }
		setTexture(tex, 0, me.program.UniformIds.OldTex, me.program.UniformIds.OldClampColors)

		tex = me.newTexture
		if tex.GlId == 0 { tex = me.empty }
		setTexture(tex, 1, me.program.UniformIds.NewTex, me.program.UniformIds.NewClampColors)

		gl.Uniform1f(int32(me.program.UniformIds.XCut), me.curTextureSplit)
		gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, common.Square.Indices.GlId)
		gl.DrawElements(gl.TRIANGLES, gl.Sizei(common.Square.Indices.ByteLen), gl.UNSIGNED_BYTE, gl.Void(nil))
		if err := gl.GetError(); err != 0 { panic(int32(err)) }
	}
}
