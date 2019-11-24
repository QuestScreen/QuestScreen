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

type config struct{}

type requestKind int

const (
	noRequest requestKind = iota
	globalRequest
	heroRequest
	stateRequest
)

type requests struct {
	mutex         sync.Mutex
	kind          requestKind
	globalVisible bool
	heroIndex     int32
	heroVisible   bool
	heroes        []bool
}

// HeroList is a module for displaying a list of heroes.
type HeroList struct {
	*config
	state
	requests
	env                         api.Environment
	index                       api.ModuleIndex
	heroes                      []displayedHero
	curGlobalVisible            bool
	curHero                     int32
	curXOffset, curYOffset      int32
	contentWidth, contentHeight int32
	borderWidth                 int32
	status                      heroStatus
}

// Init initializes the module.
func (l *HeroList) Init(renderer *sdl.Renderer, env api.Environment, index api.ModuleIndex) error {
	l.state.owner = l
	l.env = env
	l.index = index
	l.curGlobalVisible = false
	l.curXOffset = 0
	l.curYOffset = 0
	winWidth, winHeight, _ := renderer.GetOutputSize()
	l.contentWidth = winWidth / 4
	l.contentHeight = winHeight / 10
	l.borderWidth = winHeight / 133
	l.status = resting
	return nil
}

// Name returns "Hero List".
func (*HeroList) Name() string {
	return "Hero List"
}

// ID returns "herolist"
func (*HeroList) ID() string {
	return "herolist"
}

func (l *HeroList) boxWidth() int32 {
	return l.contentWidth + 5*l.borderWidth
}

func (l *HeroList) boxHeight() int32 {
	return l.contentHeight + 4*l.borderWidth
}

func (l *HeroList) renderText(
	text string, renderer *sdl.Renderer, fontFamilyIndex int,
	style api.FontStyle, r uint8, g uint8, b uint8) *sdl.Texture {
	face := l.env.Font(fontFamilyIndex, style, api.ContentFont)

	surface, err := face.RenderUTF8Blended(
		text, sdl.Color{R: r, G: g, B: b, A: 255})
	if err != nil {
		log.Println(err)
		return nil
	}
	textTexture, err := renderer.CreateTextureFromSurface(surface)
	surface.Free()
	if err != nil {
		log.Println(err)
		return nil
	}
	return textTexture
}

func (l *HeroList) rebuildHeroBoxes(renderer *sdl.Renderer) {
	if l.heroes != nil {
		for _, hero := range l.heroes {
			hero.tex.Destroy()
		}
	}
	heroes := l.env.Heroes()

	if heroes.Length() == 0 {
		l.heroes = nil
	} else {
		l.heroes = make([]displayedHero, heroes.Length())
		var err error
		for index := range l.heroes {
			hero := heroes.Item(index)
			heroBox := &l.heroes[index]
			heroBox.shown = true
			heroBox.tex, err = renderer.CreateTexture(sdl.PIXELFORMAT_RGB888,
				sdl.TEXTUREACCESS_TARGET,
				l.boxWidth(), l.boxHeight())
			if err == nil {
				renderer.SetRenderTarget(heroBox.tex)
				name := l.renderText(hero.Name(),
					renderer, 0, api.Standard, 0, 0, 0)
				_, _, nameWidth, nameHeight, _ := name.Query()
				name.SetBlendMode(sdl.BLENDMODE_BLEND)
				descr := l.renderText(hero.Description(),
					renderer, 0, api.Standard, 50, 50, 50)
				_, _, descrWidth, descrHeight, _ := descr.Query()
				descr.SetBlendMode(sdl.BLENDMODE_BLEND)
				renderer.Clear()
				renderer.SetDrawColor(0, 0, 0, 192)
				renderer.FillRect(&sdl.Rect{X: 0, Y: 0, W: l.boxWidth(), H: l.boxHeight()})
				renderer.SetDrawColor(200, 173, 127, 255)
				renderer.FillRect(&sdl.Rect{X: 0, Y: l.borderWidth, W: int32(l.contentWidth + 4*l.borderWidth),
					H: int32(l.contentHeight + 2*l.borderWidth)})
				renderer.Copy(name, nil, &sdl.Rect{X: 2 * l.borderWidth, Y: l.borderWidth, W: nameWidth, H: nameHeight})

				renderer.Copy(descr, nil, &sdl.Rect{X: 2 * l.borderWidth,
					Y: l.boxHeight() - 2*l.borderWidth - descrHeight, W: descrWidth, H: descrHeight})
				name.Destroy()
				descr.Destroy()
			} else {
				log.Println(err)
			}
		}
		renderer.SetRenderTarget(nil)
	}
}

