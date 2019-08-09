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

type keyOption struct {
	key  string
	desc string
}

func (s *Screen) shrinkByBorder(rect *sdl.Rect) {
	rect.X += s.DefaultBorderWidth
	rect.Y += s.DefaultBorderWidth
	rect.W -= 2 * s.DefaultBorderWidth
	rect.H -= 2 * s.DefaultBorderWidth
}

func shrinkTo(rect *sdl.Rect, w int32, h int32) {
	xStep := (rect.W - w) / 2
	yStep := (rect.H - h) / 2
	rect.X += xStep
	rect.Y += yStep
	rect.W -= 2 * xStep
	rect.H -= 2 * yStep
}

func (s *Screen) renderKeyOptions(frame *sdl.Rect, options ...keyOption) error {
	surfaces := make([]*sdl.Surface, len(options))
	fontFace := s.Fonts[0].GetSize(s.DefaultBodyTextSize).GetFace(module.Standard)
	var err error
	var bottomText *sdl.Surface
	if bottomText, err = fontFace.RenderUTF8Blended(
		"any other key to close", sdl.Color{R: 0, G: 0, B: 0, A: 200}); err != nil {
		return err
	}
	defer bottomText.Free()

	maxHeight := bottomText.H
	for i := range options {
		if surfaces[i], err = fontFace.RenderUTF8Blended(
			options[i].desc, sdl.Color{R: 0, G: 0, B: 0, A: 230}); err != nil {
			for j := 0; j < i; j++ {
				surfaces[j].Free()
			}
			return err
		}
		//noinspection GoDeferInLoop
		if surfaces[i].H > maxHeight {
			maxHeight = surfaces[i].H
		}
	}
	defer func() {
		for i := range surfaces {
			surfaces[i].Free()
		}
	}()
	padding := (frame.H - maxHeight*int32(len(options)+1)) / (2 * int32(len(options)+1))
	curY := frame.Y + padding
	for i := range options {
		curRect := sdl.Rect{X: frame.X + padding - 2*s.DefaultBorderWidth,
			Y: curY - 2*s.DefaultBorderWidth, W: maxHeight + 4*s.DefaultBorderWidth,
			H: maxHeight + 4*s.DefaultBorderWidth}
		s.Renderer.SetDrawColor(0, 0, 0, 255)
		s.Renderer.FillRect(&curRect)
		s.shrinkByBorder(&curRect)
		s.Renderer.SetDrawColor(255, 255, 255, 255)
		s.Renderer.FillRect(&curRect)
		var keySurface *sdl.Surface
		if keySurface, err = fontFace.RenderUTF8Blended(
			options[i].key, sdl.Color{R: 0, G: 0, B: 0, A: 230}); err != nil {
			return err
		}
		keyTex, err := s.Renderer.CreateTextureFromSurface(keySurface)
		if err != nil {
			keySurface.Free()
			return err
		}
		shrinkTo(&curRect, keySurface.W, keySurface.H)
		s.Renderer.Copy(keyTex, nil, &curRect)
		keySurface.Free()
		keyTex.Destroy()

		textTex, err := s.Renderer.CreateTextureFromSurface(surfaces[i])
		if err != nil {
			return err
		}
		curRect = sdl.Rect{X: frame.X + padding + maxHeight + 4*s.DefaultBorderWidth,
			Y: curY, W: surfaces[i].W, H: maxHeight}
		shrinkTo(&curRect, surfaces[i].W, surfaces[i].H)
		s.Renderer.Copy(textTex, nil, &curRect)
		textTex.Destroy()

		curY = curY + padding*2 + maxHeight
	}

	var bottomTextTex *sdl.Texture
	if bottomTextTex, err = s.Renderer.CreateTextureFromSurface(bottomText); err != nil {
		return err
	}
	bottomRect := sdl.Rect{X: frame.X, Y: curY, W: frame.W, H: maxHeight}
	shrinkTo(&bottomRect, bottomText.W, bottomText.H)
	s.Renderer.Copy(bottomTextTex, nil, &bottomRect)
	bottomTextTex.Destroy()
	return nil
}

