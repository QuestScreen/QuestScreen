package herolist

import (
	"log"
	"sync"
	"time"

	"github.com/flyx/pnpscreen/data"
	"github.com/flyx/pnpscreen/display"
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
	store                       *data.Store
	display                     *display.Display
	heroes                      []displayedHero
	curGlobalVisible            bool
	curHero                     int32
	curXOffset, curYOffset      int32
	contentWidth, contentHeight int32
	borderWidth                 int32
	status                      heroStatus
}

// Init initializes the module.
func (l *HeroList) Init(display *display.Display, store *data.Store) error {
	l.state.owner = l
	l.store = store
	l.display = display
	l.curGlobalVisible = false
	l.curXOffset = 0
	l.curYOffset = 0
	winWidth, winHeight, _ := display.Renderer.GetOutputSize()
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

// InternalName returns "herolist"
func (*HeroList) InternalName() string {
	return "herolist"
}

func (l *HeroList) boxWidth() int32 {
	return l.contentWidth + 5*l.borderWidth
}

func (l *HeroList) boxHeight() int32 {
	return l.contentHeight + 4*l.borderWidth
}

func renderText(text string, display *display.Display, fontIndex int32,
	style data.FontStyle, r uint8, g uint8, b uint8) *sdl.Texture {
	fontDef := data.SelectableFont{Size: data.ContentFont, Style: style,
		FamilyIndex: fontIndex, Family: display.Fonts[fontIndex].Name}
	face := display.GetFontFace(&fontDef)

	surface, err := face.RenderUTF8Blended(
		text, sdl.Color{R: r, G: g, B: b, A: 255})
	if err != nil {
		log.Println(err)
		return nil
	}
	textTexture, err := display.Renderer.CreateTextureFromSurface(surface)
	surface.Free()
	if err != nil {
		log.Println(err)
		return nil
	}
	return textTexture
}

func (l *HeroList) rebuildHeroBoxes() {
	if l.heroes != nil {
		for _, hero := range l.heroes {
			hero.tex.Destroy()
		}
	}
	if l.store.GetActiveGroup() == -1 {
		l.heroes = nil
	} else {
		l.heroes = make([]displayedHero, l.store.Config.NumHeroes(l.store.GetActiveGroup()))
		var err error
		for index := range l.heroes {
			hero := &l.heroes[index]
			hero.shown = true
			hero.tex, err = l.display.Renderer.CreateTexture(sdl.PIXELFORMAT_RGB888, sdl.TEXTUREACCESS_TARGET,
				l.boxWidth(), l.boxHeight())
			if err == nil {
				l.display.Renderer.SetRenderTarget(hero.tex)
				name := renderText(l.store.Config.HeroName(l.store.GetActiveGroup(), index),
					l.display, 0, data.Standard, 0, 0, 0)
				_, _, nameWidth, nameHeight, _ := name.Query()
				name.SetBlendMode(sdl.BLENDMODE_BLEND)
				descr := renderText(l.store.Config.HeroDescription(l.store.GetActiveGroup(), index),
					l.display, 0, data.Standard, 50, 50, 50)
				_, _, descrWidth, descrHeight, _ := descr.Query()
				descr.SetBlendMode(sdl.BLENDMODE_BLEND)
				l.display.Renderer.Clear()
				l.display.Renderer.SetDrawColor(0, 0, 0, 192)
				l.display.Renderer.FillRect(&sdl.Rect{X: 0, Y: 0, W: l.boxWidth(), H: l.boxHeight()})
				l.display.Renderer.SetDrawColor(200, 173, 127, 255)
				l.display.Renderer.FillRect(&sdl.Rect{X: 0, Y: l.borderWidth, W: int32(l.contentWidth + 4*l.borderWidth),
					H: int32(l.contentHeight + 2*l.borderWidth)})
				l.display.Renderer.Copy(name, nil, &sdl.Rect{X: 2 * l.borderWidth, Y: l.borderWidth, W: nameWidth, H: nameHeight})

				l.display.Renderer.Copy(descr, nil, &sdl.Rect{X: 2 * l.borderWidth,
					Y: l.boxHeight() - 2*l.borderWidth - descrHeight, W: descrWidth, H: descrHeight})
				name.Destroy()
				descr.Destroy()
			} else {
				log.Println(err)
			}
		}
		l.display.Renderer.SetRenderTarget(nil)
	}
}

// InitTransition starts a transition
func (l *HeroList) InitTransition() time.Duration {
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
func (l *HeroList) TransitionStep(elapsed time.Duration) {
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
func (l *HeroList) FinishTransition() {
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
func (l *HeroList) Render() {
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
		_, winHeight, _ := l.display.Renderer.GetOutputSize()
		if xOffset == 0 {
			targetRect := sdl.Rect{X: 0, Y: winHeight/10 + (l.boxHeight()+l.contentHeight/4)*shown + additionalYOffset,
				W: l.boxWidth(), H: l.boxHeight()}
			if err := l.display.Renderer.Copy(l.heroes[i].tex, nil, &targetRect); err != nil {
				log.Println(err)
			}
		} else {
			targetRect := sdl.Rect{X: 0, Y: winHeight/10 + (l.boxHeight()+l.contentHeight/4)*shown + additionalYOffset,
				W: l.boxWidth() - xOffset, H: l.boxHeight()}
			sourceRect := sdl.Rect{X: l.curXOffset, Y: 0, W: l.boxWidth(), H: l.boxHeight()}
			if err := l.display.Renderer.Copy(l.heroes[i].tex, &sourceRect, &targetRect); err != nil {
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
func (l *HeroList) SetConfig(value interface{}) bool {
	l.config = value.(*config)
	return true
}

// GetConfig retrieves the current configuration of the item.
func (l *HeroList) GetConfig() interface{} {
	return l.config
}

// GetState returns the current state.
func (l *HeroList) GetState() data.ModuleState {
	return &l.state
}

// RebuildState queries the new state through the channel and immediately
// updates everything.
func (l *HeroList) RebuildState() {
	l.rebuildHeroBoxes()
	l.requests.mutex.Lock()
	defer l.requests.mutex.Unlock()
	if l.requests.kind != stateRequest {
		panic("got something else than stateRequest on RebuildState")
	}
	for i := range l.requests.heroes {
		l.heroes[i].shown = l.requests.heroes[i]
	}
	l.curGlobalVisible = l.requests.globalVisible
	l.status = resting
	l.requests.kind = noRequest
}
