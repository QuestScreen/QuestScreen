package main

import (
	"log"
	"runtime"
	"time"

	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

func init() {
	runtime.LockOSThread()
}

func startTransition(m *moduleListItem, screen *Screen) {
	transDur := m.module.InitTransition()
	if transDur == 0 {
		m.module.FinishTransition()
	} else if transDur > 0 {
		screen.numTransitions++
		m.transStart = time.Now()
		m.transEnd = m.transStart.Add(transDur)
		m.transitioning = true
	}
}

func main() {
	if err := sdl.Init(sdl.INIT_VIDEO | sdl.INIT_EVENTS); err != nil {
		panic(err)
	}
	defer sdl.Quit()
	if err := ttf.Init(); err != nil {
		panic(err)
	}
	defer ttf.Quit()
	img.Init(img.INIT_PNG | img.INIT_JPG)
	defer img.Quit()
	screen, err := newScreen()
	if err != nil {
		panic(err)
	}
	if err := sdl.GLSetSwapInterval(-1); err != nil {
		log.Println("Could not set swap interval to -1")
	}

	moduleConfigChan := make(chan moduleConfigUpdate, screen.modules.NumItems())
	server := startServer(screen, moduleConfigChan)

	render := true
	popup := false

	var start = time.Now()
	var frameCount = int64(0)
Outer:
	for {
		curTime := time.Now()
		if render {
			frameCount++
			screen.Render(curTime, popup)
			if curTime.Sub(start) >= time.Second {
				log.Printf("FPS: %d\n", frameCount)
				start = curTime
				frameCount = 0
			}
		}
		var event sdl.Event
		if screen.numTransitions > 0 {
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
							break Outer
						case sdl.K_s:
							break Outer
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
				break Outer
			case *sdl.UserEvent:
				switch e.Type {
				case screen.moduleUpdateEventID:
					startTransition(&screen.modules.items[e.Code], screen)
					render = true
				case screen.moduleConfigEventID:
				outer:
					for {
						select {
						case data := <-moduleConfigChan:
							item := &screen.modules.items[data.moduleIndex]
							item.module.SetConfig(data.config)
							if item.module.NeedsTransition() {
								startTransition(item, screen)
								render = true
							}
						default:
							break outer
						}
					}
				}
			}
		}
		if render && !inRender {
			start = time.Now()
			frameCount = 0
		}
	}
	_ = server.Close()
	screen.Renderer.Destroy()
	screen.Window.Destroy()
}
