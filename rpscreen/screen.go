package main

import (
	"github.com/flyx/rpscreen/module"
	"github.com/flyx/rpscreen/module/background"
	"github.com/flyx/rpscreen/module/sceneTitle"
	"github.com/veandco/go-sdl2/sdl"
	"os"
	"os/user"
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
}

func newScreen() (*Screen, error) {
	usr, _ := user.Current()

	screen := new(Screen)

	var width, height int32
	width = 800
	height = 600
	/*egl.QuerySurface(eglState.Display, eglState.Surface, egl.WIDTH, &width)
	egl.QuerySurface(eglState.Display, eglState.Surface, egl.HEIGHT, &height)*/
	screen.Ratio = float32(width) / float32(height)
	var err error
	screen.Window, err = sdl.CreateWindow("rpscreen", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		width, height, sdl.WINDOW_SHOWN)
	if err != nil {
		return nil, err
	}

	screen.Renderer, err = sdl.CreateRenderer(screen.Window, -1, sdl.RENDERER_ACCELERATED|sdl.RENDERER_TARGETTEXTURE)
	if err != nil {
		screen.Window.Destroy()
		return nil, err
	}

	screen.modules = make([]moduleListItem, 0, 16)
	screen.DataDir = usr.HomeDir + "/.local/share/rpscreen"
	if err := os.MkdirAll(screen.DataDir, 0700); err != nil {
		panic(err)
	}
	screen.numTransitions = 0
	screen.moduleUpdateEventId = sdl.RegisterEvents(1)
	screen.Fonts = module.CreateFontCatalog(screen.DataDir)

	bg := new(background.Background)
	if err := bg.Init(&screen.SceneCommon); err != nil {
		panic(err)
	}
	screen.modules = append(screen.modules, moduleListItem{module: bg, enabled: true, transitioning: false})
	if len(screen.Fonts) > 0 {
		title := new(sceneTitle.SceneTitle)
		if err := title.Init(&screen.SceneCommon); err != nil {
			panic(err)
		}
		screen.modules = append(screen.modules, moduleListItem{module: title, enabled: true, transitioning: false})
	}
	return screen, nil
}

func (s *Screen) Render(cur time.Time) {
	s.Renderer.Clear()
	s.Renderer.SetDrawColor(255, 255, 255, 255)
	winWidth, winHeight := s.Window.GetSize()
	s.Renderer.FillRect(&sdl.Rect{X: 0, Y: 0, W: int32(winWidth), H: int32(winHeight)})
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
	s.Renderer.Present()
}
