package title

import (
	"log"
	"time"

	"github.com/QuestScreen/QuestScreen/api"

	"github.com/veandco/go-sdl2/sdl"
)

type config struct {
	Font       *api.SelectableFont               `yaml:"font"`
	Background *api.SelectableTexturedBackground `yaml:"background"`
}

type changeRequest struct {
	caption string
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

const (
	singleDuration = time.Second / 3
	waitTime       = time.Millisecond * 100
)

func newRenderer(renderer *sdl.Renderer) (api.ModuleRenderer, error) {
	return &Title{curTitle: nil}, nil
}

// Descriptor describes the Title module
var Descriptor = api.Module{
	Name:          "Scene Title",
	ID:            "title",
	EndpointPaths: []string{""},
	DefaultConfig: &config{Font: &api.SelectableFont{
		FamilyIndex: 0, Size: api.HeadingFont, Style: api.Bold},
		Background: &api.SelectableTexturedBackground{
			Primary:      api.RGBColor{Red: 255, Green: 255, Blue: 255},
			TextureIndex: -1,
		}},
	CreateRenderer: newRenderer, CreateState: newState,
}

// Descriptor returns the descriptor of the Title module
func (*Title) Descriptor() *api.Module {
	return &Descriptor
}

func setRGBDrawColor(renderer *sdl.Renderer, color api.RGBColor) error {
	return renderer.SetDrawColor(color.Red, color.Green, color.Blue, 255)
}

func (t *Title) genTitleTexture(ctx api.RenderContext, text string) *sdl.Texture {
	face := ctx.Font(
		t.config.Font.FamilyIndex, t.config.Font.Style, t.config.Font.Size)
	surface, err := face.RenderUTF8Blended(
		text, sdl.Color{R: 0, G: 0, B: 0, A: 230})
	if err != nil {
		log.Println(err)
		return nil
	}
	r := ctx.Renderer()
	textTexture, err := r.CreateTextureFromSurface(surface)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer textTexture.Destroy()
	winWidth, _, _ := r.GetOutputSize()
	textWidth := surface.W
	textHeight := surface.H
	surface.Free()
	if textWidth > winWidth*2/3 {
		textHeight = textHeight * (winWidth * 2 / 3) / textWidth
		textWidth = winWidth * 2 / 3
	}
	border := ctx.DefaultBorderWidth()
	ret, err := r.CreateTexture(sdl.PIXELFORMAT_RGB888,
		sdl.TEXTUREACCESS_TARGET, textWidth+6*border, textHeight+2*border)
	if err != nil {
		panic(err)
	}
	r.SetRenderTarget(ret)
	defer r.SetRenderTarget(nil)
	r.Clear()
	r.SetDrawColor(0, 0, 0, 192)
	r.FillRect(&sdl.Rect{X: 0, Y: 0,
		W: int32(textWidth + 6*border), H: int32(textHeight) + 2*border})
	setRGBDrawColor(r, t.config.Background.Primary)
	r.FillRect(&sdl.Rect{X: border, Y: 0,
		W: int32(textWidth + 4*border), H: int32(textHeight + border)})
	if t.mask != nil {
		// TODO: properly color the mask
		_, _, maskWidth, maskHeight, _ := t.mask.Query()
		for x := int32(0); x < textWidth+6*border; x += maskWidth {
			for y := int32(0); y < textHeight+2*border; y += maskHeight {
				r.Copy(t.mask, nil, &sdl.Rect{
					X: x, Y: y, W: maskWidth, H: maskHeight})
			}
		}
	}
	r.Copy(textTexture, nil,
		&sdl.Rect{X: 3 * border, Y: 0, W: textWidth, H: textHeight})
	return ret
}

// InitTransition initializes a transition.
func (t *Title) InitTransition(ctx api.RenderContext,
	data interface{}) time.Duration {
	req := data.(*changeRequest)
	t.curTitleText = req.caption
	if t.curTitleText != "" {
		t.newTitle = t.genTitleTexture(ctx, t.curTitleText)
	}
	t.swapped = false
	return singleDuration*2 + waitTime
}

// TransitionStep advances the transition.
func (t *Title) TransitionStep(ctx api.RenderContext, elapsed time.Duration) {
	if elapsed < time.Second/3 {
		if t.curTitle != nil {
			pos := api.TransitionCurve{Duration: singleDuration}.Cubic(elapsed)
			_, _, _, texHeight, _ := t.curTitle.Query()
			t.curYOffset = int32(pos * float32(texHeight))
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
			pos := api.TransitionCurve{Duration: singleDuration}.Cubic(
				elapsed - singleDuration - waitTime)
			t.curYOffset =
				int32((1.0 - pos) * float32(texHeight))
		}
	}
}

// FinishTransition finalizes the transition.
func (t *Title) FinishTransition(ctx api.RenderContext) {
	t.curYOffset = 0
}

// Render renders the module.
func (t *Title) Render(ctx api.RenderContext) {
	if t.curTitle != nil {
		r := ctx.Renderer()
		winWidth, _, _ := r.GetOutputSize()
		_, _, texWidth, texHeight, _ := t.curTitle.Query()

		dst := sdl.Rect{X: (winWidth - texWidth) / 2, Y: -t.curYOffset,
			W: texWidth, H: texHeight}
		err := r.Copy(t.curTitle, nil, &dst)
		if err != nil {
			log.Println(err)
		}
	}
}

// SetConfig sets the module's configuration
func (t *Title) SetConfig(value interface{}) {
	t.config = value.(*config)
}

// RebuildState queries the new state through the channel and immediately
// updates everything.
func (t *Title) RebuildState(ctx api.ExtendedRenderContext, data interface{}) {
	if data != nil {
		req := data.(*changeRequest)
		t.curTitleText = req.caption
	}
	if t.mask != nil {
		t.mask.Destroy()
		t.mask = nil
	}
	if t.config.Background.TextureIndex != -1 {
		var err error
		t.mask, err = ctx.LoadTexture(t.config.Background.TextureIndex,
			t.config.Background.Secondary)
		if err != nil {
			log.Println(err)
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
		t.curTitle = t.genTitleTexture(ctx, t.curTitleText)
	}
}
