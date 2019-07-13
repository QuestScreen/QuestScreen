package sceneTitle

import (
	"bytes"
	"github.com/flyx/rpscreen/module"
	"github.com/veandco/go-sdl2/sdl"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type SceneTitle struct {
	reqName      string
	reqFontIndex int
	curTitle     *sdl.Texture
	fonts        []module.LoadedFont
	inputTempl   *template.Template
}

func (me *SceneTitle) Init(common *module.SceneCommon) error {
	me.curTitle = nil
	me.reqFontIndex = -1
	me.fonts = common.Fonts
	var err error
	me.inputTempl, err = template.New("input").Parse(
		`<input type="text" id="title-text" name="text" value="{{.Value}}"/>`)
	return err
}

func (*SceneTitle) Name() string {
	return "Scene Title"
}

func (me *SceneTitle) UI() template.HTML {
	var builder strings.Builder
	shownIndex := me.reqFontIndex
	if shownIndex == -1 {
		shownIndex = 0
	}
	builder.WriteString(`<form class="pure-form" action="/sceneTitle/set" method="post">
  <fieldset>
    <legend>Set Scene Title</legend>
    <input type="hidden" name="redirect" value="1"/>
    <label for="title-font">Font</label>
    <select id="title-font" name="font">`)
	for index, font := range me.fonts {
		builder.WriteString(`<option value="`)
		builder.WriteString(strconv.Itoa(index))
		if shownIndex == index {
			builder.WriteString(`" selected="selected">`)
		} else {
			builder.WriteString(`">`)
		}
		builder.WriteString(font.Name)
		builder.WriteString(`</option>`)
	}
	builder.WriteString(`</select>
    <label for="title-text">Text</label>
`)
	var buf bytes.Buffer
	if err := me.inputTempl.Execute(&buf, struct{ Value string }{Value: me.reqName}); err != nil {
		panic(err)
	}
	builder.WriteString(buf.String())
	builder.WriteString(`
    <button type="submit" class="pure-button pure-button-primary">Update</button>
  </fieldset>
</form>`)
	return template.HTML(builder.String())
}

func (*SceneTitle) EndpointPath() string {
	return "/sceneTitle/"
}

func (me *SceneTitle) EndpointHandler(suffix string, values url.Values, w http.ResponseWriter, returnPartial bool) bool {
	if suffix == "set" {
		fontVal := values["font"]
		if len(fontVal) == 0 {
			http.Error(w, "missing font index", http.StatusBadRequest)
			return false
		}
		font := fontVal[0]
		textVal := values["text"]
		if len(textVal) == 0 {
			http.Error(w, "missing text value", http.StatusBadRequest)
			return false
		}
		text := textVal[0]
		fontIndex, err := strconv.Atoi(font)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return false
		}
		if fontIndex < 0 || fontIndex >= len(me.fonts) {
			http.Error(w, "font index out of range", http.StatusBadRequest)
			return false
		}
		me.reqFontIndex = fontIndex
		me.reqName = text

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

func (me *SceneTitle) InitTransition(common *module.SceneCommon) time.Duration {
	var ret time.Duration = -1
	if me.reqFontIndex != -1 {
		if me.curTitle != nil {
			me.curTitle.Destroy()
		}
		surface, err := common.Fonts[me.reqFontIndex].Font.RenderUTF8Blended(
			me.reqName, sdl.Color{255, 0, 0, 255})
		if err != nil {
			log.Println(err)
			return -1
		}
		textTexture, err := common.Renderer.CreateTextureFromSurface(surface)
		surface.Free()
		if err != nil {
			log.Println(err)
			return -1
		}
		defer textTexture.Destroy()
		winWidth, _ := common.Window.GetSize()
		textWidth := surface.W
		textHeight := surface.H
		if textWidth > winWidth*2/3 {
			textHeight = textHeight * (winWidth * 2 / 3) / textWidth
			textWidth = winWidth * 2 / 3
		}
		me.curTitle, err = common.Renderer.CreateTexture(sdl.PIXELFORMAT_RGB888, sdl.TEXTUREACCESS_TARGET,
			textWidth, textHeight)
		common.Renderer.SetRenderTarget(me.curTitle)
		defer common.Renderer.SetRenderTarget(nil)
		common.Renderer.Clear()
		common.Renderer.SetDrawColor(255, 255, 0, 255)
		common.Renderer.FillRect(&sdl.Rect{X: 0, Y: 0, W: int32(textWidth), H: int32(textHeight)})
		common.Renderer.Copy(textTexture, nil, nil)
		ret = 0
	}
	return ret
}

func (me *SceneTitle) TransitionStep(common *module.SceneCommon, elapsed time.Duration) {
}

func (me *SceneTitle) FinishTransition(common *module.SceneCommon) {

}

func (me *SceneTitle) Render(common *module.SceneCommon) {
	winWidth, _ := common.Window.GetSize()
	_, _, texWidth, texHeight, _ := me.curTitle.Query()
	dst := sdl.Rect{X: (winWidth - texWidth) / 2, Y: 0, W: texWidth, H: texHeight}
	common.Renderer.Copy(me.curTitle, nil, &dst)
}
