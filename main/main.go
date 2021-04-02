package main

import (
	"os"
	"runtime"

	"github.com/pborman/getopt/v2"

	"github.com/QuestScreen/QuestScreen/display"
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	fullscreenFlag := getopt.BoolLong("fullscreen", 'f', "start in fullscreen")
	port := getopt.Uint16Long("port", 'p', 0, "port to bind to")
	width := getopt.Int32Long("width", 'w', 0, "width of the window (set w and h to start windowed)")
	height := getopt.Int32Long("height", 'h', 0, "height of the window (set w and h to start windowed)")
	debug := getopt.BoolLong("debug", 'd', "use an OpenGL debug context")
	getopt.Parse()

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

	events := display.GenEvents()
	var qs QuestScreen
	qs.Init(*fullscreenFlag, *width, *height, events, *port, *debug)

	server := startServer(&qs, events, qs.appConfig.port)

	ret := qs.display.RenderLoop()
	_ = server.Close()
	qs.destroy()
	os.Exit(ret)
}
