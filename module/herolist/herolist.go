package herolist

import (
	"github.com/flyx/rpscreen/module"
	"github.com/veandco/go-sdl2/sdl"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type DisplayedHero struct {
	tex   *sdl.Texture
	shown bool
}

type Status int

const (
	Resting Status = iota
	ShowingAll
	HidingAll
	ShowingHero
	HidingHero
)

type HeroList struct {
	heroes                      []DisplayedHero
	displayedGroup              int32
	reqGroup                    int32
	hidden                      bool
	reqSwitch                   int32
	reqHidden                   bool
	curXOffset, curYOffset      int32
	contentWidth, contentHeight int32
	borderWidth                 int32
	status                      Status
}

func (l *HeroList) Init(common *module.SceneCommon) error {
	l.displayedGroup = -1
	l.reqGroup = -1
	l.hidden = false
	l.reqHidden = false
	l.reqSwitch = -1
	l.curXOffset = 0
	l.curYOffset = 0
	winWidth, winHeight := common.Window.GetSize()
	l.contentWidth = winWidth / 4
	l.contentHeight = winHeight / 10
	l.borderWidth = winHeight / 133
	l.status = Resting
	return nil
}

func (*HeroList) Name() string {
	return "Hero List"
}

func (*HeroList) InternalName() string {
	return "herolist"
}

func (l *HeroList) boxWidth() int32 {
	return l.contentWidth + 5*l.borderWidth
}

func (l *HeroList) boxHeight() int32 {
	return l.contentHeight + 4*l.borderWidth
}

func (l *HeroList) UI(common *module.SceneCommon) template.HTML {
	var builder module.UIBuilder

	builder.StartForm(l, "switchHidden", "", true)
	if l.reqHidden {
		builder.SubmitButton("Show", "Complete List", true)
	} else {
		builder.SubmitButton("Hide", "Complete List", true)
	}
	builder.EndForm()

	if l.reqGroup >= 0 {
		for index, hero := range common.Groups[common.ActiveGroup].Heroes {
			builder.StartForm(l, "switchHero", "", true)
			builder.HiddenValue("index", strconv.Itoa(index))
			shown := true
			if l.reqGroup == common.ActiveGroup {
				shown = l.heroes[index].shown
			}
			if shown {
				builder.SubmitButton("Hide", hero.Name, !l.reqHidden)
			} else {
				builder.SecondarySubmitButton("Show", hero.Name, !l.reqHidden)
			}
			builder.EndForm()
		}
	}
	return builder.Finish()
}

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

func renderText(text string, common *module.SceneCommon, fontIndex int32) *sdl.Texture {
	surface, err := common.Fonts[fontIndex].Font.RenderUTF8Blended(
		text, sdl.Color{0, 0, 0, 230})
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

func (l *HeroList) rebuildHeroBoxes(common *module.SceneCommon) {
	if l.displayedGroup != -1 {
		for _, hero := range l.heroes {
			hero.tex.Destroy()
		}
	}
	if l.reqGroup == -1 {
		l.heroes = nil
	} else {
		l.heroes = make([]DisplayedHero, len(common.Groups[l.reqGroup].Heroes))
		var err error
		for index := range l.heroes {
			hero := &l.heroes[index]
			hero.shown = true
			hero.tex, err = common.Renderer.CreateTexture(sdl.PIXELFORMAT_RGB888, sdl.TEXTUREACCESS_TARGET,
				l.boxWidth(), l.boxHeight())
			if err == nil {
				common.Renderer.SetRenderTarget(hero.tex)
				name := renderText(common.Groups[l.reqGroup].Heroes[index].Name, common, 0)
				_, _, nameWidth, nameHeight, _ := name.Query()
				descr := renderText(common.Groups[l.reqGroup].Heroes[index].Description, common, 0)
				_, _, descrWidth, descrHeight, _ := descr.Query()
				common.Renderer.Clear()
				common.Renderer.SetDrawColor(0, 0, 0, 192)
				common.Renderer.FillRect(&sdl.Rect{X: 0, Y: 0, W: l.boxWidth(), H: l.boxHeight()})
				common.Renderer.SetDrawColor(200, 173, 127, 255)
				common.Renderer.FillRect(&sdl.Rect{X: 0, Y: l.borderWidth, W: int32(l.contentWidth + 4*l.borderWidth),
					H: int32(l.contentHeight + 2*l.borderWidth)})
				common.Renderer.Copy(name, nil, &sdl.Rect{X: 2 * l.borderWidth, Y: l.borderWidth, W: nameWidth, H: nameHeight})
				common.Renderer.Copy(descr, nil, &sdl.Rect{X: 2 * l.borderWidth,
					Y: l.contentHeight - l.borderWidth - descrHeight, W: descrWidth, H: descrHeight})
				name.Destroy()
				descr.Destroy()
			} else {
				log.Println(err)
			}
		}
		common.Renderer.SetRenderTarget(nil)
	}
}

func (l *HeroList) InitTransition(common *module.SceneCommon) time.Duration {
	if l.reqGroup != l.displayedGroup {
		if l.displayedGroup == -1 {
			l.rebuildHeroBoxes(common)
			l.status = ShowingAll
			return time.Second
		} else if l.reqGroup == -1 {
			l.status = HidingAll
			return time.Second
		} else {
			l.status = HidingAll
			return time.Second * 2
		}
	} else if l.reqHidden != l.hidden {
		if l.hidden {
			l.status = ShowingAll
		} else {
			l.status = HidingAll
		}
		return time.Second
	} else {
		if l.heroes[l.reqSwitch].shown {
			l.status = HidingHero
		} else {
			l.status = ShowingHero
		}
		l.heroes[l.reqSwitch].tex.SetBlendMode(sdl.BLENDMODE_BLEND)
		return time.Second
	}
}

func (l *HeroList) TransitionStep(common *module.SceneCommon, elapsed time.Duration) {
	if l.reqGroup != l.displayedGroup {
		if l.displayedGroup == -1 {
			l.curXOffset = int32(((time.Second - elapsed) * time.Duration(l.boxWidth())) / time.Second)
		} else if l.reqGroup == -1 {
			l.curXOffset = int32((elapsed * time.Duration(l.boxWidth())) / time.Second)
		} else {
			if elapsed >= time.Second {
				if l.status == HidingAll {
					l.rebuildHeroBoxes(common)
				}
				l.curXOffset = int32(((time.Second*2 - elapsed) * time.Duration(l.boxWidth())) / time.Second)
			} else {
				l.curXOffset = int32((elapsed * time.Duration(l.boxWidth())) / time.Second)
			}
		}
	} else if l.reqHidden != l.hidden {
		if l.status == ShowingAll {
			l.curXOffset = int32(((time.Second - elapsed) * time.Duration(l.boxWidth())) / time.Second)
		} else {
			l.curXOffset = int32((elapsed * time.Duration(l.boxWidth())) / time.Second)
		}
	} else {
		if l.status == ShowingHero {
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

func (l *HeroList) FinishTransition(common *module.SceneCommon) {
	l.curXOffset = 0
	l.curYOffset = 0
	l.status = Resting
	if l.reqSwitch != -1 {
		l.heroes[l.reqSwitch].tex.SetAlphaMod(255)
		l.heroes[l.reqSwitch].tex.SetBlendMode(sdl.BLENDMODE_NONE)
	}
	l.reqSwitch = -1
	l.hidden = l.reqHidden
	if l.reqGroup == -1 {
		for i := range l.heroes {
			l.heroes[i].tex.Destroy()
		}
	}
	l.displayedGroup = l.reqGroup
}

func (l *HeroList) Render(common *module.SceneCommon) {
	shown := int32(0)
	additionalYOffset := int32(0)
	if l.hidden && l.status == Resting {
		return
	}
	for i := range l.heroes {
		if l.reqSwitch != int32(i) && !l.heroes[i].shown {
			continue
		}
		xOffset := int32(0)
		if l.status == ShowingAll || l.status == HidingAll ||
			((l.status == ShowingHero || l.status == HidingHero) && l.reqSwitch == int32(i)) {
			xOffset = l.curXOffset
		}
		_, winHeight := common.Window.GetSize()
		if xOffset == 0 {
			targetRect := sdl.Rect{X: 0, Y: winHeight/10 + (l.boxHeight()+l.contentHeight/4)*shown + additionalYOffset,
				W: l.boxWidth(), H: l.boxHeight()}
			if err := common.Renderer.Copy(l.heroes[i].tex, nil, &targetRect); err != nil {
				log.Println(err)
			}
		} else {
			targetRect := sdl.Rect{X: 0, Y: winHeight/10 + (l.boxHeight()+l.contentHeight/4)*shown + additionalYOffset,
				W: l.boxWidth() - xOffset, H: l.boxHeight()}
			sourceRect := sdl.Rect{X: l.curXOffset, Y: 0, W: l.boxWidth(), H: l.boxHeight()}
			if err := common.Renderer.Copy(l.heroes[i].tex, &sourceRect, &targetRect); err != nil {
				log.Println(err)
			}
		}

		if (l.status == ShowingHero || l.status == HidingHero) && l.reqSwitch == int32(i) {
			additionalYOffset = l.curYOffset
		} else {
			shown++
		}
	}
}

func (*HeroList) SystemChanged(common *module.SceneCommon) bool {
	return false
}

func (l *HeroList) GroupChanged(common *module.SceneCommon) bool {
	if common.ActiveGroup != l.displayedGroup {
		if l.hidden {
			l.displayedGroup = common.ActiveGroup
			return false
		} else {
			l.reqGroup = common.ActiveGroup
			return true
		}
	}
	return false
}
