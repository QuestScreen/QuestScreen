package main

import (
	"github.com/flyx/rpscreen/web"
	"github.com/veandco/go-sdl2/sdl"
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

	raw, err := web.Asset("web/templates/index.html")
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
	Modules []UIModuleData
}

func (me *ScreenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/index.html" {
		http.NotFound(w, r)
		return
	}

	data := UIData{Modules: make([]UIModuleData, 0, len(me.screen.modules))}
	for _, module := range me.screen.modules {
		if module.enabled {
			data.Modules = append(data.Modules, UIModuleData{Name: module.module.Name(), UI: module.module.UI()})
		}
	}
	w.Header().Set("Content-Type", "text/html")
	if err := me.index.Execute(w, data); err != nil {
		panic(err)
	}
}

func startServer(screen *Screen) *http.Server {
	server := &http.Server{Addr: ":8080"}

	http.Handle("/", newScreenHandler(screen))
	http.HandleFunc("/style/pure-min.css", func(w http.ResponseWriter, r *http.Request) {
		raw, err := web.Asset("web/style/pure-min.css")
		if err != nil {
			panic(err)
		}
		w.Header().Set("Content-Type", "text/css")
		if _, err = w.Write(raw); err != nil {
			panic(err)
		}
	})

	for index, item := range screen.modules {
		// needed to avoid closure over loop variable (which doesn't work)
		curIndex := index
		curItem := item
		http.HandleFunc(item.module.EndpointPath(), func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				http.Error(w, "400: module endpoints only take POST requests", http.StatusBadRequest)
			} else {
				returnPartial := r.PostFormValue("redirect") != "1"
				res := curItem.module.EndpointHandler(r.URL.Path[len(curItem.module.EndpointPath()):],
					r.PostForm, w, returnPartial)
				if res {
					sdl.PushEvent(&sdl.UserEvent{Type: screen.moduleUpdateEventId, Code: int32(curIndex)})
				}
			}
		})
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			if err != http.ErrServerClosed {
				log.Fatal(err)
			}
		}
	}()

	return server
}
