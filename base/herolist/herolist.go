package herolist

import (
	"log"
	"sync"
	"time"

	"github.com/flyx/pnpscreen/api"

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
	Font *api.SelectableFont `config:"Font" yaml:"Font"`
}

type requestKind int

const (
	noRequest requestKind = iota
	globalRequest
	heroRequest
	stateRequest
)

type sharedData struct {
	moduleIndex   api.ModuleIndex // read-only, therefore not mutex-protected
	mutex         sync.Mutex      // protects values below
	kind          requestKind
	globalVisible bool
	heroIndex     int32
	heroVisible   bool
	heroes        []bool
}

// HeroList is a module for displaying a list of heroes.
type HeroList struct {
	*config
	sharedData
	index                       api.ModuleIndex
	heroes                      []displayedHero
	curGlobalVisible            bool
	curHero                     int32
	curXOffset, curYOffset      int32
	contentWidth, contentHeight int32
	borderWidth                 int32
	status                      heroStatus
}

// CreateModule creates the HeroList module.
func CreateModule(renderer *sdl.Renderer,
	env api.StaticEnvironment, index api.ModuleIndex) (api.Module, error) {
	l := new(HeroList)
	l.index = index
	l.curGlobalVisible = false
	l.curXOffset = 0
	l.curYOffset = 0
	winWidth, winHeight, _ := renderer.GetOutputSize()
	l.contentWidth = winWidth / 4
	l.contentHeight = winHeight / 10
	l.borderWidth = winHeight / 133
	l.status = resting
	return l, nil
}

// Descriptor describes the HeroList module.
var Descriptor = api.ModuleDescriptor{
	Name:                "Hero List",
	ID:                  "herolist",
	ResourceCollections: nil,
	Actions:             []string{"switchGlobal", "switchHero"},
	DefaultConfig: &config{Font: &api.SelectableFont{
		FamilyIndex: 0, Size: api.ContentFont, Style: api.Standard}},
	CreateModule: CreateModule,
}

// Descriptor returns the descriptor of the HeroList
func (*HeroList) Descriptor() *api.ModuleDescriptor {
	return &Descriptor
}

func (l *HeroList) boxWidth() int32 {
	return l.contentWidth + 5*l.borderWidth
}

func (l *HeroList) boxHeight() int32 {
	return l.contentHeight + 4*l.borderWidth
}

func (l *HeroList) renderText(
	text string, ctx api.RenderContext, r uint8, g uint8, b uint8) *sdl.Texture {
	face := ctx.Env.Font(
		l.config.Font.FamilyIndex, l.config.Font.Style, l.config.Font.Size)

	surface, err := face.RenderUTF8Blended(
		text, sdl.Color{R: r, G: g, B: b, A: 255})
	if err != nil {
		log.Println(err)
		return nil
	}
	textTexture, err := ctx.Renderer.CreateTextureFromSurface(surface)
	surface.Free()
	if err != nil {
		log.Println(err)
		return nil
	}
	return textTexture
}

func (l *HeroList) rebuildHeroBoxes(ctx api.RenderContext) {
	if l.heroes != nil {
		for _, hero := range l.heroes {
			hero.tex.Destroy()
		}
	}
	heroes := ctx.Env.Heroes()
	defer heroes.Close()

	if heroes.NumHeroes() == 0 {
		l.heroes = nil
	} else {
		l.heroes = make([]displayedHero, heroes.NumHeroes())
		var err error
		for index := range l.heroes {
			hero := heroes.Hero(index)
			heroBox := &l.heroes[index]
			heroBox.shown = true
			heroBox.tex, err = ctx.Renderer.CreateTexture(sdl.PIXELFORMAT_RGB888,
				sdl.TEXTUREACCESS_TARGET,
				l.boxWidth(), l.boxHeight())
			if err == nil {
				ctx.Renderer.SetRenderTarget(heroBox.tex)
				name := l.renderText(hero.Name(),
					ctx, 0, 0, 0)
				_, _, nameWidth, nameHeight, _ := name.Query()
				name.SetBlendMode(sdl.BLENDMODE_BLEND)
				descr := l.renderText(hero.Description(), ctx, 50, 50, 50)
				_, _, descrWidth, descrHeight, _ := descr.Query()
				descr.SetBlendMode(sdl.BLENDMODE_BLEND)
				ctx.Renderer.Clear()
				ctx.Renderer.SetDrawColor(0, 0, 0, 192)
				ctx.Renderer.FillRect(&sdl.Rect{X: 0, Y: 0, W: l.boxWidth(), H: l.boxHeight()})
				ctx.Renderer.SetDrawColor(200, 173, 127, 255)
				ctx.Renderer.FillRect(&sdl.Rect{X: 0, Y: l.borderWidth, W: int32(l.contentWidth + 4*l.borderWidth),
					H: int32(l.contentHeight + 2*l.borderWidth)})
				ctx.Renderer.Copy(name, nil, &sdl.Rect{X: 2 * l.borderWidth, Y: l.borderWidth, W: nameWidth, H: nameHeight})

				ctx.Renderer.Copy(descr, nil, &sdl.Rect{X: 2 * l.borderWidth,
					Y: l.boxHeight() - 2*l.borderWidth - descrHeight, W: descrWidth, H: descrHeight})
				name.Destroy()
				descr.Destroy()
			} else {
				log.Println(err)
			}
		}
		ctx.Renderer.SetRenderTarget(nil)
	}
}

