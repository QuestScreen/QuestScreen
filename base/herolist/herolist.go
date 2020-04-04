package herolist

import (
	"log"
	"time"

	"github.com/QuestScreen/api"

	"github.com/veandco/go-sdl2/sdl"
)

type displayedHero struct {
	tex   *sdl.Texture
	shown bool
}

type heroStatus int

const (
	resting heroStatus = iota
	showingAll
	hidingAll
	showingHero
	hidingHero
)

type config struct {
	Font       *api.SelectableFont               `yaml:"font"`
	Background *api.SelectableTexturedBackground `yaml:"background"`
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
	heroes []bool
}

// HeroList is a module for displaying a list of heroes.
type HeroList struct {
	*config
	heroes                      []displayedHero
	curGlobalVisible            bool
	curHero                     int32
	curXOffset, curYOffset      int32
	contentWidth, contentHeight int32
	mask                        *sdl.Texture
	status                      heroStatus
}

const duration = time.Second / 2

func newRenderer(
	backend *sdl.Renderer, ms api.MessageSender) (api.ModuleRenderer, error) {
	winWidth, winHeight, _ := backend.GetOutputSize()
	return &HeroList{curGlobalVisible: false, curXOffset: 0,
		curYOffset: 0, contentWidth: winWidth / 4, contentHeight: winHeight / 10,
		status: resting}, nil
}

// Descriptor describes the HeroList module.
var Descriptor = api.Module{
	Name:                "Hero List",
	ID:                  "herolist",
	ResourceCollections: nil,
	EndpointPaths:       []string{"", "/"},
	DefaultConfig: &config{Font: &api.SelectableFont{
		FamilyIndex: 0, Size: api.ContentFont, Style: api.Standard},
		Background: &api.SelectableTexturedBackground{
			Primary:      api.RGBColor{Red: 255, Green: 255, Blue: 255},
			TextureIndex: -1,
		}},
	CreateRenderer: newRenderer, CreateState: newState,
}

// Descriptor returns the descriptor of the HeroList
func (*HeroList) Descriptor() *api.Module {
	return &Descriptor
}

func (l *HeroList) boxWidth(borderWidth int32) int32 {
	return l.contentWidth + 5*borderWidth
}

func (l *HeroList) boxHeight(borderWidth int32) int32 {
	return l.contentHeight + 4*borderWidth
}

func (l *HeroList) renderText(
	text string, ctx api.RenderContext, r uint8, g uint8, b uint8) *sdl.Texture {
	face := ctx.Font(
		l.config.Font.FamilyIndex, l.config.Font.Style, l.config.Font.Size)

	surface, err := face.RenderUTF8Blended(
		text, sdl.Color{R: r, G: g, B: b, A: 255})
	if err != nil {
		log.Println(err)
		return nil
	}
	textTexture, err := ctx.Renderer().CreateTextureFromSurface(surface)
	surface.Free()
	if err != nil {
		log.Println(err)
		return nil
	}
	return textTexture
}

func (l *HeroList) rebuildHeroBoxes(ctx api.ExtendedRenderContext) {
	if l.heroes != nil {
		for _, hero := range l.heroes {
			hero.tex.Destroy()
		}
	}

	heroes := ctx.Heroes()
	r := ctx.Renderer()

	if heroes.NumHeroes() == 0 {
		l.heroes = nil
	} else {
		l.heroes = make([]displayedHero, heroes.NumHeroes())
		unit := ctx.Unit()
		for index := range l.heroes {
			hero := heroes.Hero(index)
			heroBox := &l.heroes[index]
			heroBox.shown = true
			bgColor := l.config.Background.Primary.WithAlpha(255)
			canvas := ctx.CreateCanvas(l.boxWidth(unit)-unit,
				l.boxHeight(unit)-2*unit, &bgColor, l.mask,
				api.North|api.East|api.South)
			face := ctx.Font(
				l.config.Font.FamilyIndex, l.config.Font.Style, l.config.Font.Size)
			name := ctx.TextToTexture(hero.Name(), face,
				sdl.Color{R: 0, G: 0, B: 0, A: 255})
			_, _, nameWidth, nameHeight, _ := name.Query()
			r.Copy(name, nil, &sdl.Rect{
				X: 2 * unit, Y: unit, W: nameWidth, H: nameHeight})
			descr := ctx.TextToTexture(hero.Description(), face,
				sdl.Color{R: 50, G: 50, B: 50, A: 255})
			_, _, descrWidth, descrHeight, _ := descr.Query()
			r.Copy(descr, nil, &sdl.Rect{X: 2 * unit,
				Y: l.boxHeight(unit) - 2*unit - descrHeight,
				W: descrWidth, H: descrHeight})
			name.Destroy()
			descr.Destroy()
			heroBox.tex = canvas.Finish()
		}
	}
}

