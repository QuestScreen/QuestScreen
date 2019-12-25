package display

import (
	"log"
	"time"

	"github.com/flyx/pnpscreen/api"
	"github.com/flyx/pnpscreen/app"
	"github.com/veandco/go-sdl2/sdl"
)

// ModuleConfigUpdate is a message requesting to configure the item with the
// given index with the given config.
type ModuleConfigUpdate struct {
	Index  api.ModuleIndex
	Config interface{}
}

// SceneUpdate is a message originating in a scene change. It carries the
// information of which modules shall be enabled in the new scene.
type SceneUpdate struct {
	ModuleEnabled []bool
}

type animationState struct {
	transStart    time.Time
	transEnd      time.Time
	transitioning bool
}

// Display describes a display rendering scenes.
type Display struct {
	Events
	owner          app.App
	Renderer       *sdl.Renderer
	Window         *sdl.Window
	textureBuffer  uint32
	moduleStates   []animationState
	numTransitions int32
	popupTexture   *sdl.Texture
	welcomeTexture *sdl.Texture
	initial        bool
	enabledModules []bool
}

// Init initializes the display. The renderer and window need to be generated
// before since the app needs to load fonts based on the window size.
func (d *Display) Init(
	owner app.App, events Events, fullscreen bool, port uint16,
	window *sdl.Window, renderer *sdl.Renderer) error {
	d.owner = owner
	d.Events = events
	d.Window = window
	d.Renderer = renderer
	d.initial = true

	width, height, err := d.Renderer.GetOutputSize()
	if err != nil {
		return err
	}
	d.numTransitions = 0

	d.genPopup(width, height)
	if err = d.genWelcome(width, height, port); err != nil {
		return err
	}

	d.moduleStates = make([]animationState, d.owner.NumModules())
	return nil
}

func (d *Display) render(cur time.Time, popup bool) {
	ctx := api.RenderContext{Renderer: d.Renderer, Env: d.owner}
	d.Renderer.Clear()
	if d.initial {
		_ = d.Renderer.Copy(d.welcomeTexture, nil, nil)
	} else {
		d.Renderer.SetDrawColor(255, 255, 255, 255)
		d.Renderer.FillRect(nil)
		for i := api.ModuleIndex(0); i < d.owner.NumModules(); i++ {
			if d.enabledModules[i] {
				state := &d.moduleStates[i]
				module := d.owner.ModuleAt(i)
				if state.transitioning {
					if cur.After(state.transEnd) {
						module.FinishTransition(ctx)
						d.numTransitions--
						state.transitioning = false
					} else {
						module.TransitionStep(ctx, cur.Sub(state.transStart))
					}
				}
				module.Render(ctx)
			}
		}
	}
	if popup {
		d.Renderer.Copy(d.popupTexture, nil, nil)
	}
	d.Renderer.Present()
}

func (d *Display) startTransition(moduleIndex api.ModuleIndex) {
	ctx := api.RenderContext{Renderer: d.Renderer, Env: d.owner}
	module := d.owner.ModuleAt(moduleIndex)
	transDur := module.InitTransition(ctx)
	if transDur == 0 {
		module.FinishTransition(ctx)
	} else if transDur > 0 {
		d.numTransitions++
		state := &d.moduleStates[moduleIndex]
		state.transStart = time.Now()
		state.transEnd = state.transStart.Add(transDur)
		state.transitioning = true
	}
}

// RenderLoop implements the rendering loop for the display.
// This function MUST be called in the main thread
func (d *Display) RenderLoop(
	modConfigChan chan ModuleConfigUpdate, sceneChan chan SceneUpdate) {
	render := true
	popup := false

	var start = time.Now()
	var frameCount = int64(0)

	for {
		curTime := time.Now()
		if render {
			frameCount++
			d.render(curTime, popup)
			if curTime.Sub(start) >= time.Second {
				log.Printf("FPS: %d\n", frameCount)
				start = curTime
				frameCount = 0
			}
		}
		var event sdl.Event
		if d.numTransitions > 0 {
			waitTime := (time.Second / 30) - time.Now().Sub(curTime)
			if waitTime > 0 {
				event = sdl.WaitEventTimeout(int(waitTime / time.Millisecond))
			}
		} else {
			render = false
			event = sdl.WaitEvent()
		}
		inRender := render
		for ; event != nil; event = sdl.PollEvent() {
			switch e := event.(type) {
			case *sdl.KeyboardEvent:
				if e.Type == sdl.KEYDOWN {
					if popup {
						switch e.Keysym.Sym {
						case sdl.K_x:
							return
						case sdl.K_s:
							return
						default:
							render = true
							popup = false
						}
					} else {
						render = true
						popup = true
					}
				}
			case *sdl.QuitEvent:
				return
			case *sdl.UserEvent:
				switch e.Type {
				case d.Events.ModuleUpdateID:
					d.startTransition(api.ModuleIndex(e.Code))
					render = true
				case d.Events.SceneChangeID:
				outer1:
					for {
						select {
						case data := <-sceneChan:
							d.enabledModules = data.ModuleEnabled
							render = true
						default:
							break outer1
						}
					}
					fallthrough
				case d.Events.ModuleConfigID:
					ctx := api.RenderContext{Renderer: d.Renderer, Env: d.owner}
				outer2:
					for {
						select {
						case data := <-modConfigChan:
							module := d.owner.ModuleAt(data.Index)
							module.SetConfig(data.Config)
							module.RebuildState(ctx)
							render = true
						default:
							break outer2
						}
					}
					d.initial = false
				}
			}
		}
		if render && !inRender {
			start = time.Now()
			frameCount = 0
		}
	}
}

// Destroy destroys window and renderer
func (d *Display) Destroy() {
	d.Renderer.Destroy()
	d.Window.Destroy()
}
