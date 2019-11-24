package title

import (
	"github.com/veandco/go-sdl2/img"
	"log"
	"sync"
	"time"

	"github.com/flyx/pnpscreen/api"

	"github.com/veandco/go-sdl2/sdl"
)

type config struct {
	Font *api.SelectableFont `config:"Font" yaml:"Font"`
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

	env          api.Environment
	moduleIndex  api.ModuleIndex
	curTitleText string
	curTitle     *sdl.Texture
	newTitle     *sdl.Texture
	mask         *sdl.Texture
	curYOffset   int32
	swapped      bool
}

// Init initializes the module.
func (t *Title) Init(renderer *sdl.Renderer, env api.Environment, index api.ModuleIndex) error {
	t.env = env
	t.moduleIndex = index
	t.state.owner = t
	t.curTitle = nil
	return nil
}

// Name returns "Scene Title"
func (*Title) Name() string {
	return "Scene Title"
}

// ID returns "title"
func (*Title) ID() string {
	return "title"
}

func (t *Title) genTitleTexture(renderer *sdl.Renderer, text string) *sdl.Texture {
	face := t.env.Font(
		t.config.Font.FamilyIndex, t.config.Font.Style, t.config.Font.Size)
	surface, err := face.RenderUTF8Blended(
		text, sdl.Color{R: 0, G: 0, B: 0, A: 230})
	if err != nil {
		log.Println(err)
		return nil
	}
	textTexture, err := renderer.CreateTextureFromSurface(surface)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer textTexture.Destroy()
	winWidth, _, _ := renderer.GetOutputSize()
	textWidth := surface.W
	textHeight := surface.H
	surface.Free()
	if textWidth > winWidth*2/3 {
		textHeight = textHeight * (winWidth * 2 / 3) / textWidth
		textWidth = winWidth * 2 / 3
	}
	border := t.env.DefaultBorderWidth()
	ret, err := renderer.CreateTexture(sdl.PIXELFORMAT_RGB888, sdl.TEXTUREACCESS_TARGET,
		textWidth+6*border, textHeight+2*border)
	if err != nil {
		panic(err)
	}
	renderer.SetRenderTarget(ret)
	defer renderer.SetRenderTarget(nil)
	renderer.Clear()
	renderer.SetDrawColor(0, 0, 0, 192)
	renderer.FillRect(&sdl.Rect{X: 0, Y: 0, W: int32(textWidth + 6*border), H: int32(textHeight) + 2*border})
	renderer.SetDrawColor(200, 173, 127, 255)
	renderer.FillRect(&sdl.Rect{X: border, Y: 0, W: int32(textWidth + 4*border), H: int32(textHeight + border)})
	if t.mask != nil {
		_, _, maskWidth, maskHeight, _ := t.mask.Query()
		for x := int32(0); x < textWidth+6*border; x += maskWidth {
			for y := int32(0); y < textHeight+2*border; y += maskHeight {
				renderer.Copy(t.mask, nil, &sdl.Rect{X: x, Y: y, W: maskWidth, H: maskHeight})
			}
		}
	}
	renderer.Copy(textTexture, nil, &sdl.Rect{X: 3 * border, Y: 0, W: textWidth, H: textHeight})
	return ret
}

// InitTransition initializes a transition.
func (t *Title) InitTransition(renderer *sdl.Renderer) time.Duration {
	t.requests.mutex.Lock()
	if t.requests.kind != changeRequest {
		t.requests.mutex.Unlock()
		return -1
	}
	t.curTitleText = t.requests.caption
	t.requests.kind = noRequest
	t.requests.mutex.Unlock()
	if t.curTitleText != "" {
		t.newTitle = t.genTitleTexture(renderer, t.curTitleText)
	}
	t.swapped = false
	return time.Second*2/3 + time.Millisecond*100
}

// TransitionStep advances the transition.
func (t *Title) TransitionStep(renderer *sdl.Renderer, elapsed time.Duration) {
	if elapsed < time.Second/3 {
		if t.curTitle != nil {
			_, _, _, texHeight, _ := t.curTitle.Query()
			t.curYOffset = int32(float64(elapsed) / float64(time.Second/3) * float64(texHeight))
		}
	} else if elapsed < time.Second/3+time.Millisecond*100 {
		if t.curTitle != nil {
			_, _, _, texHeight, _ := t.curTitle.Query()
			t.curYOffset = texHeight + 1
		}
	} else {
		if !t.swapped {
			if t.curTitle != nil {
				_ = t.curTitle.Destroy()
			}
			t.curTitle = t.newTitle
			t.newTitle = nil
			t.swapped = true
		}
		if t.curTitle != nil {
			_, _, _, texHeight, _ := t.curTitle.Query()
			t.curYOffset = int32(float64(time.Second*2/3-(elapsed-time.Millisecond*100)) / float64(time.Second/3) * float64(texHeight))
		}
	}
}

// FinishTransition finalizes the transition.
func (t *Title) FinishTransition(renderer *sdl.Renderer) {
	t.curYOffset = 0
}

// Render renders the module.
func (t *Title) Render(renderer *sdl.Renderer) {
	winWidth, _, _ := renderer.GetOutputSize()
	_, _, texWidth, texHeight, _ := t.curTitle.Query()

	dst := sdl.Rect{X: (winWidth - texWidth) / 2, Y: -t.curYOffset, W: texWidth, H: texHeight}
	_ = renderer.Copy(t.curTitle, nil, &dst)
}

// EmptyConfig returns an empty configuration
func (*Title) EmptyConfig() interface{} {
	return &config{}
}

// DefaultConfig returns the default configuration
func (t *Title) DefaultConfig() interface{} {
	return &config{Font: &api.SelectableFont{
		Family: t.env.FontCatalog()[0].Name(), FamilyIndex: 0,
		Size: api.HeadingFont, Style: api.Bold}}
}

// SetConfig sets the module's configuration
func (t *Title) SetConfig(value interface{}) {
	t.config = value.(*config)
}

// State returns the current state.
func (t *Title) State() api.ModuleState {
	return &t.state
}

// RebuildState queries the new state through the channel and immediately
// updates everything.
func (t *Title) RebuildState(renderer *sdl.Renderer) {
	t.requests.mutex.Lock()
	switch t.requests.kind {
	case stateRequest:
		t.curTitleText = t.requests.caption
	case noRequest:
		break
	default:
		panic("RebuildState() called on something else than stateRequest")
	}
	t.requests.kind = noRequest
	t.requests.mutex.Unlock()
	if t.mask != nil {
		t.mask.Destroy()
	}
	if len(t.state.resources) > 0 {
		var err error
		t.mask, err = img.LoadTexture(renderer, t.state.resources[0].Path())
		if err != nil {
			log.Println(err)
			t.mask = nil
		}
	} else {
		t.mask = nil
	}

	t.curYOffset = 0
	if t.curTitle != nil {
		t.curTitle.Destroy()
		t.curTitle = nil
	}
	if t.newTitle != nil {
		t.newTitle.Destroy()
		t.newTitle = nil
	}
	if t.curTitleText != "" {
		t.curTitle = t.genTitleTexture(renderer, t.curTitleText)
	}
}

// ResourceCollections returns a singleton list describing the selector for
// texture images.
func (t *Title) ResourceCollections() []api.ResourceSelector {
	return []api.ResourceSelector{
		api.ResourceSelector{Subdirectory: "", Suffixes: nil}}
}
