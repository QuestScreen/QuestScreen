package background

import (
	"github.com/flyx/rpscreen/module"
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Background struct {
	texture, newTexture              module.Texture
	reqTextureIndex, curTextureIndex int
	images                           []string
	curTextureSplit                  float32
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

	me.texture = module.Texture{Ratio: 1}
	me.newTexture = module.Texture{Ratio: 1}

	me.reqTextureIndex = -1
	me.curTextureIndex = len(me.images)
	me.curTextureSplit = 0
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

func (me *Background) EndpointHandler(suffix string, values url.Values, w http.ResponseWriter, returnPartial bool) bool {
	if suffix == "image" {
		value := values["value"][0]
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

func offsets(inRatio float32, outRatio float32, winWidth int32, winHeight int32) sdl.Rect {
	if inRatio > outRatio {
		return sdl.Rect{X: 0, Y: int32(float32(winHeight) * (1.0 - (outRatio / inRatio)) / 2.0),
			W: winWidth, H: int32(float32(winHeight) * (outRatio / inRatio))}
	} else {
		return sdl.Rect{X: int32(float32(winWidth) * (1.0 - (inRatio / outRatio)) / 2.0),
			Y: 0, W: int32(float32(winWidth) * (inRatio / outRatio)), H: winHeight}
	}
}

func (me *Background) InitTransition(common *module.SceneCommon) time.Duration {
	var ret time.Duration = -1
	if me.reqTextureIndex != -1 {
		if me.reqTextureIndex != me.curTextureIndex {
			if me.reqTextureIndex < len(me.images) {
				name := me.images[me.reqTextureIndex]
				tex, err := img.LoadTexture(common.Renderer, common.DataDir+"/background/"+name)
				if err != nil {
					log.Println(err)
					me.newTexture.Tex = nil
				} else {
					defer tex.Destroy()
					_, _, texWidth, texHeight, err := tex.Query()
					if err != nil {
						panic(err)
					}
					winWidth, winHeight := common.Window.GetSize()
					me.newTexture.Tex, err = common.Renderer.CreateTexture(sdl.PIXELFORMAT_RGB888, sdl.TEXTUREACCESS_TARGET,
						winWidth, winHeight)
					if err != nil {
						panic(err)
					}
					common.Renderer.SetRenderTarget(me.newTexture.Tex)
					defer common.Renderer.SetRenderTarget(nil)
					common.Renderer.Clear()
					common.Renderer.SetDrawColor(255, 255, 255, 255)
					common.Renderer.FillRect(&sdl.Rect{X: 0, Y: 0, W: int32(winWidth), H: int32(winHeight)})
					dst := offsets(float32(texWidth)/float32(texHeight), float32(common.Width) / float32(common.Height),
						winWidth, winHeight)
					common.Renderer.Copy(tex, nil, &dst)
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
	if me.texture.Tex != nil {
		me.texture.Tex.Destroy()
	}
	me.texture = me.newTexture
	me.curTextureSplit = 0.0
	me.newTexture = module.Texture{Ratio: 1}
}

func (me *Background) Render(common *module.SceneCommon) {
	if me.texture.Tex != nil || me.curTextureSplit != 0 {
		winWidth, winHeight := common.Window.GetSize()
		curSplit := int32(me.curTextureSplit * float32(winWidth))
		if me.texture.Tex != nil {
			rect := sdl.Rect{X: curSplit, Y: 0, W: winWidth - curSplit, H: winHeight}
			common.Renderer.Copy(me.texture.Tex, &rect, &rect)
		}
		if me.newTexture.Tex != nil {
			rect := sdl.Rect{X: 0, Y: 0, W: curSplit, H: winHeight}
			common.Renderer.Copy(me.newTexture.Tex, &rect, &rect)
		}
	}
}