// InitTransition starts a transition
func (l *HeroList) InitTransition(
	ctx api.RenderContext, data interface{}) time.Duration {
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
		if l.heroes[req.index].shown != req.visible {
			if req.visible {
				l.status = showingHero
			} else {
				l.status = hidingHero
			}
			l.heroes[req.index].tex.SetBlendMode(sdl.BLENDMODE_BLEND)
			l.curHero = req.index
			return duration
		}
	default:
		panic("HeroList.InitTransition called with unexpected data type")
	}
	return -1
}

// TransitionStep advances the transition
func (l *HeroList) TransitionStep(ctx api.RenderContext, elapsed time.Duration) {
	pos := api.TransitionCurve{Duration: duration}.Cubic(elapsed)
	unit := ctx.Unit()
	switch l.status {
	case showingAll:
		l.curXOffset = int32((1.0 - pos) * float32(l.boxWidth(unit)))
	case hidingAll:
		l.curXOffset = int32(pos * float32(l.boxWidth(unit)))
	case showingHero:
		l.curXOffset = int32((1.0 - pos) * float32((l.boxWidth(unit))))
		l.curYOffset = int32(pos * float32(l.boxHeight(unit)+l.contentHeight/4))
		l.heroes[l.curHero].tex.SetAlphaMod(uint8(pos * 255))
	case hidingHero:
		l.curXOffset = int32(pos * float32(l.boxWidth(unit)))
		l.curYOffset = int32((1.0 - pos) * float32(l.boxHeight(unit)+l.contentHeight/4))
		l.heroes[l.curHero].tex.SetAlphaMod(uint8((1.0 - pos) * 255))
	}
}

// FinishTransition finalizes the transition
func (l *HeroList) FinishTransition(ctx api.RenderContext) {
	l.curXOffset = 0
	l.curYOffset = 0
	switch l.status {
	case showingHero, hidingHero:
		l.heroes[l.curHero].tex.SetAlphaMod(255)
		l.heroes[l.curHero].tex.SetBlendMode(sdl.BLENDMODE_NONE)
		l.heroes[l.curHero].shown = l.status == showingHero
	case hidingAll:
		l.curGlobalVisible = false
	case showingAll:
		l.curGlobalVisible = true
	}
	l.status = resting
}

// Render renders the current state of the HeroList
func (l *HeroList) Render(ctx api.RenderContext) {
	shown := int32(0)
	additionalYOffset := int32(0)
	if !l.curGlobalVisible && l.status == resting {
		return
	}
	unit := ctx.Unit()
	r := ctx.Renderer()
	for i := range l.heroes {
		if !l.heroes[i].shown && (l.curHero != int32(i) ||
			(l.status != showingHero && l.status != hidingHero)) {
			continue
		}
		xOffset := int32(0)
		if l.status == showingAll || l.status == hidingAll ||
			((l.status == showingHero || l.status == hidingHero) && l.curHero == int32(i)) {
			xOffset = l.curXOffset
		}
		_, winHeight, _ := r.GetOutputSize()
		if xOffset == 0 {
			targetRect := sdl.Rect{X: 0, Y: winHeight/10 +
				(l.boxHeight(unit)+l.contentHeight/4)*shown + additionalYOffset,
				W: l.boxWidth(unit), H: l.boxHeight(unit)}
			if err := r.Copy(l.heroes[i].tex, nil, &targetRect); err != nil {
				log.Println(err)
			}
		} else {
			targetRect := sdl.Rect{X: 0, Y: winHeight/10 +
				(l.boxHeight(unit)+l.contentHeight/4)*shown + additionalYOffset,
				W: l.boxWidth(unit) - xOffset, H: l.boxHeight(unit)}
			sourceRect := sdl.Rect{X: l.curXOffset, Y: 0, W: l.boxWidth(unit),
				H: l.boxHeight(unit)}
			if err := r.Copy(l.heroes[i].tex, &sourceRect, &targetRect); err != nil {
				log.Println(err)
			}
		}

		if (l.status == showingHero || l.status == hidingHero) && l.curHero == int32(i) {
			additionalYOffset = l.curYOffset
		} else {
			shown++
		}
	}
}

// Rebuild receives state data and config and immediately updates everything.
func (l *HeroList) Rebuild(
	ctx api.ExtendedRenderContext, data interface{}, configVal interface{}) {
	l.config = configVal.(*config)
	old := l.heroes
	ctx.UpdateMask(&l.mask, *l.config.Background)
	l.rebuildHeroBoxes(ctx)
	if data != nil {
		req := data.(*fullRequest)
		for i := range req.heroes {
			l.heroes[i].shown = req.heroes[i]
		}
		l.curGlobalVisible = req.global
	} else {
		for i := range l.heroes {
			l.heroes[i].shown = old[i].shown
		}
	}

	l.status = resting
}
