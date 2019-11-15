package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/flyx/pnpscreen/api"

	"github.com/flyx/pnpscreen/display"

	"github.com/flyx/pnpscreen/web"
	"github.com/veandco/go-sdl2/sdl"
)

type screenHandler struct {
	owner  *app
	events display.Events
	index  []byte
}

func newScreenHandler(owner *app, events display.Events) *screenHandler {
	handler := new(screenHandler)
	handler.owner = owner
	handler.events = events

	var err error
	handler.index, err = web.Asset("web/templates/index.html")
	if err != nil {
		panic(err)
	}

	return handler
}

// ServeHTTP implements the HTTP server
func (sh *screenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/index.html" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(sh.index)
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

func (sh *screenHandler) mergeAndSendConfigs(moduleConfigChan chan<- display.ModuleConfigUpdate) {
	for i := api.ModuleIndex(0); i < api.ModuleIndex(len(sh.owner.modules)); i++ {
		moduleConfigChan <- display.ModuleConfigUpdate{Index: i,
			Config: sh.owner.config.MergeConfig(i,
				sh.owner.activeSystem, sh.owner.activeGroup)}
	}
	sdl.PushEvent(&sdl.UserEvent{Type: sh.events.ModuleConfigID})
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

	handler := newScreenHandler(owner, events)
	http.Handle("/", handler)
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
				handler.mergeAndSendConfigs(moduleConfigChan)
				ret, err := owner.config.BuildStateJSON()
				sendJSON(w, ret, err)
			}
		} else {
			http.Error(w, "404: unknown group \""+groupName+"\"", http.StatusNotFound)
		}
	})
	http.HandleFunc("/static.json", func(w http.ResponseWriter, r *http.Request) {
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
						handler.mergeAndSendConfigs(moduleConfigChan)
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
						handler.mergeAndSendConfigs(moduleConfigChan)
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
