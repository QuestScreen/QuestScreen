package main

import (
	"github.com/flyx/egl"
	"github.com/flyx/egl/platform"
	"github.com/flyx/rpscreen/module"
	"github.com/flyx/rpscreen/module/background"
	gl "github.com/remogatto/opengles2"
	"os"
	"os/user"
	"time"
)

const TexCoordMax = 1

type moduleListItem struct {
	module        module.Module
	enabled       bool
	transStart    time.Time
	transEnd      time.Time
	transitioning bool
}

type Screen struct {
	module.SceneCommon
	textureBuffer  uint32
	modules        []moduleListItem
	ctrl           *controlCh
	numTransitions int32
}

func newScreen(eglState *platform.EGLState, ctrl *controlCh) (*Screen, error) {
	if ok := egl.MakeCurrent(eglState.Display, eglState.Surface, eglState.Surface, eglState.Context); !ok {
		return nil, egl.NewError(egl.GetError())
	}

	usr, _ := user.Current()

	screen := new(Screen)
	screen.modules = make([]moduleListItem, 0, 16)
	screen.ctrl = ctrl
	screen.DataDir = usr.HomeDir + "/.local/share/rpscreen"
	if err := os.MkdirAll(screen.DataDir, 0700); err != nil {
		panic(err)
	}
	screen.numTransitions = 0

	var width, height int32
	egl.QuerySurface(eglState.Display, eglState.Surface, egl.WIDTH, &width)
	egl.QuerySurface(eglState.Display, eglState.Surface, egl.HEIGHT, &height)
	screen.Ratio = float32(width) / float32(height)

	screen.Square.Vertices = module.CreateFloatBuffer([]float32{
		1, -1, 1, 1, TexCoordMax, TexCoordMax,
		1, 1, 1, 1, TexCoordMax, 0,
		-1, 1, 1, 1, 0, 0,
		-1, -1, 1, 1, 0, TexCoordMax,
	})
	screen.Square.Indices = module.CreateByteBuffer([]byte{
		0, 1, 2,
		2, 3, 0,
	})

	bg := new(background.Background)
	if err := bg.Init(&screen.SceneCommon); err != nil {
		panic(err)
	}
	screen.modules = append(screen.modules, moduleListItem{module: bg, enabled: true, transitioning: false})
	return screen, nil
}

func (s *Screen) Render(cur time.Time) {
	gl.ClearColor(0, 0, 0, 1)
	gl.Clear(gl.DEPTH_BUFFER_BIT | gl.COLOR_BUFFER_BIT)

	for _, item := range s.modules {
		if item.enabled {
			if item.transitioning {
				if cur.After(item.transEnd) {
					item.module.FinishTransition(&s.SceneCommon)
					s.numTransitions--
				} else {
					item.module.TransitionStep(&s.SceneCommon, cur.Sub(item.transStart))
				}
			}
			item.module.Render(&s.SceneCommon)
		}
	}
	gl.Flush()
	gl.Finish()
}
