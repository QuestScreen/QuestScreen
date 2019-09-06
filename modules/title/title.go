package title

import (
	"errors"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/flyx/rpscreen/data"

	"github.com/flyx/rpscreen/display"
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
)

type titleConfig struct {
	Font *data.SelectableFont `config:"Font" yaml:"Font"`
}

// The Title module draws a title box at the top of the screen.
type Title struct {
	display *display.Display
	// TODO: remove
	store      *data.Store
	config     *titleConfig
	reqName    string
	curTitle   *sdl.Texture
	newTitle   *sdl.Texture
	mask       *sdl.Texture
	curFont    data.SelectableFont
	curYOffset int32
}

// Init initializes the module.
func (t *Title) Init(display *display.Display, store *data.Store) error {
	if len(store.Fonts) == 0 {
		return errors.New("no fonts loaded")
	}
	t.display = display
	t.store = store
	t.curTitle = nil
	t.curFont = data.SelectableFont{Family: store.Fonts[0].Name,
		FamilyIndex: 0, Size: data.HeadingFont, Style: data.Bold}
	var err error
	maskPath := store.GetFilePath(t, "", "mask.png")
	if maskPath != "" {
		t.mask, err = img.LoadTexture(display.Renderer, maskPath)
		if err != nil {
			log.Println(err)
			t.mask = nil
		}
	} else {
		t.mask = nil
	}
	return nil
}

// Name returns "Scene Title"
func (*Title) Name() string {
	return "Scene Title"
}

// InternalName returns "title"
func (*Title) InternalName() string {
	return "title"
}

// UI generates the HTML UI of the module.
func (t *Title) UI() template.HTML {
	var builder display.UIBuilder
	builder.StartForm(t, "set", "Set Scene Title", false)
	builder.TextInput("Text", "title-text", "text", t.reqName)
	builder.SubmitButton("Update", "", true)
	builder.EndForm()
	return builder.Finish()
}

// EndpointHandler implements the endpoint handler of the module.
func (t *Title) EndpointHandler(suffix string, values url.Values, w http.ResponseWriter, returnPartial bool) bool {
	if suffix == "set" {
		textVal := values["text"]
		if len(textVal) == 0 {
			http.Error(w, "missing text value", http.StatusBadRequest)
			return false
		}
		text := textVal[0]
		t.reqName = text

		var returns display.EndpointReturn
		if returnPartial {
			returns = display.EndpointReturnEmpty
		} else {
			returns = display.EndpointReturnRedirect
		}
		display.WriteEndpointHeader(w, returns)
		return true
	}
	http.Error(w, "404 not found: "+suffix, http.StatusNotFound)
	return false
}

// InitTransition initializes a transition.
func (t *Title) InitTransition() time.Duration {
	var ret time.Duration = -1
	face := t.store.GetFontFace(t.config.Font)
	surface, err := face.RenderUTF8Blended(
		t.reqName, sdl.Color{R: 0, G: 0, B: 0, A: 230})
	if err != nil {
		log.Println(err)
		return -1
	}
	textTexture, err := t.display.Renderer.CreateTextureFromSurface(surface)
	if err != nil {
		log.Println(err)
		return -1
	}
	defer textTexture.Destroy()
	winWidth, _ := t.display.Window.GetSize()
	textWidth := surface.W
	textHeight := surface.H
	surface.Free()
	if textWidth > winWidth*2/3 {
		textHeight = textHeight * (winWidth * 2 / 3) / textWidth
		textWidth = winWidth * 2 / 3
	}
	border := t.display.DefaultBorderWidth
	t.newTitle, err = t.display.Renderer.CreateTexture(sdl.PIXELFORMAT_RGB888, sdl.TEXTUREACCESS_TARGET,
		textWidth+6*border, textHeight+2*border)
	t.display.Renderer.SetRenderTarget(t.newTitle)
	defer t.display.Renderer.SetRenderTarget(nil)
	t.display.Renderer.Clear()
	t.display.Renderer.SetDrawColor(0, 0, 0, 192)
	t.display.Renderer.FillRect(&sdl.Rect{X: 0, Y: 0, W: int32(textWidth + 6*border), H: int32(textHeight) + 2*border})
	t.display.Renderer.SetDrawColor(200, 173, 127, 255)
	t.display.Renderer.FillRect(&sdl.Rect{X: border, Y: 0, W: int32(textWidth + 4*border), H: int32(textHeight + border)})
	if t.mask != nil {
		_, _, maskWidth, maskHeight, _ := t.mask.Query()
		for x := int32(0); x < textWidth+6*border; x += maskWidth {
			for y := int32(0); y < textHeight+2*border; y += maskHeight {
				t.display.Renderer.Copy(t.mask, nil, &sdl.Rect{X: x, Y: y, W: maskWidth, H: maskHeight})
			}
		}
	}
	t.display.Renderer.Copy(textTexture, nil, &sdl.Rect{X: 3 * border, Y: 0, W: textWidth, H: textHeight})

	ret = time.Second*2/3 + time.Millisecond*100
	return ret
}

// TransitionStep advances the transition.
func (t *Title) TransitionStep(elapsed time.Duration) {
	if elapsed < time.Second/3 {
		if t.curTitle != nil {
			_, _, _, texHeight, _ := t.curTitle.Query()
			t.curYOffset = int32(float64(elapsed) / float64(time.Second/3) * float64(texHeight))
		}
	} else if elapsed < time.Second/3+time.Millisecond*100 {
		if t.newTitle != nil {
			_, _, _, texHeight, _ := t.curTitle.Query()
			t.curYOffset = texHeight + 1
		}
	} else {
		if t.newTitle != nil {
			_ = t.curTitle.Destroy()
			t.curTitle = t.newTitle
			t.newTitle = nil
		}
		if t.curTitle != nil {
			_, _, _, texHeight, _ := t.curTitle.Query()
			t.curYOffset = int32(float64(time.Second*2/3-(elapsed-time.Millisecond*100)) / float64(time.Second/3) * float64(texHeight))
		}
	}
}

// FinishTransition finalizes the transition.
func (t *Title) FinishTransition() {
	t.curYOffset = 0
	t.curFont = *t.config.Font
}

// Render renders the module.
func (t *Title) Render() {
	winWidth, _ := t.display.Window.GetSize()
	_, _, texWidth, texHeight, _ := t.curTitle.Query()

	dst := sdl.Rect{X: (winWidth - texWidth) / 2, Y: -t.curYOffset, W: texWidth, H: texHeight}
	_ = t.display.Renderer.Copy(t.curTitle, nil, &dst)
}

// EmptyConfig returns an empty configuration
func (*Title) EmptyConfig() interface{} {
	return &titleConfig{}
}

// DefaultConfig returns the default configuration
func (t *Title) DefaultConfig() interface{} {
	return &titleConfig{Font: &data.SelectableFont{
		Family: t.display.Fonts[0].Name, FamilyIndex: 0, Size: data.HeadingFont,
		Style: data.Bold}}
}

// SetConfig sets the module's configuration
func (t *Title) SetConfig(config interface{}) {
	t.config = config.(*titleConfig)
}

// GetConfig retrieves the current configuration of the item.
func (t *Title) GetConfig() interface{} {
	return t.config
}

// NeedsTransition returns true iff the currently used font is not the
// configured font
func (t *Title) NeedsTransition() bool {
	return t.curFont != *t.config.Font
}