// InitTransition starts a transition
func (l *HeroList) InitTransition(renderer *sdl.Renderer) time.Duration {
	l.requests.mutex.Lock()
	defer l.requests.mutex.Unlock()
	curRequest := l.requests.kind
	l.requests.kind = noRequest
	switch curRequest {
	case noRequest:
		return -1
	case globalRequest:
		if l.requests.globalVisible != l.curGlobalVisible {
			if l.requests.globalVisible {
				l.status = showingAll
			} else {
				l.status = hidingAll
			}
			return time.Second
		}
	case heroRequest:
		if l.heroes[l.requests.heroIndex].shown != l.requests.heroVisible {
			if l.requests.heroVisible {
				l.status = showingHero
			} else {
				l.status = hidingHero
			}
			l.heroes[l.requests.heroIndex].tex.SetBlendMode(sdl.BLENDMODE_BLEND)
			l.curHero = l.requests.heroIndex
			return time.Second
		}
	case stateRequest:
		return -1
	}

	return -1
}

// TransitionStep advances the transition
func (l *HeroList) TransitionStep(renderer *sdl.Renderer, elapsed time.Duration) {
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
func (l *HeroList) FinishTransition(renderer *sdl.Renderer) {
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
func (l *HeroList) Render(renderer *sdl.Renderer) {
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
		_, winHeight, _ := renderer.GetOutputSize()
		if xOffset == 0 {
			targetRect := sdl.Rect{X: 0, Y: winHeight/10 + (l.boxHeight()+l.contentHeight/4)*shown + additionalYOffset,
				W: l.boxWidth(), H: l.boxHeight()}
			if err := renderer.Copy(l.heroes[i].tex, nil, &targetRect); err != nil {
				log.Println(err)
			}
		} else {
			targetRect := sdl.Rect{X: 0, Y: winHeight/10 + (l.boxHeight()+l.contentHeight/4)*shown + additionalYOffset,
				W: l.boxWidth() - xOffset, H: l.boxHeight()}
			sourceRect := sdl.Rect{X: l.curXOffset, Y: 0, W: l.boxWidth(), H: l.boxHeight()}
			if err := renderer.Copy(l.heroes[i].tex, &sourceRect, &targetRect); err != nil {
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

// EmptyConfig returns an empty configuration
func (*HeroList) EmptyConfig() interface{} {
	return &config{}
}

// DefaultConfig returns the default configuration
func (*HeroList) DefaultConfig() interface{} {
	return &config{}
}

// SetConfig sets the module's configuration
func (l *HeroList) SetConfig(value interface{}) {
	l.config = value.(*config)
}

// State returns the current state.
func (l *HeroList) State() api.ModuleState {
	return &l.state
}

// RebuildState queries the new state through the channel and immediately
// updates everything.
func (l *HeroList) RebuildState(renderer *sdl.Renderer) {
	l.rebuildHeroBoxes(renderer)
	l.requests.mutex.Lock()
	defer l.requests.mutex.Unlock()
	switch l.requests.kind {
	case stateRequest:
		for i := range l.requests.heroes {
			l.heroes[i].shown = l.requests.heroes[i]
		}
		l.curGlobalVisible = l.requests.globalVisible
	case noRequest:
		break
	default:
		panic("got something else than stateRequest on RebuildState")
	}
	l.status = resting
	l.requests.kind = noRequest
}

// ResourceCollections returns nil.
func (l *HeroList) ResourceCollections() []api.ResourceSelector {
	return nil
}
