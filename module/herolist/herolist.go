package herolist

import (
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/flyx/rpscreen/data"
	"github.com/flyx/rpscreen/module"
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

type heroListConfig struct{}

// HeroList is a module for displaying a list of heroes.
type HeroList struct {
	common                      *module.SceneCommon
	config                      *heroListConfig
	heroes                      []displayedHero
	displayedGroup              int
	hidden                      bool
	reqSwitch                   int32
	reqHidden                   bool
	curXOffset, curYOffset      int32
	contentWidth, contentHeight int32
	borderWidth                 int32
	status                      heroStatus
}

// Init initializes the module.
func (l *HeroList) Init(common *module.SceneCommon) error {
	l.common = common
	l.displayedGroup = -1
	l.hidden = false
	l.reqHidden = false
	l.reqSwitch = -1
	l.curXOffset = 0
	l.curYOffset = 0
	winWidth, winHeight := common.Window.GetSize()
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

// UI constructs the HTML UI of the modules
func (l *HeroList) UI() template.HTML {
	var builder module.UIBuilder

	builder.StartForm(l, "switchHidden", "", true)
	if l.reqHidden {
		builder.SubmitButton("Show", "Complete List", true)
	} else {
		builder.SubmitButton("Hide", "Complete List", true)
	}
	builder.EndForm()

	if l.displayedGroup >= 0 {
		for i := 0; i < l.common.Config.NumHeroes(l.displayedGroup); i++ {
			builder.StartForm(l, "switchHero", "", true)
			builder.HiddenValue("index", strconv.Itoa(i))
			shown := l.heroes[i].shown
			if shown {
				builder.SubmitButton("Hide",
					l.common.Config.HeroName(l.displayedGroup, i), !l.reqHidden)
			} else {
				builder.SecondarySubmitButton("Show",
					l.common.Config.HeroName(l.displayedGroup, i), !l.reqHidden)
			}
			builder.EndForm()
		}
	}
	return builder.Finish()
}

// EndpointHandler implement the module's endpoint handler.
func (l *HeroList) EndpointHandler(suffix string, values url.Values, w http.ResponseWriter, returnPartial bool) bool {
	if suffix == "switchHidden" {
		l.reqHidden = !l.hidden
		var returns module.EndpointReturn
		if returnPartial {
			returns = module.EndpointReturnEmpty
		} else {
			returns = module.EndpointReturnRedirect
		}
		module.WriteEndpointHeader(w, returns)
		return true
	} else if suffix == "switchHero" {
		index, err := strconv.Atoi(values["index"][0])
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return false
		} else if index < 0 || index >= len(l.heroes) {
			http.Error(w, "index out of range", http.StatusBadRequest)
			return false
		}
		l.reqSwitch = int32(index)
		l.heroes[l.reqSwitch].shown = !l.heroes[l.reqSwitch].shown

		var returns module.EndpointReturn
		if returnPartial {
			returns = module.EndpointReturnEmpty
		} else {
			returns = module.EndpointReturnRedirect
		}
		module.WriteEndpointHeader(w, returns)
		return true
	} else {
		http.Error(w, "404 not found: "+suffix, http.StatusNotFound)
		return false
	}
}

func renderText(text string, common *module.SceneCommon, fontIndex int32,
	style data.FontStyle, r uint8, g uint8, b uint8) *sdl.Texture {
	face := common.Fonts[fontIndex].GetSize(common.DefaultBodyTextSize).GetFace(style)

	surface, err := face.RenderUTF8Blended(
		text, sdl.Color{R: r, G: g, B: b, A: 255})
	if err != nil {
		log.Println(err)
		return nil
	}
	textTexture, err := common.Renderer.CreateTextureFromSurface(surface)
	surface.Free()
	if err != nil {
		log.Println(err)
		return nil
	}
	return textTexture
}

func (l *HeroList) rebuildHeroBoxes() {
	if l.displayedGroup != -1 {
		for _, hero := range l.heroes {
			hero.tex.Destroy()
		}
	}
	if l.common.ActiveGroup == -1 {
		l.heroes = nil
	} else {
		l.heroes = make([]displayedHero, l.common.Config.NumHeroes(l.common.ActiveGroup))
		var err error
		for index := range l.heroes {
			hero := &l.heroes[index]
			hero.shown = true
			hero.tex, err = l.common.Renderer.CreateTexture(sdl.PIXELFORMAT_RGB888, sdl.TEXTUREACCESS_TARGET,
				l.boxWidth(), l.boxHeight())
			if err == nil {
				l.common.Renderer.SetRenderTarget(hero.tex)
				name := renderText(l.common.Config.HeroName(l.common.ActiveGroup, index),
					l.common, 0, data.Standard, 0, 0, 0)
				_, _, nameWidth, nameHeight, _ := name.Query()
				name.SetBlendMode(sdl.BLENDMODE_BLEND)
				descr := renderText(l.common.Config.HeroDescription(l.common.ActiveGroup, index),
					l.common, 0, data.Standard, 50, 50, 50)
				_, _, descrWidth, descrHeight, _ := descr.Query()
				descr.SetBlendMode(sdl.BLENDMODE_BLEND)
				l.common.Renderer.Clear()
				l.common.Renderer.SetDrawColor(0, 0, 0, 192)
				l.common.Renderer.FillRect(&sdl.Rect{X: 0, Y: 0, W: l.boxWidth(), H: l.boxHeight()})
				l.common.Renderer.SetDrawColor(200, 173, 127, 255)
				l.common.Renderer.FillRect(&sdl.Rect{X: 0, Y: l.borderWidth, W: int32(l.contentWidth + 4*l.borderWidth),
					H: int32(l.contentHeight + 2*l.borderWidth)})
				l.common.Renderer.Copy(name, nil, &sdl.Rect{X: 2 * l.borderWidth, Y: l.borderWidth, W: nameWidth, H: nameHeight})

				l.common.Renderer.Copy(descr, nil, &sdl.Rect{X: 2 * l.borderWidth,
					Y: l.boxHeight() - 2*l.borderWidth - descrHeight, W: descrWidth, H: descrHeight})
				name.Destroy()
				descr.Destroy()
			} else {
				log.Println(err)
			}
		}
		l.common.Renderer.SetRenderTarget(nil)
	}
}

// InitTransition starts a transition
func (l *HeroList) InitTransition() time.Duration {
	if l.common.ActiveGroup != l.displayedGroup {
		if l.displayedGroup == -1 {
			l.rebuildHeroBoxes()
			l.status = showingAll
			return time.Second
		} else if l.common.ActiveGroup == -1 {
			l.status = hidingAll
			return time.Second
		} else {
			l.status = hidingAll
			return time.Second * 2
		}
	} else if l.reqHidden != l.hidden {
		if l.hidden {
			l.status = showingAll
		} else {
			l.status = hidingAll
		}
		return time.Second
	}

	if l.heroes[l.reqSwitch].shown {
		l.status = hidingHero
	} else {
		l.status = showingHero
	}
	l.heroes[l.reqSwitch].tex.SetBlendMode(sdl.BLENDMODE_BLEND)
	return time.Second
}

// TransitionStep advances the transition
func (l *HeroList) TransitionStep(elapsed time.Duration) {
	if l.common.ActiveGroup != l.displayedGroup {
		if l.displayedGroup == -1 {
			l.curXOffset = int32(((time.Second - elapsed) * time.Duration(l.boxWidth())) / time.Second)
		} else if l.common.ActiveGroup == -1 {
			l.curXOffset = int32((elapsed * time.Duration(l.boxWidth())) / time.Second)
		} else {
			if elapsed >= time.Second {
				if l.status == hidingAll {
					l.rebuildHeroBoxes()
				}
				l.curXOffset = int32(((time.Second*2 - elapsed) * time.Duration(l.boxWidth())) / time.Second)
			} else {
				l.curXOffset = int32((elapsed * time.Duration(l.boxWidth())) / time.Second)
			}
		}
	} else if l.reqHidden != l.hidden {
		if l.status == showingAll {
			l.curXOffset = int32(((time.Second - elapsed) * time.Duration(l.boxWidth())) / time.Second)
		} else {
			l.curXOffset = int32((elapsed * time.Duration(l.boxWidth())) / time.Second)
		}
	} else {
		if l.status == showingHero {
			l.curXOffset = int32((elapsed * time.Duration(l.boxWidth())) / time.Second)
			l.curYOffset = int32(((time.Second - elapsed) * time.Duration(l.boxHeight()+l.contentHeight/4)) / time.Second)
			l.heroes[l.reqSwitch].tex.SetAlphaMod(uint8(((time.Second - elapsed) * 255) / time.Second))
		} else {
			l.curXOffset = int32(((time.Second - elapsed) * time.Duration(l.boxWidth())) / time.Second)
			l.curYOffset = int32((elapsed * time.Duration(l.boxHeight()+l.contentHeight/4)) / time.Second)
			l.heroes[l.reqSwitch].tex.SetAlphaMod(uint8((elapsed * 255) / time.Second))
		}
	}
}

// FinishTransition finalizes the transition
func (l *HeroList) FinishTransition() {
	l.curXOffset = 0
	l.curYOffset = 0
	l.status = resting
	if l.reqSwitch != -1 {
		l.heroes[l.reqSwitch].tex.SetAlphaMod(255)
		l.heroes[l.reqSwitch].tex.SetBlendMode(sdl.BLENDMODE_NONE)
	}
	l.reqSwitch = -1
	l.hidden = l.reqHidden
	if l.common.ActiveGroup == -1 {
		for i := range l.heroes {
			l.heroes[i].tex.Destroy()
		}
	}
	l.displayedGroup = l.common.ActiveGroup
}

// Render renders the current state of the HeroList
func (l *HeroList) Render() {
	shown := int32(0)
	additionalYOffset := int32(0)
	if l.hidden && l.status == resting {
		return
	}
	for i := range l.heroes {
		if l.reqSwitch != int32(i) && !l.heroes[i].shown {
			continue
		}
		xOffset := int32(0)
		if l.status == showingAll || l.status == hidingAll ||
			((l.status == showingHero || l.status == hidingHero) && l.reqSwitch == int32(i)) {
			xOffset = l.curXOffset
		}
		_, winHeight := l.common.Window.GetSize()
		if xOffset == 0 {
			targetRect := sdl.Rect{X: 0, Y: winHeight/10 + (l.boxHeight()+l.contentHeight/4)*shown + additionalYOffset,
				W: l.boxWidth(), H: l.boxHeight()}
			if err := l.common.Renderer.Copy(l.heroes[i].tex, nil, &targetRect); err != nil {
				log.Println(err)
			}
		} else {
			targetRect := sdl.Rect{X: 0, Y: winHeight/10 + (l.boxHeight()+l.contentHeight/4)*shown + additionalYOffset,
				W: l.boxWidth() - xOffset, H: l.boxHeight()}
			sourceRect := sdl.Rect{X: l.curXOffset, Y: 0, W: l.boxWidth(), H: l.boxHeight()}
			if err := l.common.Renderer.Copy(l.heroes[i].tex, &sourceRect, &targetRect); err != nil {
				log.Println(err)
			}
		}

		if (l.status == showingHero || l.status == hidingHero) && l.reqSwitch == int32(i) {
			additionalYOffset = l.curYOffset
		} else {
			shown++
		}
	}
}

// EmptyConfig returns an empty configuration
func (*HeroList) EmptyConfig() interface{} {
	return &heroListConfig{}
}

// DefaultConfig returns the default configuration
func (*HeroList) DefaultConfig() interface{} {
	return &heroListConfig{}
}

// SetConfig sets the module's configuration
func (l *HeroList) SetConfig(config interface{}) {
	l.config = config.(*heroListConfig)
}

// GetConfig retrieves the current configuration of the item.
func (l *HeroList) GetConfig() interface{} {
	return l.config
}

// NeedsTransition return true iff the group has been changed.
func (l *HeroList) NeedsTransition() bool {
	return l.displayedGroup != l.common.ActiveGroup
}
