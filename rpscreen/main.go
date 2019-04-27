package main

import (
	"fmt"
	"github.com/flyx/egl"
	"runtime"
	"time"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	ctrl := newControlCh()
	eglState := InitEGL(ctrl, 800, 600)

	screen, err := newScreen(eglState, ctrl)
	if err != nil {
		panic(err)
	}

	server := startServer(screen)

Outer:
	for {
		curTime := time.Now()
		screen.Render(curTime)
		egl.SwapBuffers(eglState.Display, eglState.Surface)
		var waitTime time.Duration
		if screen.numTransitions > 0 {
			waitTime = (time.Second / 30) - time.Now().Sub(curTime)
		} else {
			waitTime = time.Hour - time.Now().Sub(curTime)
		}
		if waitTime > 0 {
			select {
			case curUpdate := <-ctrl.ModuleUpdate:
				fmt.Println("updating module", screen.modules[curUpdate.index].module.Name())
				curModule := &screen.modules[curUpdate.index]
				transDur := curModule.module.InitTransition(&screen.SceneCommon)
				if transDur == 0 {
					curModule.module.FinishTransition(&screen.SceneCommon)
				} else if transDur > 0 {
					screen.numTransitions++
					curModule.transStart = time.Now()
					curModule.transEnd = curModule.transStart.Add(transDur)
					curModule.transitioning = true
				}
				break
			case <-ctrl.Exit:
				_ = server.Close()
				egl.DestroySurface(eglState.Display, eglState.Surface)
				egl.DestroyContext(eglState.Display, eglState.Context)
				egl.Terminate(eglState.Display)
				break Outer
			case <-time.After(waitTime):
				fmt.Println("wake after timeout of", waitTime)
				break
			}
		}
	}
}
