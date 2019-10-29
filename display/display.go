package display

import (
	"log"
	"time"

	"github.com/flyx/rpscreen/data"
	"github.com/veandco/go-sdl2/sdl"
)

// ItemConfigUpdate is a message requesting to configure the item with the given
// index with the given config.
type ItemConfigUpdate struct {
	ItemIndex int
	Config    interface{}
}

type moduleListItem struct {
	module        Module
	enabled       bool
	transStart    time.Time
	transEnd      time.Time
	transitioning bool
}

type moduleList struct {
	items []moduleListItem
}

func (ml *moduleList) NumItems() int {
	return len(ml.items)
}

func (ml *moduleList) ItemAt(index int) data.ConfigurableItem {
	return ml.items[index].module
}

// Display describes a display rendering scenes.
type Display struct {
	data.StaticData
	Events
	Renderer       *sdl.Renderer
	Window         *sdl.Window
	textureBuffer  uint32
	modules        moduleList
	numTransitions int32
	popupTexture   *sdl.Texture
}

// RegisterModule registers the given, uninitialized module with the display.
func (d *Display) RegisterModule(module Module) {
	d.modules.items = append(d.modules.items, moduleListItem{
		module: module, enabled: false})
}

// InitModuleConfigs will initialize configuration of all modules.
// This will call Init on each module.
func (d *Display) InitModuleConfigs(store *data.Store) {
	for i := range d.modules.items {
		module := d.modules.items[i].module
		if err := module.Init(d, store); err != nil {
			log.Printf("Unable to initialize module %s: %s", module.Name(), err)
			continue
		}
		module.SetConfig(
			store.Config.MergeConfig(&store.StaticData, i, -1, -1))
		d.modules.items[i].enabled = true
	}
}

// ConfigurableItems returns the list of configurable items (i.e. modules)
func (d *Display) ConfigurableItems() data.ConfigurableItemProvider {
	return &d.modules
}

// NewDisplay creates a new display.
func NewDisplay(events Events) (*Display, error) {
	display := new(Display)
	var err error
	display.Events = events
	display.Window, err = sdl.CreateWindow("rpscreen", sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED, 800, 600,
		sdl.WINDOW_OPENGL|sdl.WINDOW_ALLOW_HIGHDPI)
	if err != nil {
		return nil, err
	}

	display.Renderer, err = sdl.CreateRenderer(display.Window, -1,
		sdl.RENDERER_ACCELERATED|sdl.RENDERER_TARGETTEXTURE)
	if err != nil {
		display.Window.Destroy()
		return nil, err
	}
	width, height, err := display.Renderer.GetOutputSize()
	if err != nil {
		display.Window.Destroy()
		return nil, err
	}

	display.modules = moduleList{items: make([]moduleListItem, 0, 16)}
	display.StaticData.Init(width, height, &display.modules)
	display.numTransitions = 0

	display.genPopup(width, height)

	return display, nil
}

func (d *Display) render(cur time.Time, popup bool) {
	d.Renderer.Clear()
	d.Renderer.SetDrawColor(255, 255, 255, 255)
	d.Renderer.FillRect(nil)
	for i := 0; i < len(d.modules.items); i++ {
		item := &d.modules.items[i]
		if item.enabled {
			if item.transitioning {
				if cur.After(item.transEnd) {
					item.module.FinishTransition()
					d.numTransitions--
					item.transitioning = false
				} else {
					item.module.TransitionStep(cur.Sub(item.transStart))
				}
			}
			item.module.Render()
		}
	}
	if popup {
		d.Renderer.Copy(d.popupTexture, nil, nil)
	}
	d.Renderer.Present()
}

func (d *Display) startTransition(m *moduleListItem) {
	transDur := m.module.InitTransition()
	if transDur == 0 {
		m.module.FinishTransition()
	} else if transDur > 0 {
		d.numTransitions++
		m.transStart = time.Now()
		m.transEnd = m.transStart.Add(transDur)
		m.transitioning = true
	}
}

// RenderLoop implements the rendering loop for the display.
// This function MUST be called in the main thread
func (d *Display) RenderLoop(itemConfigChan chan ItemConfigUpdate) {
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
					d.startTransition(&d.modules.items[e.Code])
					render = true
				case d.Events.ModuleConfigID:
				outer:
					for {
						select {
						case data := <-itemConfigChan:
							item := &d.modules.items[data.ItemIndex]
							if item.module.SetConfig(data.Config) {
								item.module.RebuildState()
								render = true
							}
						default:
							break outer
						}
					}
				case d.Events.GroupChangeID:
					for i := range d.modules.items {
						d.modules.items[i].module.RebuildState()
					}
					render = true
				}
			}
		}
		if render && !inRender {
			start = time.Now()
			frameCount = 0
		}
	}
}
