package title

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/flyx/rpscreen/data"

	"github.com/flyx/rpscreen/display"
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
)

type config struct {
	Font *data.SelectableFont `config:"Font" yaml:"Font"`
}

type requestKind int

const (
	noRequest requestKind = iota
	changeRequest
	stateRequest
)

type requests struct {
	mutex   sync.Mutex
	kind    requestKind
	caption string
}

// The Title module draws a title box at the top of the screen.
type Title struct {
	*config
	state
	requests

	display    *display.Display
	store      *data.Store
	curTitle   *sdl.Texture
	newTitle   *sdl.Texture
	mask       *sdl.Texture
	curYOffset int32
}

// Init initializes the module.
func (t *Title) Init(display *display.Display, store *data.Store) error {
	if len(store.Fonts) == 0 {
		return errors.New("no fonts loaded")
	}
	t.state.owner = t
	t.display = display
	t.curTitle = nil
	t.store = store
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

func (t *Title) genTitleTexture(text string) *sdl.Texture {
	face := t.store.GetFontFace(t.config.Font)
	surface, err := face.RenderUTF8Blended(
		text, sdl.Color{R: 0, G: 0, B: 0, A: 230})
	if err != nil {
		log.Println(err)
		return nil
	}
	textTexture, err := t.display.Renderer.CreateTextureFromSurface(surface)
	if err != nil {
		log.Println(err)
		return nil
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
	ret, err := t.display.Renderer.CreateTexture(sdl.PIXELFORMAT_RGB888, sdl.TEXTUREACCESS_TARGET,
		textWidth+6*border, textHeight+2*border)
	if err != nil {
		panic(err)
	}
	t.display.Renderer.SetRenderTarget(ret)
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
	return ret
}

// InitTransition initializes a transition.
func (t *Title) InitTransition() time.Duration {
	t.requests.mutex.Lock()
	if t.requests.kind != changeRequest {
		t.requests.mutex.Unlock()
		return -1
	}
	caption := t.requests.caption
	t.requests.kind = noRequest
	t.requests.mutex.Unlock()
	t.newTitle = t.genTitleTexture(caption)
	return time.Second*2/3 + time.Millisecond*100
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
	return &config{}
}

// DefaultConfig returns the default configuration
func (t *Title) DefaultConfig() interface{} {
	return &config{Font: &data.SelectableFont{
		Family: t.display.Fonts[0].Name, FamilyIndex: 0, Size: data.HeadingFont,
		Style: data.Bold}}
}

// SetConfig sets the module's configuration
func (t *Title) SetConfig(value interface{}) bool {
	t.config = value.(*config)
	return true
}

// GetConfig retrieves the current configuration of the item.
func (t *Title) GetConfig() interface{} {
	return t.config
}

// GetState returns the current state.
func (t *Title) GetState() data.ModuleState {
	return &t.state
}

// RebuildState queries the new state through the channel and immediately
// updates everything.
func (t *Title) RebuildState() {
	t.requests.mutex.Lock()
	if t.requests.kind != stateRequest {
		panic("RebuildState() called on something else than stateRequest")
	}
	t.requests.kind = noRequest
	caption := t.requests.caption
	t.requests.mutex.Unlock()
	t.curYOffset = 0
	if t.curTitle != nil {
		t.curTitle.Destroy()
	}
	if t.newTitle != nil {
		t.newTitle.Destroy()
		t.newTitle = nil
	}
	t.curTitle = t.genTitleTexture(caption)
}
