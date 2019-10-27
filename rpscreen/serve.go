package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/flyx/rpscreen/data"
	"github.com/flyx/rpscreen/display"

	"github.com/flyx/rpscreen/web"
	"github.com/veandco/go-sdl2/sdl"
)

type screenHandler struct {
	store  *data.Store
	items  data.ConfigurableItemProvider
	events display.Events
	index  []byte
}

func newScreenHandler(store *data.Store, items data.ConfigurableItemProvider, events display.Events) *screenHandler {
	handler := new(screenHandler)
	handler.store = store
	handler.items = items
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

func (sh *screenHandler) mergeAndSendConfigs(moduleConfigChan chan<- display.ItemConfigUpdate) {
	for i := 0; i < sh.items.NumItems(); i++ {
		moduleConfigChan <- display.ItemConfigUpdate{ItemIndex: i,
			Config: sh.store.Config.MergeConfig(&sh.store.StaticData, i,
				sh.store.GetActiveSystem(), sh.store.GetActiveGroup())}
	}
	sdl.PushEvent(&sdl.UserEvent{Type: sh.events.ModuleConfigID})
}

func startServer(store *data.Store, items data.ConfigurableItemProvider,
	itemConfigChan chan<- display.ItemConfigUpdate, events display.Events) *http.Server {
	server := &http.Server{Addr: ":8080"}

	handler := newScreenHandler(store, items, events)
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
		for i := 0; i < store.Config.NumGroups(); i++ {
			if store.Config.GroupDirectory(i) == groupName {
				newGroupIndex = i
				break
			}
		}
		if newGroupIndex != -2 {
			if err := store.SetActiveGroup(newGroupIndex); err != nil {
				http.Error(w, "400: Could not set group: "+err.Error(),
					http.StatusBadRequest)
			} else {
				sdl.PushEvent(&sdl.UserEvent{Type: events.GroupChangeID})
				store.SendStateJSON(w)
			}
		} else {
			http.Error(w, "404: unknown group \""+groupName+"\"", http.StatusNotFound)
		}
	})
	http.HandleFunc("/static.json", func(w http.ResponseWriter, r *http.Request) {
		store.SendGlobalJSON(w)
	})
	http.HandleFunc("/config/", func(w http.ResponseWriter, r *http.Request) {
		post := false
		switch r.Method {
		case "GET":
			break
		case "POST":
			post = true
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
					store.ReceiveBaseJSON(w, r.Body)
				} else {
					store.SendBaseJSON(w)
				}
			}
		case "groups":
			if isLast {
				http.Error(w, "400: group missing", http.StatusBadRequest)
			} else {
				groupName := r.URL.Path[len("/config/groups/"):]
				if post {
					if store.ReceiveGroupJSON(w, groupName, r.Body) {
						handler.mergeAndSendConfigs(itemConfigChan)
					}
				} else {
					store.SendGroupJSON(w, groupName)
				}
			}
		case "systems":
			if isLast {
				http.Error(w, "400: group missing", http.StatusBadRequest)
			} else {
				systemName := r.URL.Path[len("/config/systems/"):]
				if post {
					if store.ReceiveSystemJSON(w, systemName, r.Body) {
						handler.mergeAndSendConfigs(itemConfigChan)
					}
				} else {
					store.SendSystemJSON(w, systemName)
				}
			}
		default:
			http.Error(w, "404: \""+r.URL.Path+"\" not found", http.StatusNotFound)
		}
	})

	for i := 0; i < items.NumItems(); i++ {
		// needed to avoid closure over loop variable (which doesn't work)
		curModuleIndex := int32(i)
		curItem := items.ItemAt(i)
		actions := curItem.GetState().Actions()
		for j := range actions {
			curActionIndex := j
			http.HandleFunc("/module/"+curItem.InternalName()+"/"+actions[j],
				func(w http.ResponseWriter, r *http.Request) {
					if r.Method != "POST" {
						http.Error(w, "400: module endpoints only take POST requests",
							http.StatusBadRequest)
					} else {
						payload, _ := ioutil.ReadAll(r.Body)
						if err := curItem.GetState().HandleAction(curActionIndex, payload,
							store); err != nil {
							http.Error(w, "400: "+err.Error(), http.StatusBadRequest)
						} else {
							sdl.PushEvent(&sdl.UserEvent{
								Type: events.ModuleUpdateID, Code: curModuleIndex})
							w.WriteHeader(http.StatusNoContent)
							newStateYaml := store.GenGroupStateYaml()
							go func(content []byte, filename string) {
								ioutil.WriteFile(filename, content, 0644)
							}(newStateYaml, store.PathToState())
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
