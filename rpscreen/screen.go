package main

import (
	"github.com/flyx/rpscreen/module"
	"github.com/flyx/rpscreen/module/background"
	"github.com/flyx/rpscreen/module/herolist"
	"github.com/flyx/rpscreen/module/persons"
	"github.com/flyx/rpscreen/module/title"
	"github.com/veandco/go-sdl2/sdl"
	"time"
)

type moduleListItem struct {
	module        module.Module
	enabled       bool
	transStart    time.Time
	transEnd      time.Time
	transitioning bool
}

type Screen struct {
	module.SceneCommon
	textureBuffer       uint32
	modules             []moduleListItem
	numTransitions      int32
	moduleUpdateEventId uint32
	groupUpdateEventId  uint32
	systemUpdateEventId uint32
	popupTexture        *sdl.Texture
}

func newScreen() (*Screen, error) {
	screen := new(Screen)
	/*egl.QuerySurface(eglState.Display, eglState.Surface, egl.WIDTH, &width)
	egl.QuerySurface(eglState.Display, eglState.Surface, egl.HEIGHT, &height)*/
	var err error
	screen.Window, err = sdl.CreateWindow("rpscreen", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		800, 600, sdl.WINDOW_OPENGL)
	if err != nil {
		return nil, err
	}
	width, height := screen.Window.GetSize()
	screen.DefaultBorderWidth = height / 133

	screen.Renderer, err = sdl.CreateRenderer(screen.Window, -1,
		sdl.RENDERER_ACCELERATED|sdl.RENDERER_TARGETTEXTURE)
	if err != nil {
		screen.Window.Destroy()
		return nil, err
	}

	screen.modules = make([]moduleListItem, 0, 16)
	screen.SharedData = module.InitSharedData()
	screen.numTransitions = 0
	screen.moduleUpdateEventId = sdl.RegisterEvents(3)
	screen.groupUpdateEventId = screen.moduleUpdateEventId + 1
	screen.systemUpdateEventId = screen.moduleUpdateEventId + 2
	screen.Fonts = module.CreateFontCatalog(&screen.SharedData, int(height)/13)

	bg := new(background.Background)
	if err := bg.Init(&screen.SceneCommon); err != nil {
		panic(err)
	}
	screen.modules = append(screen.modules, moduleListItem{module: bg, enabled: true, transitioning: false})
	if len(screen.Fonts) > 0 {
		t := new(title.Title)
		if err := t.Init(&screen.SceneCommon); err != nil {
			panic(err)
		}
		screen.modules = append(screen.modules, moduleListItem{module: t, enabled: true, transitioning: false})
	}

	p := new(persons.Persons)
	if err := p.Init(&screen.SceneCommon); err != nil {
		panic(err)
	}
	screen.modules = append(screen.modules, moduleListItem{module: p, enabled: true, transitioning: false})

	h := new(herolist.HeroList)
	if err := h.Init(&screen.SceneCommon); err != nil {
		panic(err)
	}
	screen.modules = append(screen.modules, moduleListItem{module: h, enabled: true, transitioning: false})

	// create picture of popup for key presses

	screen.popupTexture, err = screen.Renderer.CreateTexture(sdl.PIXELFORMAT_RGB888, sdl.TEXTUREACCESS_TARGET,
		width, height)
	if err != nil {
		panic(err)
	}
	screen.Renderer.SetRenderTarget(screen.popupTexture)
	defer screen.Renderer.SetRenderTarget(nil)
	screen.Renderer.Clear()
	screen.Renderer.SetDrawColor(0, 0, 0, 127)
	screen.Renderer.FillRect(nil)
	rect := sdl.Rect{X: width / 4, Y: height / 4, W: width / 2, H: height / 2}
	screen.Renderer.SetDrawColor(0, 0, 0, 255)
	screen.Renderer.FillRect(&rect)
	rect.X += screen.DefaultBorderWidth
	rect.Y += screen.DefaultBorderWidth
	rect.W -= 2 * screen.DefaultBorderWidth
	rect.H -= 2 * screen.DefaultBorderWidth
	screen.Renderer.SetDrawColor(255, 255, 255, 255)
	screen.Renderer.FillRect(&rect)

	surface, err := screen.Fonts[0].Font.RenderUTF8Blended(
		"Press X to quit, S to shut down, or any other key to close popup", sdl.Color{R: 0, G: 0, B: 0, A: 230})
	if err != nil {
		panic(err)
	}
	textTexture, err := screen.Renderer.CreateTextureFromSurface(surface)
	if err != nil {
		panic(err)
	}
	defer textTexture.Destroy()
	textWidth := surface.W
	textHeight := surface.H
	screen.Renderer.Copy(textTexture, nil, &sdl.Rect{X: (width - textWidth) / 2, Y: (height - textHeight) / 2,
		W: textWidth, H: textHeight})
	surface.Free()
	screen.popupTexture.SetBlendMode(sdl.BLENDMODE_BLEND)
	return screen, nil
}

func (s *Screen) Render(cur time.Time, popup bool) {
	s.Renderer.Clear()
	s.Renderer.SetDrawColor(255, 255, 255, 255)
	s.Renderer.FillRect(nil)
	for i := 0; i < len(s.modules); i++ {
		item := &s.modules[i]
		if item.enabled {
			if item.transitioning {
				if cur.After(item.transEnd) {
					item.module.FinishTransition(&s.SceneCommon)
					s.numTransitions--
					item.transitioning = false
				} else {
					item.module.TransitionStep(&s.SceneCommon, cur.Sub(item.transStart))
				}
			}
			item.module.Render(&s.SceneCommon)
		}
	}
	if popup {
		s.Renderer.Copy(s.popupTexture, nil, nil)
	}
	s.Renderer.Present()
}
