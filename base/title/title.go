package title

import (
	"log"
	"time"

	"github.com/veandco/go-sdl2/img"

	"github.com/flyx/pnpscreen/api"

	"github.com/veandco/go-sdl2/sdl"
)

type config struct {
	Font *api.SelectableFont `config:"Font" yaml:"Font"`
}

type changeRequest struct {
	caption string
}

type fullRequest struct {
	caption string
	mask    api.Resource
}

// The Title module draws a title box at the top of the screen.
type Title struct {
	*config
	curTitleText string
	curTitle     *sdl.Texture
	newTitle     *sdl.Texture
	mask         *sdl.Texture
	curYOffset   int32
	swapped      bool
}

func newModule(renderer *sdl.Renderer,
	env api.StaticEnvironment) (api.Module, error) {
	return &Title{curTitle: nil}, nil
}

// Descriptor describes the Title module
var Descriptor = api.ModuleDescriptor{
	Name: "Scene Title",
	ID:   "title",
	ResourceCollections: []api.ResourceSelector{
		api.ResourceSelector{Subdirectory: "", Suffixes: nil}},
	Actions: []string{"set"},
	DefaultConfig: &config{Font: &api.SelectableFont{
		FamilyIndex: 0, Size: api.HeadingFont, Style: api.Bold}},
	CreateModule: newModule, CreateState: newState,
}

// Descriptor returns the descriptor of the Title module
func (*Title) Descriptor() *api.ModuleDescriptor {
	return &Descriptor
}

func (t *Title) genTitleTexture(ctx api.RenderContext, text string) *sdl.Texture {
	face := ctx.Env.Font(
		t.config.Font.FamilyIndex, t.config.Font.Style, t.config.Font.Size)
	surface, err := face.RenderUTF8Blended(
		text, sdl.Color{R: 0, G: 0, B: 0, A: 230})
	if err != nil {
		log.Println(err)
		return nil
	}
	textTexture, err := ctx.Renderer.CreateTextureFromSurface(surface)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer textTexture.Destroy()
	winWidth, _, _ := ctx.Renderer.GetOutputSize()
	textWidth := surface.W
	textHeight := surface.H
	surface.Free()
	if textWidth > winWidth*2/3 {
		textHeight = textHeight * (winWidth * 2 / 3) / textWidth
		textWidth = winWidth * 2 / 3
	}
	border := ctx.Env.DefaultBorderWidth()
	ret, err := ctx.Renderer.CreateTexture(sdl.PIXELFORMAT_RGB888, sdl.TEXTUREACCESS_TARGET,
		textWidth+6*border, textHeight+2*border)
	if err != nil {
		panic(err)
	}
	ctx.Renderer.SetRenderTarget(ret)
	defer ctx.Renderer.SetRenderTarget(nil)
	ctx.Renderer.Clear()
	ctx.Renderer.SetDrawColor(0, 0, 0, 192)
	ctx.Renderer.FillRect(&sdl.Rect{X: 0, Y: 0, W: int32(textWidth + 6*border), H: int32(textHeight) + 2*border})
	ctx.Renderer.SetDrawColor(200, 173, 127, 255)
	ctx.Renderer.FillRect(&sdl.Rect{X: border, Y: 0, W: int32(textWidth + 4*border), H: int32(textHeight + border)})
	if t.mask != nil {
		_, _, maskWidth, maskHeight, _ := t.mask.Query()
		for x := int32(0); x < textWidth+6*border; x += maskWidth {
			for y := int32(0); y < textHeight+2*border; y += maskHeight {
				ctx.Renderer.Copy(t.mask, nil, &sdl.Rect{X: x, Y: y, W: maskWidth, H: maskHeight})
			}
		}
	}
	ctx.Renderer.Copy(textTexture, nil, &sdl.Rect{X: 3 * border, Y: 0, W: textWidth, H: textHeight})
	return ret
}

// InitTransition initializes a transition.
func (t *Title) InitTransition(ctx api.RenderContext, data interface{}) time.Duration {
	req := data.(*changeRequest)
	t.curTitleText = req.caption
	if t.curTitleText != "" {
		t.newTitle = t.genTitleTexture(ctx, t.curTitleText)
	}
	t.swapped = false
	return time.Second*2/3 + time.Millisecond*100
}

// TransitionStep advances the transition.
func (t *Title) TransitionStep(ctx api.RenderContext, elapsed time.Duration) {
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
func (t *Title) FinishTransition(ctx api.RenderContext) {
	t.curYOffset = 0
}

// Render renders the module.
func (t *Title) Render(ctx api.RenderContext) {
	winWidth, _, _ := ctx.Renderer.GetOutputSize()
	_, _, texWidth, texHeight, _ := t.curTitle.Query()

	dst := sdl.Rect{X: (winWidth - texWidth) / 2, Y: -t.curYOffset, W: texWidth, H: texHeight}
	_ = ctx.Renderer.Copy(t.curTitle, nil, &dst)
}

// SetConfig sets the module's configuration
func (t *Title) SetConfig(value interface{}) {
	t.config = value.(*config)
}

// RebuildState queries the new state through the channel and immediately
// updates everything.
func (t *Title) RebuildState(ctx api.RenderContext, data interface{}) {
	var maskFile api.Resource
	gotRequest := false
	if data != nil {
		req := data.(*fullRequest)
		t.curTitleText = req.caption
		gotRequest = true
		maskFile = req.mask
	}
	if gotRequest {
		if t.mask != nil {
			t.mask.Destroy()
		}
		if maskFile != nil {
			var err error
			t.mask, err = img.LoadTexture(ctx.Renderer, maskFile.Path())
			if err != nil {
				log.Println(err)
				t.mask = nil
			}
		} else {
			t.mask = nil
		}
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
		t.curTitle = t.genTitleTexture(ctx, t.curTitleText)
	}
}
