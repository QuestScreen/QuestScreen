package background

import (
	"fmt"
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
		ProjectionView, OldTex /*, NewTex, OldScale, NewScale, YCut*/ uint32
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

	me.texture = module.Texture{Ratio: 1}
	me.newTexture = module.Texture{Ratio: 1}

	me.reqTextureIndex = -1
	me.curTextureIndex = len(me.images)
	me.curTextureSplit = 0
	/*me.program.id = module.CreateProgram(`
			#version 101
			precision mediump float;
			uniform mat4 projectionView;
			uniform vec2 oldScale;
			uniform vec2 newScale;
			attribute vec4 pos;
			attribute vec2 texIn;
			varying vec2 oldTexOut;
			varying vec2 newTexOut;
			varying float yPos;
			void main() {
				gl_Position = projectionView*pos;
				oldTexOut = vec2(texIn.x * oldScale.x, texIn.y * oldScale.y);
				newTexOut = vec2(texIn.x * newScale.x, texIn.y * newScale.y);
				yPos = texIn.y;
			}`, `
			#version 101
			precision mediump float;
			uniform sampler2D oldTex;
			uniform sampler2D newTex;
			uniform float yCut;
			varying vec2 oldTexOut;
			varying vec2 newTexOut;
			varying float yPos;
			void main() {
				if (yCut > 2.0) {
					gl_FragColor = yCut > yPos ? texture2D(newTex, newTexOut) : texture2D(oldTex, oldTexOut);
				} else {
					gl_FragColor = vec4(1,0,0,1);
				}
			}`, &me.program)*/
		me.program.id = module.CreateProgram(`
			#version 101
			precision mediump float;
			uniform mat4 projectionView;
			attribute vec4 pos;
			attribute vec2 texIn;
			varying vec2 texOut;
			void main() {
				gl_Position = projectionView*pos;
				texOut = texIn;
			}`, `
			#version 101
			precision mediump float;
			uniform sampler2D oldTex;
			varying vec2 texOut;
			void main() {
				gl_FragColor = texture2D(oldTex, texOut);
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
	fmt.Println("InitTransition with reqTexIndex=", me.reqTextureIndex)
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
			me.curTextureSplit = 0
			ret = time.Second
		}
		me.reqTextureIndex = -1
	}
	return ret
}

func (me *Background) TransitionStep(common *module.SceneCommon, elapsed time.Duration) {
	me.curTextureSplit = float32(elapsed) / float32(time.Second)
}

func (me *Background) FinishTransition(common *module.SceneCommon) {
	if me.texture.GlId != 0 {
		gl.DeleteTextures(1, &me.texture.GlId)
		me.texture.GlId = 0
	}
	me.texture = me.newTexture
	me.curTextureSplit = 0
	me.newTexture = module.Texture{Ratio: 1}
}

func (me *Background) Render(common *module.SceneCommon) {
	fmt.Println("rendering with textureSplit=", me.curTextureSplit)
	if me.texture.GlId != 0 || me.curTextureSplit != 0 {
		/*var oldScale, newScale [2]float32
		if me.texture.Ratio > common.Ratio {
			oldScale[0] = 1
			oldScale[1] = common.Ratio / me.texture.Ratio
		} else {
			oldScale[0] = me.texture.Ratio / common.Ratio
			oldScale[1] = 1
		}
		if me.newTexture.Ratio > common.Ratio {
			newScale[0] = 1
			newScale[1] = common.Ratio / me.newTexture.Ratio
		} else {
			newScale[0] = me.newTexture.Ratio / common.Ratio
			newScale[1] = 1
		}*/

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

		gl.UniformMatrix4fv(int32(me.program.UniformIds.ProjectionView), 1, false,
			(*float32)(&module.OrthoMatrix[0]))
		/*gl.Uniform2fv(int32(me.program.UniformIds.OldScale), 1, (*float32)(&oldScale[0]))
		gl.Uniform2fv(int32(me.program.UniformIds.NewScale), 1, (*float32)(&newScale[0]))*/

		gl.ActiveTexture(gl.TEXTURE0)
		if me.texture.GlId == 0 {
			fmt.Println("old texture empty")
			gl.BindTexture(gl.TEXTURE_2D, me.empty.GlId)
		} else {
			gl.BindTexture(gl.TEXTURE_2D, me.texture.GlId)
		}
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.Uniform1i(int32(me.program.UniformIds.OldTex), 0)

		/*gl.ActiveTexture(gl.TEXTURE1)
		if me.newTexture.GlId == 0 {
			fmt.Println("new texture empty")
			gl.BindTexture(gl.TEXTURE_2D, me.empty.GlId)
		} else {
			gl.BindTexture(gl.TEXTURE_2D, me.newTexture.GlId)
		}
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.Uniform1i(int32(me.program.UniformIds.NewTex), 1)

		gl.Uniform1f(int32(me.program.UniformIds.YCut), me.curTextureSplit)*/
		gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, common.Square.Indices.GlId)
		fmt.Println("drawing", common.Square.Indices.ByteLen, "elements")
		gl.DrawElements(gl.TRIANGLES, gl.Sizei(common.Square.Indices.ByteLen), gl.UNSIGNED_BYTE, gl.Void(nil))
		if err := gl.GetError(); err != 0 {
			panic(gl.GetString(err))
		}
	}
}
