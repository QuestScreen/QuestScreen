package herolist

import (
	"time"

	"github.com/QuestScreen/api"
	"github.com/QuestScreen/api/config"
	"github.com/QuestScreen/api/modules"
	"github.com/QuestScreen/api/render"
	"github.com/QuestScreen/api/server"

	"github.com/veandco/go-sdl2/sdl"
)

type heroData struct {
	name, desc string
	visible    bool
}

type displayedHero struct {
	heroData
	box render.Image
}

type heroStatus int

const (
	resting heroStatus = iota
	showingAll
	hidingAll
	showingHero
	hidingHero
)

type mConfig struct {
	NameFont   *config.FontSelect       `yaml:"nameFont"`
	DescrFont  *config.FontSelect       `yaml:"descrFont"`
	Background *config.BackgroundSelect `yaml:"background"`
}

type heroRequest struct {
	index   int32
	visible bool
}

type globalRequest struct {
	visible bool
}

type fullRequest struct {
	global bool
	heroes []heroData
}

// HeroList is a module for displaying a list of heroes.
type HeroList struct {
	config                      *mConfig
	heroes                      []displayedHero
	curGlobalVisible            bool
	curHero                     int32
	curXOffset, curYOffset      int32
	contentWidth, contentHeight int32
	mask                        *sdl.Texture
	status                      heroStatus
	alphaMod                    uint8
}

const duration = time.Second / 2

func newRenderer(r render.Renderer,
	ms server.MessageSender) (modules.Renderer, error) {
	frame := r.OutputSize()
	return &HeroList{curGlobalVisible: false, curXOffset: 0,
		curYOffset: 0, contentWidth: frame.Width / 4,
		contentHeight: frame.Height / 10, status: resting}, nil
}

// Descriptor describes the HeroList module.
var Descriptor = modules.Module{
	Name:                "Hero List",
	ID:                  "herolist",
	ResourceCollections: nil,
	EndpointPaths:       []string{"", "/"},
	DefaultConfig: &mConfig{NameFont: config.NewFontSelect(0, api.ContentFont,
		api.RegularFont, api.RGBA{R: 0, G: 0, B: 0, A: 255}),
		DescrFont: config.NewFontSelect(0, api.ContentFont, api.RegularFont,
			api.RGBA{R: 0, G: 0, B: 0, A: 255}),
		Background: config.NewBackgroundSelect(
			api.RGBA{R: 255, G: 255, B: 255, A: 255}.AsBackground())},
	CreateRenderer: newRenderer, CreateState: newState,
}

func (l *HeroList) boxWidth(borderWidth int32) int32 {
	return l.contentWidth + 5*borderWidth
}

func (l *HeroList) boxHeight(borderWidth int32) int32 {
	return l.contentHeight + 4*borderWidth
}

func (l *HeroList) buildHeroBox(r render.Renderer, h heroData) render.Image {
	unit := r.Unit()
	canvas, frame := r.CreateCanvas(l.boxWidth(unit)-unit,
		l.boxHeight(unit)-2*unit, l.config.Background.Background,
		render.North|render.East|render.South)
	_, frame = frame.Carve(render.West, 2*unit)
	nameImg := r.RenderText(h.name, l.config.NameFont.Font)
	defer r.FreeImage(&nameImg)
	descrImg := r.RenderText(h.desc, l.config.DescrFont.Font)
	defer r.FreeImage(&descrImg)
	nameFrame := frame.Position(nameImg.Width, nameImg.Height, render.Left,
		render.Top)
	nameImg.Draw(r, nameFrame, 255)
	descrFrame := frame.Position(descrImg.Width, descrImg.Height, render.Left,
		render.Bottom)
	descrImg.Draw(r, descrFrame, 255)
	return canvas.Finish()
}

