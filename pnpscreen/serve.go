package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/flyx/pnpscreen/api"
	"github.com/flyx/pnpscreen/web"

	"github.com/flyx/pnpscreen/display"

	"github.com/veandco/go-sdl2/sdl"
)

type staticResource struct {
	contentType string
	content     []byte
}

type staticResourceHandler struct {
	owner     *app
	events    display.Events
	resources map[string]staticResource
}

func newStaticResourceHandler(owner *app, events display.Events) *staticResourceHandler {
	handler := new(staticResourceHandler)
	handler.owner = owner
	handler.events = events
	handler.resources = make(map[string]staticResource)
	indexRes := staticResource{
		contentType: "text/html; charset=utf-8", content: owner.html}
	handler.resources["/"] = indexRes
	handler.resources["/index.html"] = indexRes
	handler.resources["/all.js"] = staticResource{
		contentType: "application/javascript", content: owner.js}
	handler.resources["/style.css"] = staticResource{
		contentType: "text/css", content: owner.css}
	return handler
}

// ServeHTTP implements the HTTP server
func (sh *staticResourceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Clacks-Overhead", "GNU Terry Pratchett")
	res, ok := sh.resources[r.URL.Path]
	if ok {
		w.Header().Set("Content-Type", res.contentType)
		w.Write(res.content)
	} else {
		http.NotFound(w, r)
	}
}

func (sh *staticResourceHandler) add(path string, contentType string) {
	sh.resources[path] = staticResource{
		contentType: contentType, content: web.MustAsset("web" + path)}
}

func nextPathItem(value string) (string, bool) {
	pos := strings.Index(value, "/")
	if pos == -1 {
		return value, true
	}
	return value[0:pos], false
}

func mergeAndSendConfigs(a *app, eventID uint32,
	moduleConfigChan chan<- display.ModuleConfigUpdate) {
	for i := api.ModuleIndex(0); i < api.ModuleIndex(len(a.modules)); i++ {
		moduleConfigChan <- display.ModuleConfigUpdate{Index: i,
			Config: a.config.MergeConfig(i,
				a.activeSystem, a.activeGroup)}
	}
	sdl.PushEvent(&sdl.UserEvent{Type: eventID})
}

