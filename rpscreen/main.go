package main

import (
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
	"runtime"
	"time"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		panic(err)
	}
	defer sdl.Quit()
	img.Init(img.INIT_PNG | img.INIT_JPG)
	defer img.Quit()
	screen, err := newScreen()
	if err != nil {
		panic(err)
	}

	server := startServer(screen)

	var render = true
Outer:
	for {
		curTime := time.Now()
		if render {
			screen.Render(curTime)
		}
		var event sdl.Event
		if screen.numTransitions > 0 {
			waitTime := (time.Second / 30) - time.Now().Sub(curTime)
			event = sdl.WaitEventTimeout(int(waitTime / time.Millisecond))
		} else {
			render = false
			event = sdl.WaitEvent()
		}
		for ; event != nil; event = sdl.PollEvent() {
			switch e := event.(type) {
			case *sdl.QuitEvent:
				break Outer
			case *sdl.UserEvent:
				switch e.Type {
				case screen.moduleUpdateEventId:
					curModule := &screen.modules[e.Code]
					transDur := curModule.module.InitTransition(&screen.SceneCommon)
					if transDur == 0 {
						curModule.module.FinishTransition(&screen.SceneCommon)
					} else if transDur > 0 {
						screen.numTransitions++
						curModule.transStart = time.Now()
						curModule.transEnd = curModule.transStart.Add(transDur)
						curModule.transitioning = true
					}
					render = true
				}
			}
		}
	}
	_ = server.Close()
	screen.Renderer.Destroy()
	screen.Window.Destroy()
}
