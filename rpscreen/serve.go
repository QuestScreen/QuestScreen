package main

import (
	"github.com/flyx/rpscreen/module"
	"github.com/flyx/rpscreen/web"
	"github.com/veandco/go-sdl2/sdl"
	"html/template"
	"log"
	"net/http"
)

type UIModuleData struct {
	Name    string
	UI      template.HTML
	Enabled bool
}

type UISystemData struct {
	*module.System
	Selected bool
}

type UIGroupData struct {
	*module.Group
	Selected bool
}

type UIData struct {
	Modules []UIModuleData
	Systems []UISystemData
	Groups  []UIGroupData
}

type ScreenHandler struct {
	screen *Screen
	index  *template.Template
	data   UIData
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

	handler.data = UIData{Modules: make([]UIModuleData, 0, len(screen.modules)),
		Systems: make([]UISystemData, 0, len(screen.Systems)),
		Groups:  make([]UIGroupData, 0, len(screen.Groups))}
	for _, mod := range screen.modules {
		handler.data.Modules = append(handler.data.Modules,
			UIModuleData{Name: mod.module.Name(), Enabled: false})
	}
	for index := range screen.Systems {
		handler.data.Systems = append(handler.data.Systems,
			UISystemData{System: &screen.Systems[index], Selected: false})
	}
	for index := range screen.Groups {
		handler.data.Groups = append(handler.data.Groups,
			UIGroupData{Group: &screen.Groups[index], Selected: false})
	}

	return handler
}

func (sh *ScreenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/index.html" {
		http.NotFound(w, r)
		return
	}

	for index, mod := range sh.screen.modules {
		sh.data.Modules[index].Enabled = mod.enabled
		sh.data.Modules[index].UI = mod.module.UI(&sh.screen.SceneCommon)
	}
	for index := range sh.data.Systems {
		sh.data.Systems[index].Selected = sh.screen.ActiveSystem == int32(index)
	}
	for index := range sh.data.Groups {
		sh.data.Groups[index].Selected = sh.screen.ActiveGroup == int32(index)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := sh.index.Execute(w, sh.data); err != nil {
		panic(err)
	}
}

func setupResourceHandler(server *http.Server, path string, contentType string) {
	http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		raw, err := web.Asset("web" + path)
		if err != nil {
			panic(err)
		}
		w.Header().Set("Content-Type", contentType)
		if _, err = w.Write(raw); err != nil {
			panic(err)
		}
	})
}

func startServer(screen *Screen) *http.Server {
	server := &http.Server{Addr: ":8080"}

	http.Handle("/", newScreenHandler(screen))
	setupResourceHandler(server, "/style/pure-min.css", "text/css")
	setupResourceHandler(server, "/style/style.css", "text/css")
	setupResourceHandler(server, "/js/ui.js", "application/javascript")
	setupResourceHandler(server, "/js/sharedData.js", "application/javascript")

	http.HandleFunc("/systems/", func(w http.ResponseWriter, r *http.Request) {
		systemName := r.URL.Path[len("/systems/"):]
		newSystemIndex := int32(-2)
		for index, item := range screen.Systems {
			if item.DirName == systemName {
				newSystemIndex = int32(index)
				break
			}
		}
		if newSystemIndex != -2 {
			if newSystemIndex != screen.ActiveSystem {
				screen.ActiveSystem = newSystemIndex
				sdl.PushEvent(&sdl.UserEvent{Type: screen.systemUpdateEventId})
			}
			module.WriteEndpointHeader(w, module.EndpointReturnRedirect)
		} else {
			http.Error(w, "404: unknown system \""+systemName+"\"", http.StatusNotFound)
		}
	})
	http.HandleFunc("/groups/", func(w http.ResponseWriter, r *http.Request) {
		groupName := r.URL.Path[len("/groups/"):]
		newGroupIndex := int32(-2)
		for index, item := range screen.Groups {
			if item.DirName == groupName {
				newGroupIndex = int32(index)
				break
			}
		}
		if newGroupIndex != -2 {
			if screen.ActiveGroup != newGroupIndex {
				screen.ActiveGroup = newGroupIndex
				sdl.PushEvent(&sdl.UserEvent{Type: screen.groupUpdateEventId, Code: 0})
			}
			module.WriteEndpointHeader(w, module.EndpointReturnRedirect)
		} else {
			http.Error(w, "404: unknown group \""+groupName+"\"", http.StatusNotFound)
		}
	})
	http.HandleFunc("/static.json", func(w http.ResponseWriter, r *http.Request) {
		screen.SendJson(w)
	})

	for index, item := range screen.modules {
		// needed to avoid closure over loop variable (which doesn't work)
		curIndex := index
		curItem := item
		http.HandleFunc("/"+curItem.module.InternalName()+"/", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				http.Error(w, "400: module endpoints only take POST requests", http.StatusBadRequest)
			} else {
				returnPartial := r.PostFormValue("redirect") != "1"
				res := curItem.module.EndpointHandler(r.URL.Path[len(curItem.module.InternalName())+2:],
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