func (s *Screen) genPopup(width int32, height int32) {
	var err error
	s.popupTexture, err = s.Renderer.CreateTexture(sdl.PIXELFORMAT_RGB888, sdl.TEXTUREACCESS_TARGET,
		width, height)
	if err != nil {
		panic(err)
	}
	s.Renderer.SetRenderTarget(s.popupTexture)
	defer s.Renderer.SetRenderTarget(nil)
	s.Renderer.Clear()
	s.Renderer.SetDrawColor(0, 0, 0, 127)
	s.Renderer.FillRect(nil)
	rect := sdl.Rect{X: width / 4, Y: height / 4, W: width / 2, H: height / 2}
	s.Renderer.SetDrawColor(0, 0, 0, 255)
	s.Renderer.FillRect(&rect)
	s.shrinkByBorder(&rect)
	s.Renderer.SetDrawColor(255, 255, 255, 255)
	s.Renderer.FillRect(&rect)

	/*surface, err := s.Fonts[0].GetSize(s.DefaultBodyTextSize).GetFace(module.Standard).RenderUTF8Blended(
		"Press X to quit, S to shut down, or any other key to close popup", sdl.Color{R: 0, G: 0, B: 0, A: 230})
	if err != nil {
		panic(err)
	}
	textTexture, err := s.Renderer.CreateTextureFromSurface(surface)
	if err != nil {
		panic(err)
	}
	defer textTexture.Destroy()
	textWidth := surface.W
	textHeight := surface.H
	s.Renderer.Copy(textTexture, nil, &sdl.Rect{X: (width - textWidth) / 2, Y: (height - textHeight) / 2,
		W: textWidth, H: textHeight})
	surface.Free()*/
	if err = s.renderKeyOptions(&rect, keyOption{key: "X", desc: "Quit"}, keyOption{key: "S", desc: "Shutdown"}); err != nil {
		panic(err)
	}
	s.popupTexture.SetBlendMode(sdl.BLENDMODE_BLEND)
}

func (s *Screen) loadModules() {
	s.modules = make([]moduleListItem, 0, 16)
	bg := new(background.Background)
	if err := bg.Init(&s.SceneCommon); err != nil {
		panic(err)
	}
	s.modules = append(s.modules, moduleListItem{module: bg, enabled: true, transitioning: false})
	if len(s.Fonts) > 0 {
		t := new(title.Title)
		if err := t.Init(&s.SceneCommon); err != nil {
			panic(err)
		}
		s.modules = append(s.modules, moduleListItem{module: t, enabled: true, transitioning: false})
	}

	p := new(persons.Persons)
	if err := p.Init(&s.SceneCommon); err != nil {
		panic(err)
	}
	s.modules = append(s.modules, moduleListItem{module: p, enabled: true, transitioning: false})

	h := new(herolist.HeroList)
	if err := h.Init(&s.SceneCommon); err != nil {
		panic(err)
	}
	s.modules = append(s.modules, moduleListItem{module: h, enabled: true, transitioning: false})
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
	screen.DefaultHeadingTextSize = height / 13
	screen.DefaultBodyTextSize = height / 27

	screen.Renderer, err = sdl.CreateRenderer(screen.Window, -1,
		sdl.RENDERER_ACCELERATED|sdl.RENDERER_TARGETTEXTURE)
	if err != nil {
		screen.Window.Destroy()
		return nil, err
	}

	screen.SharedData = module.InitSharedData()
	screen.numTransitions = 0
	screen.moduleUpdateEventId = sdl.RegisterEvents(3)
	screen.groupUpdateEventId = screen.moduleUpdateEventId + 1
	screen.systemUpdateEventId = screen.moduleUpdateEventId + 2
	screen.Fonts = module.CreateFontCatalog(&screen.SharedData, screen.DefaultBodyTextSize)

	screen.loadModules()
	screen.genPopup(width, height)

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
