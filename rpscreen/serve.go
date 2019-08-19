package main

import (
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/flyx/rpscreen/module"
	"github.com/flyx/rpscreen/web"
	"github.com/veandco/go-sdl2/sdl"
)

type uiModuleData struct {
	Name    string
	UI      template.HTML
	Enabled bool
}

type uiSystemData struct {
	Selected bool
}

type uiGroupData struct {
	Selected bool
}

type uiData struct {
	Modules []uiModuleData
	Systems []uiSystemData
	Groups  []uiGroupData
}

type screenHandler struct {
	screen *Screen
	index  *template.Template
	data   uiData
}

func newScreenHandler(screen *Screen) *screenHandler {
	handler := new(screenHandler)
	handler.screen = screen

	raw, err := web.Asset("web/templates/index.html")
	if err != nil {
		panic(err)
	}
	handler.index, err = template.New("index.html").Parse(string(raw))
	if err != nil {
		panic(err)
	}

	handler.data = uiData{Modules: make([]uiModuleData, 0, screen.modules.NumItems()),
		Systems: make([]uiSystemData, 0, screen.Config.NumSystems()),
		Groups:  make([]uiGroupData, 0, screen.Config.NumGroups())}
	for _, item := range screen.modules.items {
		handler.data.Modules = append(handler.data.Modules,
			uiModuleData{Name: item.module.Name(), Enabled: false})
	}
	for i := 0; i < screen.Config.NumSystems(); i++ {
		handler.data.Systems = append(handler.data.Systems,
			uiSystemData{Selected: false})
	}
	for i := 0; i < screen.Config.NumGroups(); i++ {
		handler.data.Groups = append(handler.data.Groups,
			uiGroupData{Selected: false})
	}

	return handler
}

// ServeHTTP implements the HTTP server
func (sh *screenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/index.html" {
		http.NotFound(w, r)
		return
	}

	for index, item := range sh.screen.modules.items {
		sh.data.Modules[index].Enabled = item.enabled
		sh.data.Modules[index].UI = item.module.UI()
	}
	for index := range sh.data.Systems {
		sh.data.Systems[index].Selected = sh.screen.ActiveSystem == index
	}
	for index := range sh.data.Groups {
		sh.data.Groups[index].Selected = sh.screen.ActiveGroup == index
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

func nextPathItem(value string) (string, bool) {
	pos := strings.Index(value, "/")
	if pos == -1 {
		return value, true
	}
	return value[0:pos], false
}

func startServer(screen *Screen) *http.Server {
	server := &http.Server{Addr: ":8080"}

	http.Handle("/", newScreenHandler(screen))
	setupResourceHandler(server, "/css/pure-min.css", "text/css")
	setupResourceHandler(server, "/css/grids-responsive-min.css", "text/css")
	setupResourceHandler(server, "/css/style.css", "text/css")
	setupResourceHandler(server, "/css/fontawesome.min.css", "text/css")
	setupResourceHandler(server, "/css/solid.min.css", "text/css")
	setupResourceHandler(server, "/js/ui.js", "application/javascript")
	setupResourceHandler(server, "/js/sharedData.js", "application/javascript")
	setupResourceHandler(server, "/webfonts/fa-solid-900.eot", "application/vnd.ms-fontobject")
	setupResourceHandler(server, "/webfonts/fa-solid-900.svg", "image/svg+xml")
	setupResourceHandler(server, "/webfonts/fa-solid-900.ttf", "font/ttf")
	setupResourceHandler(server, "/webfonts/fa-solid-900.woff", "font/woff")
	setupResourceHandler(server, "/webfonts/fa-solid-900.woff2", "font/woff2")

	http.HandleFunc("/systems/", func(w http.ResponseWriter, r *http.Request) {
		systemName := r.URL.Path[len("/systems/"):]
		newSystemIndex := -2
		for i := 0; i < screen.Config.NumSystems(); i++ {
			if screen.Config.SystemDirectory(i) == systemName {
				newSystemIndex = i
				break
			}
		}
		if newSystemIndex != -2 {
			if newSystemIndex != screen.ActiveSystem {
				screen.ActiveSystem = newSystemIndex
				sdl.PushEvent(&sdl.UserEvent{Type: screen.systemUpdateEventID})
			}
			module.WriteEndpointHeader(w, module.EndpointReturnRedirect)
		} else {
			http.Error(w, "404: unknown system \""+systemName+"\"", http.StatusNotFound)
		}
	})
	http.HandleFunc("/groups/", func(w http.ResponseWriter, r *http.Request) {
		groupName := r.URL.Path[len("/groups/"):]
		newGroupIndex := -2
		for i := 0; i < screen.Config.NumGroups(); i++ {
			if screen.Config.GroupDirectory(i) == groupName {
				newGroupIndex = i
				break
			}
		}
		if newGroupIndex != -2 {
			if screen.ActiveGroup != newGroupIndex {
				screen.ActiveGroup = newGroupIndex
				sdl.PushEvent(&sdl.UserEvent{Type: screen.groupUpdateEventID, Code: 0})
			}
			module.WriteEndpointHeader(w, module.EndpointReturnRedirect)
		} else {
			http.Error(w, "404: unknown group \""+groupName+"\"", http.StatusNotFound)
		}
	})
	http.HandleFunc("/static.json", func(w http.ResponseWriter, r *http.Request) {
		screen.SendJSON(w, screen.ActiveGroup, screen.ActiveSystem)
	})
	http.HandleFunc("/config/", func(w http.ResponseWriter, r *http.Request) {
		item, isLast := nextPathItem(r.URL.Path[len("/config/"):])
		switch item {
		case "base.json":
			if !isLast {
				http.Error(w, "404: \""+item+"\" not found", http.StatusNotFound)
			} else {
				screen.SendBaseJSON(w)
			}
		case "groups":
			if isLast {
				http.Error(w, "400: group missing", http.StatusBadRequest)
			} else {
				screen.SendGroupJSON(w, r.URL.Path[len("/config/groups/"):])
			}
		case "systems":
			if isLast {
				http.Error(w, "400: group missing", http.StatusBadRequest)
			} else {
				screen.SendSystemJSON(w, r.URL.Path[len("/config/systems/"):])
			}
		default:
			http.Error(w, "404: \""+r.URL.Path+"\" not found", http.StatusNotFound)
		}
	})

	for index, item := range screen.modules.items {
		// needed to avoid closure over loop variable (which doesn't work)
		curIndex := index
		curItem := item
		http.HandleFunc("/"+curItem.module.InternalName()+"/", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				http.Error(w, "400: module endpoints only take POST requests", http.StatusBadRequest)
			} else if !curItem.enabled {
				http.Error(w, "400: module is not enabled", http.StatusBadRequest)
			} else {
				returnPartial := r.PostFormValue("redirect") != "1"
				res := curItem.module.EndpointHandler(r.URL.Path[len(curItem.module.InternalName())+2:],
					r.PostForm, w, returnPartial)
				if res {
					sdl.PushEvent(&sdl.UserEvent{Type: screen.moduleUpdateEventID, Code: int32(curIndex)})
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
