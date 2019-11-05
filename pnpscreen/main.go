package main

import (
	"log"
	"runtime"

	"github.com/flyx/pnpscreen/data"

	"github.com/pborman/getopt"

	"github.com/flyx/pnpscreen/display"
	"github.com/flyx/pnpscreen/modules/background"
	"github.com/flyx/pnpscreen/modules/herolist"
	"github.com/flyx/pnpscreen/modules/persons"
	"github.com/flyx/pnpscreen/modules/title"
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
	d, err := display.NewDisplay(events, *fullscreenFlag, *port)
	if err != nil {
		panic(err)
	}
	if err := sdl.GLSetSwapInterval(-1); err != nil {
		log.Println("Could not set swap interval to -1")
	}
	width, height, _ := d.Renderer.GetOutputSize()

	d.RegisterModule(new(background.Background))
	d.RegisterModule(new(herolist.HeroList))
	d.RegisterModule(new(persons.Persons))
	d.RegisterModule(new(title.Title))
	var store data.Store
	store.Init(d.ConfigurableItems(), width, height)
	d.InitModuleConfigs(&store)

	items := d.ConfigurableItems()
	itemConfigChan := make(chan display.ItemConfigUpdate, items.NumItems())
	server := startServer(&store, items, itemConfigChan, events, *port)

	d.RenderLoop(itemConfigChan)
	_ = server.Close()
	d.Renderer.Destroy()
	d.Window.Destroy()
}
