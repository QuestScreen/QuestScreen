package display

import (
	"errors"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/QuestScreen/QuestScreen/app"
	"github.com/QuestScreen/QuestScreen/shared"
	"github.com/QuestScreen/api/modules"
	"github.com/QuestScreen/api/render"
	"github.com/QuestScreen/api/server"
	"github.com/veandco/go-sdl2/sdl"
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
	r                    renderer
	actions              []KeyAction
	moduleRenderers      []modules.Renderer
	Window               *sdl.Window
	textureBuffer        uint32
	moduleStates         []moduleState
	numTransitions       int32
	popupTexture         render.Image
	welcomeTexture       render.Image
	initial              bool
	enabledModules       []bool
	queuedEnabledModules []bool
	request              uint32
}

// KeyAction describes a key that closes the app with the given return value
type KeyAction struct {
	Key         sdl.Keycode
	ReturnValue int
	Description string
}

// Init initializes the display. The renderer and window need to be generated
// before since the app needs to load fonts based on the window size.
func (d *Display) Init(
	owner app.App, events Events, fullscreen bool, port uint16,
	actions []KeyAction, window *sdl.Window, debug bool) error {
	d.owner = owner
	d.Events = events
	d.actions = actions
	d.Window = window
	d.initial = true

	sdl.ShowCursor(sdl.DISABLE)

	width, height := window.GLGetDrawableSize()
	d.r.init(width, height, len(owner.GetTextures()), debug)
	d.numTransitions = 0

	dRect := d.OutputSize()
	d.genPopup(dRect, actions)
	if err := d.genWelcome(dRect, port); err != nil {
		return err
	}

	d.moduleStates = make([]moduleState, d.owner.NumModules())

	d.moduleRenderers = make([]modules.Renderer, d.owner.NumModules())
	for i := shared.FirstModule; i < shared.ModuleIndex(len(d.moduleRenderers)); i++ {
		var err error
		d.moduleRenderers[i], err = d.owner.ModuleAt(i).CreateRenderer(
			d, owner.MessageSenderFor(i))
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Display) render(cur time.Time, popup bool) {
	d.r.clear()
	frame := d.OutputSize()
	if d.initial {
		d.welcomeTexture.Draw(d, frame, 255)
	} else {
		for i := shared.FirstModule; i < d.owner.NumModules(); i++ {
			if d.enabledModules[i] {
				ccount := d.r.canvasCount()
				state := &d.moduleStates[i]
				r := d.moduleRenderers[i]
				if state.transitioning {
					if cur.After(state.transEnd) {
						r.FinishTransition(d)
						d.numTransitions--
						state.transitioning = false
					} else {
						r.TransitionStep(d, cur.Sub(state.transStart))
					}
				}
				r.Render(d)
				if ccount != d.r.canvasCount() {
					panic("module " + d.owner.ModuleAt(i).Name + " failed to close all its canvases!")
				}
			}
		}
	}

	if popup && !d.popupTexture.IsEmpty() {
		d.popupTexture.Draw(d, frame, 255)
	}
	d.Window.GLSwap()
}

func (d *Display) startTransition(moduleIndex shared.ModuleIndex) {
	r := d.moduleRenderers[moduleIndex]
	state := &d.moduleStates[moduleIndex]
	if state.queuedData == nil {
		panic("Trying to call InitTransition without data")
	}
	if state.transitioning {
		r.FinishTransition(d)
		d.numTransitions--
		state.transitioning = false
	}

	transDur := r.InitTransition(d, state.queuedData)
	state.queuedData = nil
	if transDur == 0 {
		r.FinishTransition(d)
	} else if transDur > 0 {
		d.numTransitions++

		state.transStart = time.Now()
		state.transEnd = state.transStart.Add(transDur)
		state.transitioning = true
	}
}

// RenderLoop implements the rendering loop for the display.
// This function MUST be called in the main thread
func (d *Display) RenderLoop() int {
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
			/*waitTime := (time.Second / 80) - time.Now().Sub(curTime)
			if waitTime > 0 {
				event = sdl.WaitEventTimeout(int(waitTime / time.Millisecond))
			}*/
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
						for i := range d.actions {
							if e.Keysym.Sym == d.actions[i].Key {
								return d.actions[i].ReturnValue
							}
						}
						render = true
						popup = false
					} else {
						render = true
						popup = true
					}
				}
			case *sdl.QuitEvent:
				return 0
			case *sdl.WindowEvent:
				switch e.Event {
				case sdl.WINDOWEVENT_SHOWN, sdl.WINDOWEVENT_EXPOSED:
					render = true
				}
			case *sdl.UserEvent:
				switch e.Type {
				case d.Events.ModuleUpdateID:
					d.startTransition(shared.ModuleIndex(e.Code))
				case d.Events.SceneChangeID:
					d.enabledModules = d.queuedEnabledModules
					d.queuedEnabledModules = nil
					d.initial = false
					fallthrough
				case d.Events.ModuleConfigID:
					for i := shared.FirstModule; i < d.owner.NumModules(); i++ {
						r := d.moduleRenderers[i]
						state := &d.moduleStates[i]
						if state.queuedConfig != nil || state.queuedData != nil {
							r.Rebuild(d, state.queuedData, state.queuedConfig)
							state.queuedData = nil
							state.queuedConfig = nil
						}
					}
				case d.Events.HeroesChangedID:
					for i := shared.FirstModule; i < d.owner.NumModules(); i++ {
						state := &d.moduleStates[i]
						if state.queuedData != nil {
							r := d.moduleRenderers[i]
							r.Rebuild(d, state.queuedData, nil)
							state.queuedData = nil
						}
					}
				case d.Events.LeaveGroupID:
					d.initial = true
				}
				render = true
				atomic.StoreUint32(&d.request, noRequest)
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
	server.Error) {
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
func (r *Request) SendModuleConfig(index shared.ModuleIndex, config interface{}) error {
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

// SendRendererData queues the given data for the module at the given ID as part
// of the request. Whether the data is used for RebuiltState or InitTransition
// depends on the event ID of the request.
func (r *Request) SendRendererData(index shared.ModuleIndex, data interface{}) error {
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
	d.r.close()
	d.Window.Destroy()
}
