package display

import (
	"errors"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/flyx/pnpscreen/api"
	"github.com/flyx/pnpscreen/app"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

type moduleState struct {
	queuedData, queuedConfig interface{}
	transStart, transEnd     time.Time
	transitioning            bool
}

const (
	noRequest uint32 = iota
	activeRequest
)

// Display describes a display rendering scenes.
type Display struct {
	Events
	owner                app.App
	Renderer             *sdl.Renderer
	Window               *sdl.Window
	defaultBorderWidth   int32
	textureBuffer        uint32
	moduleStates         []moduleState
	numTransitions       int32
	popupTexture         *sdl.Texture
	welcomeTexture       *sdl.Texture
	initial              bool
	enabledModules       []bool
	queuedEnabledModules []bool
	request              uint32
}

// renderContext implements api.ExtendedRenderContext
type renderContext struct {
	*Display
	moduleIndex app.ModuleIndex
	heroes      app.HeroView
}

func (rc renderContext) GetResources(index api.ResourceCollectionIndex) []api.Resource {
	return rc.owner.GetResources(rc.moduleIndex, index)
}

func (rc renderContext) Renderer() *sdl.Renderer {
	return rc.Display.Renderer
}

func (rc renderContext) Font(
	fontFamily int, style api.FontStyle, size api.FontSize) *ttf.Font {
	return rc.owner.Font(fontFamily, style, size)
}

func (rc renderContext) DefaultBorderWidth() int32 {
	return rc.defaultBorderWidth
}

func (rc renderContext) Heroes() api.HeroList {
	return rc.heroes
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
	d.defaultBorderWidth = height / 133
	d.numTransitions = 0

	d.genPopup(width, height)
	if err = d.genWelcome(width, height, port); err != nil {
		return err
	}

	d.moduleStates = make([]moduleState, d.owner.NumModules())
	return nil
}

func (d *Display) render(cur time.Time, popup bool) {
	ctx := renderContext{Display: d}
	d.Renderer.Clear()
	if d.initial {
		_ = d.Renderer.Copy(d.welcomeTexture, nil, nil)
	} else {
		d.Renderer.SetDrawColor(255, 255, 255, 255)
		d.Renderer.FillRect(nil)
		for i := app.FirstModule; i < d.owner.NumModules(); i++ {
			if d.enabledModules[i] {
				ctx.moduleIndex = i
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

func (d *Display) startTransition(moduleIndex app.ModuleIndex) {
	ctx := renderContext{Display: d, moduleIndex: moduleIndex}
	module := d.owner.ModuleAt(moduleIndex)
	state := &d.moduleStates[moduleIndex]
	if state.queuedData == nil {
		panic("Trying to call InitTransition without data")
	}
	transDur := module.InitTransition(ctx, state.queuedData)
	state.queuedData = nil
	if transDur == 0 {
		module.FinishTransition(ctx)
	} else if transDur > 0 {
		d.numTransitions++

		state.transStart = time.Now()
		state.transEnd = state.transStart.Add(transDur)
		state.transitioning = true
	}
}

// RenderLoop implements the rendering loop for the display.
// This function MUST be called in the main thread
func (d *Display) RenderLoop() {
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
					d.startTransition(app.ModuleIndex(e.Code))
					render = true
					atomic.StoreUint32(&d.request, noRequest)
				case d.Events.SceneChangeID:
					d.enabledModules = d.queuedEnabledModules
					d.queuedEnabledModules = nil
					fallthrough
				case d.Events.ModuleConfigID:
					ctx := renderContext{Display: d, heroes: d.owner.ViewHeroes()}
					for i := app.FirstModule; i < d.owner.NumModules(); i++ {
						ctx.moduleIndex = i
						module := d.owner.ModuleAt(i)
						state := &d.moduleStates[i]
						forceRebuild := false
						if state.queuedConfig != nil {
							module.SetConfig(state.queuedConfig)
							state.queuedConfig = nil
							forceRebuild = true
						}
						if forceRebuild || state.queuedData != nil {
							module.RebuildState(ctx, state.queuedData)
							state.queuedData = nil
						}
					}
					ctx.heroes.Close()
					render = true
					atomic.StoreUint32(&d.request, noRequest)
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

// Request is a pending message to the display thread.
// Successful generation of a request leads to exclusive access to the display's
// communication channel.
// A Request must be either committed or closed.
type Request struct {
	d         *Display
	eventID   uint32
	eventCode int32
}

var errMultipleModuleConfigs = errors.New("Cannot send multiple configs to same module in one request")
var errMultipleModuleData = errors.New("Cannot send multiple data objects to same module in one request")
var errMultipleEnabledModules = errors.New("Cannot send multiple enabledModules lists in one request")
var errAlreadyCommitted = errors.New("Request has already been committed")

// StartRequest starts a new request to the display thread.
// Returns an error if there is already a pending request.
func (d *Display) StartRequest(eventID uint32, eventCode int32) (Request,
	api.SendableError) {
	if eventID == sdl.FIRSTEVENT {
		panic("illegal SDL event ID")
	}
	if atomic.CompareAndSwapUint32(&d.request, noRequest, activeRequest) {
		return Request{d: d, eventID: eventID, eventCode: eventCode}, nil
	}
	return Request{}, &app.TooManyRequests{}
}

// SendModuleConfig queues the given config for the module at the given ID as
// part of the request.
func (r *Request) SendModuleConfig(index app.ModuleIndex, config interface{}) error {
	if index < 0 || index >= r.d.owner.NumModules() {
		return fmt.Errorf("Module index %d outside of range 0..%d", index, len(r.d.moduleStates))
	}
	state := &r.d.moduleStates[index]
	if state.queuedConfig != nil {
		return errMultipleModuleConfigs
	}
	state.queuedConfig = config
	return nil
}

// SendModuleData queues the given data for the module at the given ID as part
// of the request. Whether the data is used for RebuiltState or InitTransition
// depends on the event ID of the request.
func (r *Request) SendModuleData(index app.ModuleIndex, data interface{}) error {
	if index < 0 || index >= r.d.owner.NumModules() {
		return fmt.Errorf("Module index %d outside of range 0..%d", index, len(r.d.moduleStates))
	}
	state := &r.d.moduleStates[index]
	if state.queuedData != nil {
		return errMultipleModuleData
	}
	state.queuedData = data
	return nil
}

// SendEnabledModulesList queues the list of enabled modules as part of the
// request.
func (r *Request) SendEnabledModulesList(value []bool) error {
	if r.d.queuedEnabledModules != nil {
		return errMultipleEnabledModules
	}
	r.d.queuedEnabledModules = value
	return nil
}

// Commit sends the request to the display thread
func (r *Request) Commit() error {
	if r.eventID == sdl.FIRSTEVENT {
		return errAlreadyCommitted
	}
	sdl.PushEvent(&sdl.UserEvent{Type: r.eventID, Code: r.eventCode})
	r.eventID = sdl.FIRSTEVENT
	return nil
}

// Close closes the request. If the request has not been committed, the queued
// data will be erased. This function is idempotent.
func (r *Request) Close() {
	if r.eventID != sdl.FIRSTEVENT {
		for i := range r.d.moduleStates {
			state := r.d.moduleStates[i]
			state.queuedData = nil
			state.queuedConfig = nil
		}
		r.d.queuedEnabledModules = nil
		r.eventID = sdl.FIRSTEVENT
		atomic.StoreUint32(&r.d.request, noRequest)
	}
}

// Destroy destroys window and renderer
func (d *Display) Destroy() {
	d.Renderer.Destroy()
	d.Window.Destroy()
}
