package title

import (
	"log"
	"time"

	"github.com/QuestScreen/api/colors"
	"github.com/QuestScreen/api/fonts"
	"github.com/QuestScreen/api/modules"
	"github.com/QuestScreen/api/render"
	"github.com/QuestScreen/api/server"
)

type titleConfig struct {
	Font       *fonts.Config      `yaml:"font"`
	Background *colors.Background `yaml:"background"`
}

type changeRequest struct {
	caption string
}

// The Title module draws a title box at the top of the screen.
type Title struct {
	*titleConfig
	curTitleText string
	curTitle     render.Image
	newTitle     render.Image
	curYOffset   int32
	swapped      bool
}

const (
	singleDuration = time.Second / 3
	waitTime       = time.Millisecond * 100
)

func newRenderer(
	r render.Renderer, ms server.MessageSender) (modules.Renderer, error) {
	return &Title{}, nil
}

// Descriptor describes the Title module
var Descriptor = modules.Module{
	Name:          "Scene Title",
	ID:            "title",
	EndpointPaths: []string{""},
	DefaultConfig: &titleConfig{Font: &fonts.Config{
		FamilyIndex: 0, Size: fonts.Heading, Style: fonts.Bold},
		Background: colors.NewBackground(
			colors.RGBA{R: 255, G: 255, B: 255, A: 255})},
	CreateRenderer: newRenderer, CreateState: newState,
}

func (t *Title) genTitleTexture(r render.Renderer, text string) render.Image {
	tex := r.RenderText(text, *t.titleConfig.Font)
	if tex.IsEmpty() {
		log.Println("failed to render title text")
		return tex
	}
	defer r.FreeImage(&tex)

	window := r.OutputSize()
	resWidth, resHeight := tex.Width, tex.Height
	if resWidth > window.Width*2/3 {
		scaleFactor := float32(window.Width*2/3) / float32(resWidth)
		resHeight = int32(float32(resHeight) * scaleFactor)
		resWidth = window.Width * 2 / 3
	}
	unit := r.Unit()
	canvas, inner := r.CreateCanvas(resWidth+4*unit, resHeight,
		*t.titleConfig.Background, render.West|render.East|render.South)
	frame := inner.Position(
		resWidth, resHeight, render.Center, render.Middle)
	tex.Draw(r, frame, 255)

	ret := canvas.Finish()
	return ret
}

// InitTransition initializes a transition.
func (t *Title) InitTransition(r render.Renderer,
	data interface{}) time.Duration {
	req := data.(*changeRequest)
	t.curTitleText = req.caption
	if t.curTitleText != "" {
		t.newTitle = t.genTitleTexture(r, t.curTitleText)
	}
	t.swapped = false
	return singleDuration*2 + waitTime
}

// TransitionStep advances the transition.
func (t *Title) TransitionStep(r render.Renderer, elapsed time.Duration) {
	if elapsed < time.Second/3 {
		if !t.curTitle.IsEmpty() {
			pos := render.TransitionCurve{Duration: singleDuration}.Cubic(elapsed)
			t.curYOffset = int32(pos * float32(t.curTitle.Height) * 1.1)
		}
	} else if elapsed < time.Second/3+time.Millisecond*100 {
		if !t.curTitle.IsEmpty() {
			t.curYOffset = t.curTitle.Height + 1
		}
	} else {
		if !t.swapped {
			r.FreeImage(&t.curTitle)
			t.curTitle = t.newTitle
			t.newTitle = render.Image{}
			t.swapped = true
		}
		if !t.curTitle.IsEmpty() {
			pos := render.TransitionCurve{Duration: singleDuration}.Cubic(
				elapsed - singleDuration - waitTime)
			t.curYOffset =
				int32((1.0 - pos) * float32(t.curTitle.Height) * 1.1)
		}
	}
}

// FinishTransition finalizes the transition.
func (t *Title) FinishTransition(r render.Renderer) {
	t.curYOffset = 0
}

// Render renders the module.
func (t *Title) Render(r render.Renderer) {
	if !t.curTitle.IsEmpty() {
		window := r.OutputSize()
		frame := window.Position(t.curTitle.Width,
			t.curTitle.Height, render.Center, render.Top)
		frame.Y += t.curYOffset
		t.curTitle.Draw(r, frame, 255)
	}
}

// Rebuild receives state data and config and immediately updates everything.
func (t *Title) Rebuild(r render.Renderer, data interface{},
	configVal interface{}) {
	t.titleConfig = configVal.(*titleConfig)
	if data != nil {
		req := data.(*changeRequest)
		t.curTitleText = req.caption
	}
	/*ctx.UpdateMask(&t.mask, *t.config.Background)*/

	t.curYOffset = 0
	r.FreeImage(&t.curTitle)
	r.FreeImage(&t.newTitle)
	if t.curTitleText != "" {
		t.curTitle = t.genTitleTexture(r, t.curTitleText)
	}
}
