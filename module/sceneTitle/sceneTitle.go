package sceneTitle

import (
	"bytes"
	"github.com/flyx/rpscreen/module"
	"github.com/veandco/go-sdl2/img"
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
	newTitle     *sdl.Texture
	mask         *sdl.Texture
	fonts        []module.LoadedFont
	inputTempl   *template.Template
	curYOffset   int32
}

func (me *SceneTitle) Init(common *module.SceneCommon) error {
	me.curTitle = nil
	me.reqFontIndex = -1
	me.fonts = common.Fonts
	var err error
	me.inputTempl, err = template.New("input").Parse(
		`<input type="text" id="title-text" name="text" value="{{.Value}}"/>`)
	if err != nil {
		return err
	}
	me.mask, err = img.LoadTexture(common.Renderer, common.DataDir+"/title/mask.png")
	if err != nil {
		me.mask = nil
	}
	return nil
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
	builder.WriteString(`<form class="pure-form" action="/sceneTitle/set" method="post" accept-charset="UTF-8">
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
		surface, err := common.Fonts[me.reqFontIndex].Font.RenderUTF8Blended(
			me.reqName, sdl.Color{0, 0, 0, 230})
		if err != nil {
			log.Println(err)
			return -1
		}
		textTexture, err := common.Renderer.CreateTextureFromSurface(surface)
		if err != nil {
			log.Println(err)
			return -1
		}
		defer textTexture.Destroy()
		winWidth, _ := common.Window.GetSize()
		textWidth := surface.W
		textHeight := surface.H
		surface.Free()
		if textWidth > winWidth*2/3 {
			textHeight = textHeight * (winWidth * 2 / 3) / textWidth
			textWidth = winWidth * 2 / 3
		}
		border := common.Height / 133
		me.newTitle, err = common.Renderer.CreateTexture(sdl.PIXELFORMAT_RGB888, sdl.TEXTUREACCESS_TARGET,
			textWidth + 6 * border, textHeight + 2 * border)
		common.Renderer.SetRenderTarget(me.newTitle)
		defer common.Renderer.SetRenderTarget(nil)
		common.Renderer.Clear()
		common.Renderer.SetDrawColor(0, 0, 0, 192)
		common.Renderer.FillRect(&sdl.Rect{X: 0, Y: 0, W: int32(textWidth + 6 * border), H: int32(textHeight) + 2 * border})
		common.Renderer.SetDrawColor(200, 173, 127, 255)
		common.Renderer.FillRect(&sdl.Rect{X: border, Y: 0, W: int32(textWidth + 4 * border), H: int32(textHeight + border)})
		if me.mask != nil {
			_, _, maskWidth, maskHeight, _ := me.mask.Query()
			for x := int32(0); x < textWidth + 6 * border; x += maskWidth {
				for y := int32(0); y < textHeight + 2 * border; y += maskHeight {
					common.Renderer.Copy(me.mask, nil, &sdl.Rect{X: x, Y: y, W: maskWidth, H: maskHeight})
				}
			}
		}
		common.Renderer.Copy(textTexture, nil, &sdl.Rect{X: 3 * border, Y: 0, W: textWidth, H: textHeight})

		ret = time.Second * 2 / 3 + time.Millisecond * 100
	}
	return ret
}

func (me *SceneTitle) TransitionStep(common *module.SceneCommon, elapsed time.Duration) {
	if elapsed < time.Second / 3 {
		if me.curTitle != nil {
			_, _, _, texHeight, _ := me.curTitle.Query()
			me.curYOffset = int32(float64(elapsed) / float64(time.Second/3) * float64(texHeight))
		}
	} else if elapsed < time.Second / 3 + time.Millisecond * 100 {
		if me.newTitle != nil {
			_, _, _, texHeight, _ := me.curTitle.Query()
		  me.curYOffset = texHeight + 1;
		}
	} else {
		if me.newTitle != nil {
			me.curTitle.Destroy()
			me.curTitle = me.newTitle
			me.newTitle = nil
		}
		if me.curTitle != nil {
			_, _, _, texHeight, _ := me.curTitle.Query()
			me.curYOffset = int32(float64(time.Second*2/3-(elapsed - time.Millisecond * 100)) / float64(time.Second/3) * float64(texHeight))
		}
	}
}

func (me *SceneTitle) FinishTransition(common *module.SceneCommon) {
	me.curYOffset = 0
}

func (me *SceneTitle) Render(common *module.SceneCommon) {
	winWidth, _ := common.Window.GetSize()
	_, _, texWidth, texHeight, _ := me.curTitle.Query()

	dst := sdl.Rect{X: (winWidth - texWidth) / 2, Y: -me.curYOffset, W: texWidth, H: texHeight}
	common.Renderer.Copy(me.curTitle, nil, &dst)
}
