package main

import (
	"log"
	"runtime"

	"github.com/pborman/getopt"

	"github.com/flyx/pnpscreen/display"
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	fullscreenFlag := getopt.BoolLong("fullscreen", 'f', "start in fullscreen")
	port := getopt.Uint16Long("port", 'p', 8080, "port to bind to")
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
	var a app
	a.Init(*fullscreenFlag, events, *port)
	if err := sdl.GLSetSwapInterval(-1); err != nil {
		log.Println("Could not set swap interval to -1")
	}

	server := startServer(&a, events, *port)

	a.display.RenderLoop()
	_ = server.Close()
	a.destroy()
}
