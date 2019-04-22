package background

import (
	"github.com/flyx/rpscreen/module"
	"github.com/go-gl/mathgl/mgl32"
	gl "github.com/remogatto/opengles2"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type Background struct {
	texture                          module.Texture
	reqTextureIndex, curTextureIndex int
	images                           []string
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

	me.reqTextureIndex = -1
	me.curTextureIndex = len(me.images)
	return err
}

func (me *Background) Render(common *module.SceneCommon) {
	if me.texture.GlId != 0 {
		model := mgl32.Ident4()
		if me.texture.Ratio > common.Ratio {
			model = mgl32.Scale3D(1, common.Ratio/me.texture.Ratio, 1)
		} else {
			model = mgl32.Scale3D(me.texture.Ratio/common.Ratio, 1, 1)
		}
		common.DrawSquare(me.texture.GlId, model)
	}
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
		http.Error(w, "404 not found: " + suffix, http.StatusNotFound)
		return false
	}
}

func (me *Background) ProcessUpdate(common *module.SceneCommon) {
	if me.reqTextureIndex != -1 {
		defer func() { me.reqTextureIndex = -1 }()
		if me.reqTextureIndex != me.curTextureIndex {
			if me.texture.GlId != 0 {
				gl.DeleteTextures(1, &me.texture.GlId)
				me.texture.GlId = 0
			}
			if me.reqTextureIndex < len(me.images) {
				name := me.images[me.reqTextureIndex]
				var err error
				me.texture, err = module.LoadTextureFromFile(common.DataDir + "/background/" + name)
				if err != nil {
					log.Println(err)
				}
			}
			me.curTextureIndex = me.reqTextureIndex
		}
	}
}
