package main

import (
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
			waitTime = time.Now().Sub(curTime) - (time.Second / 60)
		} else {
			waitTime = time.Now().Sub(curTime) - time.Hour
		}
		if waitTime > 0 {
			select {
			case curUpdate := <-ctrl.ModuleUpdate:
				curModule := &screen.modules[curUpdate.index]
				transDur := curModule.module.InitTransition(&screen.SceneCommon)
				if transDur == 0 {
					curModule.module.FinishTransition(&screen.SceneCommon)
				} else if transDur > 0 {
					screen.numTransitions++
					curModule.transStart = time.Now()
					curModule.transEnd = curModule.transStart.Add(transDur)
				}
				break
			case <-ctrl.Exit:
				_ = server.Close()
				egl.DestroySurface(eglState.Display, eglState.Surface)
				egl.DestroyContext(eglState.Display, eglState.Context)
				egl.Terminate(eglState.Display)
				break Outer
			case <-time.After(waitTime):
				break
			}
		}
	}
}