func sendJSON(w http.ResponseWriter, content []byte, err error) {
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	} else {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func startServer(owner *app,
	moduleConfigChan chan<- display.ModuleConfigUpdate, events display.Events,
	port uint16) *http.Server {
	server := &http.Server{Addr: ":" + strconv.Itoa(int(port))}

	handler := newStaticResourceHandler(owner, events)
	handler.add("/css/pure-min.css", "text/css")
	handler.add("/css/grids-responsive-min.css", "text/css")
	handler.add("/css/fontawesome.min.css", "text/css")
	handler.add("/css/solid.min.css", "text/css")
	handler.add("/js/ui.js", "application/javascript")
	handler.add("/webfonts/fa-solid-900.eot", "application/vnd.ms-fontobject")
	handler.add("/webfonts/fa-solid-900.svg", "image/svg+xml")
	handler.add("/webfonts/fa-solid-900.ttf", "font/ttf")
	handler.add("/webfonts/fa-solid-900.woff", "font/woff")
	handler.add("/webfonts/fa-solid-900.woff2", "font/woff2")
	http.Handle("/", handler)

	http.HandleFunc("/groups/", func(w http.ResponseWriter, r *http.Request) {
		groupName := r.URL.Path[len("/groups/"):]
		newGroupIndex := -2
		for i := 0; i < owner.config.NumGroups(); i++ {
			if owner.config.GroupID(i) == groupName {
				newGroupIndex = i
				break
			}
		}
		if newGroupIndex != -2 {
			if err := owner.setActiveGroup(newGroupIndex); err != nil {
				http.Error(w, "400: Could not set group: "+err.Error(),
					http.StatusBadRequest)
			} else {
				mergeAndSendConfigs(owner, events.ModuleConfigID, moduleConfigChan)
				ret, err := owner.config.BuildStateJSON()
				sendJSON(w, ret, err)
			}
		} else {
			http.Error(w, "404: unknown group \""+groupName+"\"", http.StatusNotFound)
		}
	})
	http.HandleFunc("/app", func(w http.ResponseWriter, r *http.Request) {
		ret, err := owner.config.BuildGlobalJSON(owner, owner.activeGroup)
		sendJSON(w, ret, err)
	})
	http.HandleFunc("/config/", func(w http.ResponseWriter, r *http.Request) {
		post := false
		var raw []byte
		switch r.Method {
		case "GET":
			break
		case "POST":
			post = true
			var err error
			raw, err = ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			break
		default:
			http.Error(w, "405: Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		item, isLast := nextPathItem(r.URL.Path[len("/config/"):])
		switch item {
		case "base.json":
			if !isLast {
				http.Error(w, "404: \""+r.URL.Path+"\" not found", http.StatusNotFound)
			} else {
				if post {
					if err := owner.config.LoadBaseJSON(raw); err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
					} else {
						w.WriteHeader(http.StatusNoContent)
					}
				} else {
					var err error
					raw, err := owner.config.BuildBaseJSON()
					sendJSON(w, raw, err)
				}
			}
		case "groups":
			if isLast {
				http.Error(w, "400: group missing", http.StatusBadRequest)
			} else {
				groupName := r.URL.Path[len("/config/groups/"):]
				if post {
					if err := owner.config.LoadGroupJSON(raw, groupName); err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
					} else {
						w.WriteHeader(http.StatusNoContent)
						mergeAndSendConfigs(owner, events.ModuleConfigID, moduleConfigChan)
					}
				} else {
					var err error
					raw, err = owner.config.BuildGroupJSON(groupName)
					sendJSON(w, raw, err)
				}
			}
		case "systems":
			if isLast {
				http.Error(w, "400: system missing", http.StatusBadRequest)
			} else {
				systemName := r.URL.Path[len("/config/systems/"):]
				if post {
					if err := owner.config.LoadSystemJSON(raw, systemName); err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
					} else {
						w.WriteHeader(http.StatusNoContent)
						mergeAndSendConfigs(owner, events.ModuleConfigID, moduleConfigChan)
					}
				} else {
					var err error
					raw, err = owner.config.BuildSystemJSON(systemName)
					sendJSON(w, raw, err)
				}
			}
		default:
			http.Error(w, "404: \""+r.URL.Path+"\" not found", http.StatusNotFound)
		}
	})

	for i := range owner.modules {
		// needed to avoid closure over loop variable (which doesn't work)
		curModuleIndex := int32(i)
		module := owner.modules[i].module
		actions := module.State().Actions()
		for j := range actions {
			curActionIndex := j
			http.HandleFunc("/module/"+module.ID()+"/"+actions[j],
				func(w http.ResponseWriter, r *http.Request) {
					if r.Method != "POST" {
						http.Error(w, "400: module endpoints only take POST requests",
							http.StatusBadRequest)
					} else {
						payload, _ := ioutil.ReadAll(r.Body)
						response, err := module.State().HandleAction(
							curActionIndex, payload)
						if err != nil {
							http.Error(w, "400: "+err.Error(), http.StatusBadRequest)
						} else {
							sdl.PushEvent(&sdl.UserEvent{
								Type: events.ModuleUpdateID, Code: curModuleIndex})
							if response == nil {
								w.WriteHeader(http.StatusNoContent)
							} else {
								w.Header().Set("Content-Type", "application/json")
								w.WriteHeader(http.StatusOK)
								w.Write(response)
							}
							newStateYaml, err := owner.config.BuildStateYaml()
							if err != nil {
								panic(err)
							}
							go func(content []byte, filename string) {
								ioutil.WriteFile(filename, content, 0644)
							}(newStateYaml, owner.pathToState())
						}
					}
				})
		}
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