// InitTransition starts a transition
func (l *HeroList) InitTransition(ctx api.RenderContext) time.Duration {
	l.sharedData.mutex.Lock()
	defer l.sharedData.mutex.Unlock()
	curRequest := l.sharedData.kind
	l.sharedData.kind = noRequest
	switch curRequest {
	case noRequest:
		return -1
	case globalRequest:
		if l.sharedData.globalVisible != l.curGlobalVisible {
			if l.sharedData.globalVisible {
				l.status = showingAll
			} else {
				l.status = hidingAll
			}
			return time.Second
		}
	case heroRequest:
		if l.heroes[l.sharedData.heroIndex].shown != l.sharedData.heroVisible {
			if l.sharedData.heroVisible {
				l.status = showingHero
			} else {
				l.status = hidingHero
			}
			l.heroes[l.sharedData.heroIndex].tex.SetBlendMode(sdl.BLENDMODE_BLEND)
			l.curHero = l.sharedData.heroIndex
			return time.Second
		}
	case stateRequest:
		return -1
	}

	return -1
}

// TransitionStep advances the transition
func (l *HeroList) TransitionStep(ctx api.RenderContext, elapsed time.Duration) {
	switch l.status {
	case showingAll:
		l.curXOffset = int32(((time.Second - elapsed) * time.Duration(l.boxWidth())) / time.Second)
	case hidingAll:
		l.curXOffset = int32((elapsed * time.Duration(l.boxWidth())) / time.Second)
	case showingHero:
		l.curXOffset = int32(((time.Second - elapsed) * time.Duration(l.boxWidth())) / time.Second)
		l.curYOffset = int32((elapsed * time.Duration(l.boxHeight()+l.contentHeight/4)) / time.Second)
		l.heroes[l.curHero].tex.SetAlphaMod(uint8((elapsed * 255) / time.Second))
	case hidingHero:
		l.curXOffset = int32((elapsed * time.Duration(l.boxWidth())) / time.Second)
		l.curYOffset = int32(((time.Second - elapsed) * time.Duration(l.boxHeight()+l.contentHeight/4)) / time.Second)
		l.heroes[l.curHero].tex.SetAlphaMod(uint8(((time.Second - elapsed) * 255) / time.Second))
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
	for i := range l.heroes {
		if !l.heroes[i].shown && (l.curHero != int32(i) || (l.status != showingHero && l.status != hidingHero)) {
			continue
		}
		xOffset := int32(0)
		if l.status == showingAll || l.status == hidingAll ||
			((l.status == showingHero || l.status == hidingHero) && l.curHero == int32(i)) {
			xOffset = l.curXOffset
		}
		_, winHeight, _ := ctx.Renderer.GetOutputSize()
		if xOffset == 0 {
			targetRect := sdl.Rect{X: 0, Y: winHeight/10 + (l.boxHeight()+l.contentHeight/4)*shown + additionalYOffset,
				W: l.boxWidth(), H: l.boxHeight()}
			if err := ctx.Renderer.Copy(l.heroes[i].tex, nil, &targetRect); err != nil {
				log.Println(err)
			}
		} else {
			targetRect := sdl.Rect{X: 0, Y: winHeight/10 + (l.boxHeight()+l.contentHeight/4)*shown + additionalYOffset,
				W: l.boxWidth() - xOffset, H: l.boxHeight()}
			sourceRect := sdl.Rect{X: l.curXOffset, Y: 0, W: l.boxWidth(), H: l.boxHeight()}
			if err := ctx.Renderer.Copy(l.heroes[i].tex, &sourceRect, &targetRect); err != nil {
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

// SetConfig sets the module's configuration
func (l *HeroList) SetConfig(value interface{}) {
	l.config = value.(*config)
}

// CreateState creates a new state for the HeroList
func (l *HeroList) CreateState(yamlSubtree interface{}, env api.Environment) (api.ModuleState, error) {
	return newState(yamlSubtree, env, &l.sharedData)
}

// RebuildState queries the new state through the channel and immediately
// updates everything.
func (l *HeroList) RebuildState(ctx api.RenderContext) {
	old := l.heroes
	l.rebuildHeroBoxes(ctx)
	l.sharedData.mutex.Lock()
	defer l.sharedData.mutex.Unlock()
	switch l.sharedData.kind {
	case stateRequest:
		for i := range l.sharedData.heroes {
			l.heroes[i].shown = l.sharedData.heroes[i]
		}
		l.curGlobalVisible = l.sharedData.globalVisible
	case noRequest:
		for i := range l.heroes {
			l.heroes[i].shown = old[i].shown
		}
	default:
		panic("got something else than stateRequest on RebuildState")
	}

	l.status = resting
	l.sharedData.kind = noRequest
}
