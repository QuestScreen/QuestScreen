package display

import (
	"errors"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/QuestScreen/QuestScreen/app"
	"github.com/QuestScreen/api"
	"github.com/veandco/go-sdl2/img"
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
	renderers            []api.ModuleRenderer
	Backend              *sdl.Renderer
	Window               *sdl.Window
	unit                 int32
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

type canvas struct {
	prevRenderTarget *sdl.Texture
	target           *sdl.Texture
	renderer         *sdl.Renderer
}

func (c canvas) Finish() *sdl.Texture {
	ret := c.target
	c.target = nil
	c.renderer.SetRenderTarget(c.prevRenderTarget)
	return ret
}

func (c canvas) Close() {
	if c.target != nil {
		c.target.Destroy()
		c.target = nil
		c.renderer.SetRenderTarget(c.prevRenderTarget)
	}
}

func (rc renderContext) GetResources(index api.ResourceCollectionIndex) []api.Resource {
	return rc.owner.GetResources(rc.moduleIndex, index)
}

func (rc renderContext) GetTextures() []api.Resource {
	return rc.owner.GetTextures()
}

func (rc renderContext) Renderer() *sdl.Renderer {
	return rc.Display.Backend
}

func (rc renderContext) Font(
	fontFamily int, style api.FontStyle, size api.FontSize) *ttf.Font {
	return rc.owner.Font(fontFamily, style, size)
}

func (rc renderContext) Unit() int32 {
	return rc.unit
}

func (rc renderContext) Heroes() api.HeroList {
	return rc.heroes
}

func (rc renderContext) UpdateMask(target **sdl.Texture,
	bg api.SelectableTexturedBackground) {
	if *target != nil {
		(*target).Destroy()
		(*target) = nil
	}
	if bg.TextureIndex != -1 {
		textures := rc.owner.GetTextures()
		path := textures[bg.TextureIndex].Path()
		surface, err := img.Load(path)
		if err != nil {
			log.Printf("unable to load %s: %s\n", path, err.Error())
			return
		}
		if surface.Format.Format != sdl.PIXELFORMAT_INDEX8 {
			grayscale, err := surface.ConvertFormat(sdl.PIXELFORMAT_INDEX8, 0)
			if err != nil {
				log.Printf("could not convert %s to grayscale: %s\n", path,
					err.Error())
				return
			}
			surface.Free()
			surface = grayscale
		}
		colorSurface, err := sdl.CreateRGBSurfaceWithFormat(0, surface.W,
			surface.H, 32, uint32(sdl.PIXELFORMAT_RGBA32))
		grayPixels := surface.Pixels()
		colorPixels := colorSurface.Pixels()
		color := bg.Secondary
		for y := int32(0); y < colorSurface.H; y++ {
			for x := int32(0); x < colorSurface.W; x++ {
				offset := (y*colorSurface.W + x)
				cOffset := offset * 4
				copy(colorPixels[cOffset:cOffset+4], []byte{color.Red, color.Green,
					color.Blue, 255 - grayPixels[offset]})
			}
		}
		surface.Free()
		*target, err = rc.Display.Backend.CreateTextureFromSurface(colorSurface)
		colorSurface.Free()
		if err != nil {
			log.Printf("unable to create texture from %s: %s\n", path, err.Error())
		}
	}
}

func (rc renderContext) TextToTexture(
	text string, font *ttf.Font, color sdl.Color) *sdl.Texture {
	surface, err := font.RenderUTF8Blended(text, color)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer surface.Free()
	r := rc.Display.Backend
	textTexture, err := r.CreateTextureFromSurface(surface)
	if err != nil {
		log.Println(err)
		return nil
	}
	textTexture.SetBlendMode(sdl.BLENDMODE_BLEND)
	return textTexture
}

func (rc renderContext) CreateCanvas(innerWidth, innerHeight int32,
	background *sdl.Color, tile *sdl.Texture, borders api.Directions) api.Canvas {
	ret := canvas{renderer: rc.Display.Backend,
		prevRenderTarget: rc.Display.Backend.GetRenderTarget()}
	var err error
	width := innerWidth
	xOffset, yOffset := int32(0), int32(0)
	if borders&api.East != 0 {
		width += rc.Display.unit
	}
	if borders&api.West != 0 {
		width += rc.Display.unit
		xOffset = rc.Display.unit
	}
	height := innerHeight
	if borders&api.North != 0 {
		height += rc.Display.unit
		yOffset = rc.Display.unit
	}
	if borders&api.South != 0 {
		height += rc.Display.unit
	}
	ret.target, err = ret.renderer.CreateTexture(sdl.PIXELFORMAT_RGB888,
		sdl.TEXTUREACCESS_TARGET, width, height)
	if err != nil {
		panic(err)
	}
	ret.renderer.SetRenderTarget(ret.target)
	ret.renderer.SetDrawColor(0, 0, 0, 192)
	ret.renderer.Clear()
	if background != nil {
		ret.renderer.SetDrawColor(
			background.R, background.G, background.B, background.A)
		ret.renderer.FillRect(
			&sdl.Rect{X: xOffset, Y: yOffset, W: innerWidth, H: innerHeight})
	}
	if tile != nil {
		_, _, w, h, _ := tile.Query()
		targetRect := sdl.Rect{X: 0, Y: 0, W: w, H: h}
		for y := yOffset; y < innerHeight+yOffset; y += h {
			targetRect.Y, targetRect.H = y, h
			srcRect := sdl.Rect{X: -1, Y: -1, W: w, H: h}
			if y+h > innerHeight+yOffset {
				targetRect.H = innerHeight + yOffset - y
				srcRect = sdl.Rect{X: 0, Y: 0, W: w, H: targetRect.H}
			}

			for x := xOffset; x < innerWidth+xOffset; x += w {
				targetRect.X, targetRect.W = x, w
				if x+w > innerWidth+yOffset {
					targetRect.W = innerWidth + xOffset - x
					srcRect.X, srcRect.Y, srcRect.W = 0, 0, targetRect.W
				}
				if srcRect.X == -1 {
					ret.renderer.Copy(tile, &srcRect, &targetRect)
				} else {
					ret.renderer.Copy(tile, nil, &targetRect)
				}
			}

		}
	}
	return ret
}

// Init initializes the display. The renderer and window need to be generated
// before since the app needs to load fonts based on the window size.
func (d *Display) Init(
	owner app.App, events Events, fullscreen bool, port uint16,
	window *sdl.Window, renderer *sdl.Renderer) error {
	d.owner = owner
	d.Events = events
	d.Window = window
	d.Backend = renderer
	d.initial = true

	width, height, err := d.Backend.GetOutputSize()
	if err != nil {
		return err
	}
	if width < height {
		d.unit = width / 144
	} else {
		d.unit = height / 144
	}
	d.numTransitions = 0

	d.genPopup(width, height)
	if err = d.genWelcome(width, height, port); err != nil {
		return err
	}

	d.moduleStates = make([]moduleState, d.owner.NumModules())

	d.renderers = make([]api.ModuleRenderer, d.owner.NumModules())
	for i := app.FirstModule; i < app.ModuleIndex(len(d.renderers)); i++ {
		d.renderers[i], err = d.owner.ModuleAt(i).CreateRenderer(
			d.Backend, owner.MessageSenderFor(i))
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Display) render(cur time.Time, popup bool) {
	ctx := renderContext{Display: d}
	d.Backend.Clear()
	if d.initial {
		_ = d.Backend.Copy(d.welcomeTexture, nil, nil)
	} else {
		d.Backend.SetDrawColor(255, 255, 255, 255)
		d.Backend.FillRect(nil)
		for i := app.FirstModule; i < d.owner.NumModules(); i++ {
			if d.enabledModules[i] {
				ctx.moduleIndex = i
				state := &d.moduleStates[i]
				r := d.renderers[i]
				if state.transitioning {
					if cur.After(state.transEnd) {
						r.FinishTransition(ctx)
						d.numTransitions--
						state.transitioning = false
					} else {
						r.TransitionStep(ctx, cur.Sub(state.transStart))
					}
				}
				r.Render(ctx)
			}
		}
	}
	if popup && d.popupTexture != nil {
		d.Backend.Copy(d.popupTexture, nil, nil)
	}
	d.Backend.Present()
}

func (d *Display) startTransition(moduleIndex app.ModuleIndex) {
	ctx := renderContext{Display: d, moduleIndex: moduleIndex}
	r := d.renderers[moduleIndex]
	state := &d.moduleStates[moduleIndex]
	if state.queuedData == nil {
		panic("Trying to call InitTransition without data")
	}
	if state.transitioning {
		r.FinishTransition(ctx)
		d.numTransitions--
		state.transitioning = false
	}

	transDur := r.InitTransition(ctx, state.queuedData)
	state.queuedData = nil
	if transDur == 0 {
		r.FinishTransition(ctx)
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
			waitTime := (time.Second / 60) - time.Now().Sub(curTime)
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
				case d.Events.SceneChangeID:
					d.enabledModules = d.queuedEnabledModules
					d.queuedEnabledModules = nil
					d.initial = false
					fallthrough
				case d.Events.ModuleConfigID:
					ctx := renderContext{Display: d, heroes: d.owner.ViewHeroes()}
					for i := app.FirstModule; i < d.owner.NumModules(); i++ {
						r := d.renderers[i]
						state := &d.moduleStates[i]
						if state.queuedConfig != nil || state.queuedData != nil {
							ctx.moduleIndex = i
							r.Rebuild(ctx, state.queuedData, state.queuedConfig)
							state.queuedData = nil
							state.queuedConfig = nil
						}
					}
					ctx.heroes.Close()
				case d.Events.HeroesChangedID:
					ctx := renderContext{Display: d, heroes: d.owner.ViewHeroes()}
					for i := app.FirstModule; i < d.owner.NumModules(); i++ {
						state := &d.moduleStates[i]
						if state.queuedData != nil {
							ctx.moduleIndex = i
							r := d.renderers[i]
							r.Rebuild(ctx, state.queuedData, nil)
							state.queuedData = nil
						}
					}
					ctx.heroes.Close()
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

// SendRendererData queues the given data for the module at the given ID as part
// of the request. Whether the data is used for RebuiltState or InitTransition
// depends on the event ID of the request.
func (r *Request) SendRendererData(index app.ModuleIndex, data interface{}) error {
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
	d.Backend.Destroy()
	d.Window.Destroy()
}
