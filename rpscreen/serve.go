package main

import (
	"github.com/flyx/rpscreen/web"
	"html/template"
	"log"
	"net/http"
)

type ScreenHandler struct {
	screen *Screen
	index  *template.Template
}

func newScreenHandler(screen *Screen) *ScreenHandler {
	handler := new(ScreenHandler)
	handler.screen = screen

	raw, err := web.Asset("index.html")
	if err != nil {
		panic(err)
	}
	handler.index, err = template.New("index.html").Parse(string(raw))
	if err != nil {
		panic(err)
	}
	return handler
}

type UIModuleData struct {
	Name string
	UI   template.HTML
}

type UIData struct {
	modules []UIModuleData
}

func (me *ScreenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := UIData{modules: make([]UIModuleData, len(me.screen.modules))}
	i := 0
	for _, module := range me.screen.modules {
		if module.enabled {
			data.modules[i] = UIModuleData{Name: module.module.Name(), UI: module.module.UI()}
		}
	}
	if err := me.index.Execute(w, data); err != nil {
		panic(err)
	}
}

func startServer(screen *Screen) *http.Server {
	server := &http.Server{Addr: ":8080"}

	http.Handle("/", newScreenHandler(screen))

	go func() {
		if err := server.ListenAndServe(); err != nil {
			if err != http.ErrServerClosed {
				log.Fatal(err)
			}
		}
	}()

	return server
}