// InitTransition starts a transition
func (l *HeroList) InitTransition(
	r render.Renderer, data interface{}) time.Duration {
	switch req := data.(type) {
	case *globalRequest:
		if req.visible != l.curGlobalVisible {
			if req.visible {
				l.status = showingAll
			} else {
				l.status = hidingAll
			}
			return duration
		}
	case *heroRequest:
		if l.heroes[req.index].visible != req.visible {
			if req.visible {
				l.status = showingHero
			} else {
				l.status = hidingHero
			}
			l.curHero = req.index
			return duration
		}
	default:
		panic("HeroList.InitTransition called with unexpected data type")
	}
	return -1
}

// TransitionStep advances the transition
func (l *HeroList) TransitionStep(r render.Renderer, elapsed time.Duration) {
	pos := render.TransitionCurve{Duration: duration}.Cubic(elapsed)
	unit := r.Unit()
	switch l.status {
	case showingAll:
		l.curXOffset = int32((1.0 - pos) * float32(l.boxWidth(unit)))
	case hidingAll:
		l.curXOffset = int32(pos * float32(l.boxWidth(unit)))
	case showingHero:
		l.curXOffset = int32((1.0 - pos) * float32((l.boxWidth(unit))))
		l.curYOffset = int32(pos * float32(l.boxHeight(unit)+l.contentHeight/4))
		l.alphaMod = uint8(pos * 255)
	case hidingHero:
		l.curXOffset = int32(pos * float32(l.boxWidth(unit)))
		l.curYOffset = int32((1.0 - pos) * float32(l.boxHeight(unit)+l.contentHeight/4))
		l.alphaMod = uint8((1.0 - pos) * 255)
	}
}

// FinishTransition finalizes the transition
func (l *HeroList) FinishTransition(r render.Renderer) {
	l.curXOffset = 0
	l.curYOffset = 0
	switch l.status {
	case showingHero, hidingHero:
		l.heroes[l.curHero].visible = l.status == showingHero
	case hidingAll:
		l.curGlobalVisible = false
	case showingAll:
		l.curGlobalVisible = true
	}
	l.status = resting
}

// Render renders the current state of the HeroList
func (l *HeroList) Render(r render.Renderer) {
	if !l.curGlobalVisible && l.status == resting {
		return
	}
	unit := r.Unit()

	frame := r.OutputSize()
	_, frame = frame.Carve(render.North, frame.Height/7)
	frame, _ = frame.Carve(render.West, l.boxWidth(unit))

	for i := range l.heroes {
		if !l.heroes[i].visible && (l.curHero != int32(i) ||
			(l.status != showingHero && l.status != hidingHero)) {
			continue
		}
		var targetRect render.Rectangle
		targetRect, frame = frame.Carve(render.North, l.boxHeight(unit))

		if l.status == showingAll || l.status == hidingAll ||
			((l.status == showingHero || l.status == hidingHero) && l.curHero == int32(i)) {
			targetRect.X -= l.curXOffset
			_, frame = frame.Carve(render.North, l.curYOffset-l.boxHeight(unit))
		} else {
			_, frame = frame.Carve(render.North, unit*4)
		}

		if targetRect.X == 0 {
			l.heroes[i].box.Draw(r, targetRect, 255)
		} else {
			l.heroes[i].box.Draw(r, targetRect, l.alphaMod)
		}
	}
}

// Rebuild receives state data and config and immediately updates everything.
func (l *HeroList) Rebuild(r render.Renderer, data interface{}, configVal interface{}) {
	l.config = configVal.(*mConfig)
	for i := range l.heroes {
		r.FreeImage(&l.heroes[i].box)
	}
	if data != nil {
		req := data.(*fullRequest)
		if req.heroes == nil {
			l.heroes = nil
		} else {
			l.heroes = make([]displayedHero, len(req.heroes))
			for i := range req.heroes {
				l.heroes[i] = displayedHero{
					heroData: req.heroes[i]}
			}
		}
		l.curGlobalVisible = req.global
	}
	for i := range l.heroes {
		l.heroes[i].box = l.buildHeroBox(r, l.heroes[i].heroData)
	}

	l.status = resting
}
