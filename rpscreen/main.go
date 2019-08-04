package main

import (
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
	"log"
	"runtime"
	"time"
)

func init() {
	runtime.LockOSThread()
}

func startTransition(m *moduleListItem, screen *Screen) {
	transDur := m.module.InitTransition(&screen.SceneCommon)
	if transDur == 0 {
		m.module.FinishTransition(&screen.SceneCommon)
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

	server := startServer(screen)

	var render = true
	var animationStart = time.Now()
	var frameCount = time.Duration(0)
Outer:
	for {
		curTime := time.Now()
		if render {
			frameCount++
			screen.Render(curTime)
		}
		var event sdl.Event
		if screen.numTransitions > 0 {
			waitTime := (time.Second / 30) - time.Now().Sub(curTime)
			if waitTime > 0 {
				event = sdl.WaitEventTimeout(int(waitTime / time.Millisecond))
			}
		} else {
			if render {
				log.Printf("animation finished; FPS: %d\n", time.Now().Sub(animationStart)*frameCount/time.Second)
			}
			render = false
			event = sdl.WaitEvent()
		}
		inRender := render
		for ; event != nil; event = sdl.PollEvent() {
			switch e := event.(type) {
			case *sdl.QuitEvent:
				break Outer
			case *sdl.UserEvent:
				switch e.Type {
				case screen.moduleUpdateEventId:
					startTransition(&screen.modules[e.Code], screen)
					render = true
				case screen.systemUpdateEventId:
					for i := range screen.modules {
						if screen.modules[i].module.SystemChanged(&screen.SceneCommon) {
							startTransition(&screen.modules[i], screen)
							render = true
						}
					}
				case screen.groupUpdateEventId:
					for i := range screen.modules {
						if screen.modules[i].module.GroupChanged(&screen.SceneCommon) {
							startTransition(&screen.modules[i], screen)
							render = true
						}
					}
				}
			}
		}
		if render && !inRender {
			animationStart = time.Now()
			frameCount = 0
		}
	}
	_ = server.Close()
	screen.Renderer.Destroy()
	screen.Window.Destroy()
}
