package main

import (
	"encoding/json"
	"fmt"
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

type httpMethods int

const (
	httpGet httpMethods = 1 << iota
	httpPost
	httpUnknown
)

type subjectProvider int

const (
	noSubject subjectProvider = iota
	jsonSubject
	pathSubject
)

func parseMethod(raw string) httpMethods {
	switch raw {
	case "GET":
		return httpGet
	case "POST":
		return httpPost
	default:
		return httpUnknown
	}
}

// String returns the method name for a single method and "[ <name1>... ]" for
// multiple methods.
func (m httpMethods) String() string {
	switch m {
	case httpGet:
		return "GET"
	case httpPost:
		return "POST"
	case httpUnknown:
		return "UNKNOWN"
	default:
		var sb strings.Builder
		sb.WriteByte('[')
		for c := httpMethods(1); c < httpUnknown; c = c << 1 {
			sb.WriteByte(' ')
			sb.WriteString(c.String())
		}
		sb.WriteString(" ]")
		return sb.String()
	}
}

type endpointEnv struct {
	a      *app
	events display.Events
}

type endpoint interface {
	Handle(method httpMethods, idParam string, w http.ResponseWriter, r *http.Request)
}

type handler struct {
	name           string
	path           string
	allowedMethods httpMethods
	subject        subjectProvider
	ep             endpoint
}

func nextPathItem(value string) (string, bool) {
	pos := strings.Index(value, "/")
	if pos == -1 {
		return value, true
	}
	return value[0:pos], false
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Clacks-Overhead", "GNU Terry Pratchett")
	method := parseMethod(r.Method)
	if method&h.allowedMethods != 0 {
		var id string
		switch h.subject {
		case noSubject:
			break
		case jsonSubject:
			raw, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, fmt.Sprintf("[%s] 500: unable to read body: %s",
					h.name, err), http.StatusInternalServerError)
				return
			}
			if err = json.Unmarshal(raw, &id); err != nil {
				http.Error(w, fmt.Sprintf("[%s] 400: body is not a string: %s",
					h.name, err), http.StatusBadRequest)
				return
			}
		case pathSubject:
			var isLast bool
			id, isLast = nextPathItem(r.URL.Path[len(h.path):])
			if !isLast {
				http.Error(w, fmt.Sprintf("[%s] 404: %s",
					h.name, r.URL.Path), http.StatusNotFound)
				return
			}
		}
		h.ep.Handle(method, id, w, r)
	} else {
		http.Error(w, fmt.Sprintf("[%s] 405: Method not allowed (supports %s, got %s)",
			h.name, h.allowedMethods, method), http.StatusMethodNotAllowed)
	}
}

type staticResource struct {
	contentType string
	content     []byte
}

type staticResourceEndpoint struct {
	env       *endpointEnv
	resources map[string]staticResource
}

// Handle implements the endpoint handler
func (ep *staticResourceEndpoint) Handle(
	method httpMethods, idParam string, w http.ResponseWriter, r *http.Request) {
	res, ok := ep.resources[r.URL.Path]
	if ok {
		w.Header().Set("Content-Type", res.contentType)
		w.Write(res.content)
	} else {
		http.NotFound(w, r)
	}
}

func newStaticResourceEndpoint(env *endpointEnv) *staticResourceEndpoint {
	ep := &staticResourceEndpoint{
		env: env, resources: make(map[string]staticResource)}

	indexRes := staticResource{
		contentType: "text/html; charset=utf-8", content: env.a.html}
	ep.resources["/"] = indexRes
	ep.resources["/index.html"] = indexRes
	ep.resources["/all.js"] = staticResource{
		contentType: "application/javascript", content: env.a.js}
	ep.resources["/style.css"] = staticResource{
		contentType: "text/css", content: env.a.css}
	return ep
}

func (ep *staticResourceEndpoint) add(path string, contentType string) {
	ep.resources[path] = staticResource{
		contentType: contentType, content: web.MustAsset("web" + path)}
}

func sendScene(a *app, sceneChan chan<- display.SceneUpdate) {
	data := make([]bool, len(a.modules))
	scene := a.config.Group(a.activeGroup).Scene(a.groupState.ActiveScene())
	for i := api.ModuleIndex(0); i < api.ModuleIndex(len(a.modules)); i++ {
		data[i] = scene.UsesModule(i)
		if data[i] {
			a.groupState.State(i).SendToModule()
		}
	}
	sceneChan <- display.SceneUpdate{ModuleEnabled: data}
}

func mergeAndSendConfigs(a *app,
	moduleConfigChan chan<- display.ModuleConfigUpdate) {
	scene := a.config.Group(a.activeGroup).Scene(a.groupState.ActiveScene())
	for i := api.ModuleIndex(0); i < api.ModuleIndex(len(a.modules)); i++ {
		if scene.UsesModule(i) {
			moduleConfigChan <- display.ModuleConfigUpdate{Index: i,
				Config: a.config.MergeConfig(i,
					a.activeSystem, a.activeGroup, a.groupState.ActiveScene())}
		}
	}
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

type sceneChangeEndpoint struct {
	env              *endpointEnv
	moduleConfigChan chan<- display.ModuleConfigUpdate
	sceneChan        chan<- display.SceneUpdate
	scenes           map[string]int
}

func newSceneChangeEndpoint(env *endpointEnv,
	moduleConfigChan chan<- display.ModuleConfigUpdate,
	sceneChan chan<- display.SceneUpdate) *sceneChangeEndpoint {
	return &sceneChangeEndpoint{env: env, moduleConfigChan: moduleConfigChan,
		sceneChan: sceneChan}
}

func (ep *sceneChangeEndpoint) Handle(
	method httpMethods, idParam string, w http.ResponseWriter, r *http.Request) {
	sceneIndex, ok := ep.scenes[idParam]
	if !ok {
		http.Error(w, "404: unknown scene \""+idParam+"\"", http.StatusNotFound)
		return
	}
	a := ep.env.a
	if err := a.setActiveScene(sceneIndex); err != nil {
		http.Error(w, "500: Failed to load active scene for target group",
			http.StatusInternalServerError)
		return
	}
	sendScene(a, ep.sceneChan)
	mergeAndSendConfigs(a, ep.moduleConfigChan)
	sdl.PushEvent(&sdl.UserEvent{Type: ep.env.events.SceneChangeID})
	moduleStates := a.groupState.BuildSceneStateJSON(a)
	ret, err := json.Marshal(groupChangeResponse{
		ActiveScene: a.groupState.ActiveScene(),
		Modules:     moduleStates,
	})
	sendJSON(w, ret, err)
}

type groupChangeEndpoint struct {
	env           *endpointEnv
	sceneChangeEP *sceneChangeEndpoint
	groups        map[string]int
}

func newGroupChangeEndpoint(env *endpointEnv,
	sceneChangeEP *sceneChangeEndpoint) *groupChangeEndpoint {
	ep := &groupChangeEndpoint{env: env, sceneChangeEP: sceneChangeEP,
		groups: make(map[string]int)}
	for i := 0; i < env.a.config.NumGroups(); i++ {
		group := env.a.config.Group(i)
		ep.groups[group.ID()] = i
	}
	return ep
}

type groupChangeResponse struct {
	ActiveScene int         `json:"activeScene"`
	Modules     interface{} `json:"modules"`
}

// ServeHTTP implements the HTTP server
func (ep *groupChangeEndpoint) Handle(
	method httpMethods, idParam string, w http.ResponseWriter, r *http.Request) {
	groupIndex, ok := ep.groups[idParam]
	if !ok {
		http.Error(w, "404: unknown group \""+idParam+"\"", http.StatusNotFound)
		return
	}
	a := ep.env.a
	activeScene, sceneNames, err := a.setActiveGroup(groupIndex)
	if err != nil {
		http.Error(w, "400: Could not set group: "+err.Error(),
			http.StatusBadRequest)
		return
	}
	if err = a.setActiveScene(activeScene); err != nil {
		http.Error(w, "500: Failed to load active scene for target group",
			http.StatusInternalServerError)
		return
	}
	sendScene(a, ep.sceneChangeEP.sceneChan)
	mergeAndSendConfigs(a, ep.sceneChangeEP.moduleConfigChan)
	sdl.PushEvent(&sdl.UserEvent{Type: ep.env.events.SceneChangeID})
	moduleStates := a.groupState.BuildSceneStateJSON(a)
	ep.sceneChangeEP.scenes = sceneNames
	ret, _ := json.Marshal(groupChangeResponse{
		ActiveScene: a.groupState.ActiveScene(),
		Modules:     moduleStates,
	})
	sendJSON(w, ret, err)
}

type configEndpoint struct {
	env              *endpointEnv
	moduleConfigChan chan<- display.ModuleConfigUpdate
}

func newConfigEndpoint(env *endpointEnv,
	moduleConfigChan chan<- display.ModuleConfigUpdate) *configEndpoint {
	return &configEndpoint{env: env, moduleConfigChan: moduleConfigChan}
}

// ServeHTTP implements the HTTP server
func (ch *configEndpoint) Handle(
	method httpMethods, idParam string, w http.ResponseWriter, r *http.Request) {
	post := method == httpPost
	var raw []byte
	if post {
		post = true
		var err error
		raw, err = ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	item, isLast := nextPathItem(r.URL.Path[len("/config/"):])
	a := ch.env.a
	switch item {
	case "base":
		if !isLast {
			http.Error(w, "404: \""+r.URL.Path+"\" not found", http.StatusNotFound)
		} else {
			if post {
				if err := a.config.LoadBaseJSON(raw); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
				} else {
					w.WriteHeader(http.StatusNoContent)
				}
			} else {
				var err error
				raw, err := a.config.BuildBaseJSON()
				sendJSON(w, raw, err)
			}
		}
	case "groups":
		if isLast {
			http.Error(w, "400: group missing", http.StatusBadRequest)
		} else {
			groupName := r.URL.Path[len("/config/groups/"):]
			if post {
				if err := a.config.LoadGroupJSON(raw, groupName); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
				} else {
					w.WriteHeader(http.StatusNoContent)
					mergeAndSendConfigs(a, ch.moduleConfigChan)
					sdl.PushEvent(&sdl.UserEvent{Type: ch.env.events.ModuleConfigID})
				}
			} else {
				var err error
				raw, err = a.config.BuildGroupJSON(groupName)
				sendJSON(w, raw, err)
			}
		}
	case "systems":
		if isLast {
			http.Error(w, "400: system missing", http.StatusBadRequest)
		} else {
			systemName := r.URL.Path[len("/config/systems/"):]
			if post {
				if err := a.config.LoadSystemJSON(raw, systemName); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
				} else {
					w.WriteHeader(http.StatusNoContent)
					mergeAndSendConfigs(a, ch.moduleConfigChan)
					sdl.PushEvent(&sdl.UserEvent{Type: ch.env.events.ModuleConfigID})
				}
			} else {
				var err error
				raw, err = a.config.BuildSystemJSON(systemName)
				sendJSON(w, raw, err)
			}
		}
	default:
		http.Error(w, "404: \""+r.URL.Path+"\" not found", http.StatusNotFound)
	}
}

type moduleStateEndpoint struct {
	env         *endpointEnv
	moduleIndex api.ModuleIndex
	prefixLen   int
	actions     map[string]int
}

func newModuleStateEndpoint(env *endpointEnv, actions []string,
	moduleIndex api.ModuleIndex, prefixLen int) *moduleStateEndpoint {
	ep := &moduleStateEndpoint{
		env: env, moduleIndex: moduleIndex, prefixLen: prefixLen,
		actions: make(map[string]int)}
	for i := range actions {
		ep.actions[actions[i]] = i
	}
	return ep
}

// ServeHTTP implements the HTTP server
func (ep *moduleStateEndpoint) Handle(
	method httpMethods, subject string, w http.ResponseWriter, r *http.Request) {
	actionIndex, ok := ep.actions[subject]
	if !ok {
		http.Error(w, "404: unknown action \""+subject+"\"",
			http.StatusNotFound)
		return
	}
	state := ep.env.a.groupState.State(ep.moduleIndex)
	if state == nil {
		http.Error(w, "400: module \""+
			ep.env.a.modules[ep.moduleIndex].Descriptor().ID+
			"\" not enabled for current scene", http.StatusBadRequest)
		return
	}

	payload, _ := ioutil.ReadAll(r.Body)
	responseObj, err := state.HandleAction(actionIndex, payload)
	var response []byte
	if err == nil {
		response, err = json.Marshal(responseObj)
	}
	if err != nil {
		http.Error(w, "400: "+err.Error(), http.StatusBadRequest)
	} else {
		sdl.PushEvent(&sdl.UserEvent{
			Type: ep.env.events.ModuleUpdateID, Code: int32(ep.moduleIndex)})
		if response == nil {
			w.WriteHeader(http.StatusNoContent)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(response)
		}
		newStateYaml, err :=
			ep.env.a.groupState.BuildYaml(ep.env.a, ep.env.a.config.Group(ep.env.a.activeGroup))
		if err != nil {
			panic(err)
		}
		go func(content []byte, filename string) {
			ioutil.WriteFile(filename, content, 0644)
		}(newStateYaml, ep.env.a.pathToState())
	}
}

func reg(h *handler) {
	http.Handle(h.path, h)
}

func startServer(owner *app,
	moduleConfigChan chan<- display.ModuleConfigUpdate,
	sceneChan chan<- display.SceneUpdate, events display.Events,
	port uint16) *http.Server {
	server := &http.Server{Addr: ":" + strconv.Itoa(int(port))}
	base := &endpointEnv{a: owner, events: events}

	sep := newStaticResourceEndpoint(base)
	sep.add("/css/pure-min.css", "text/css")
	sep.add("/css/grids-responsive-min.css", "text/css")
	sep.add("/css/fontawesome.min.css", "text/css")
	sep.add("/css/solid.min.css", "text/css")
	sep.add("/webfonts/fa-solid-900.eot", "application/vnd.ms-fontobject")
	sep.add("/webfonts/fa-solid-900.svg", "image/svg+xml")
	sep.add("/webfonts/fa-solid-900.ttf", "font/ttf")
	sep.add("/webfonts/fa-solid-900.woff", "font/woff")
	sep.add("/webfonts/fa-solid-900.woff2", "font/woff2")
	reg(&handler{name: "StaticResourceHandler", path: "/", allowedMethods: httpGet, ep: sep})

	scep := newSceneChangeEndpoint(base, moduleConfigChan, sceneChan)
	reg(&handler{name: "SceneChangeHandler", path: "/setscene",
		allowedMethods: httpPost, subject: jsonSubject, ep: scep})

	gcep := newGroupChangeEndpoint(base, scep)
	reg(&handler{name: "GroupChangeHandler", path: "/setgroup",
		allowedMethods: httpPost, subject: jsonSubject, ep: gcep})

	http.HandleFunc("/app", func(w http.ResponseWriter, r *http.Request) {
		activeScene := -1
		if owner.activeGroup != -1 {
			activeScene = owner.groupState.ActiveScene()
		}

		ret, err := owner.config.BuildGlobalJSON(
			owner, owner.activeGroup, activeScene)
		sendJSON(w, ret, err)
	})

	cfgEp := newConfigEndpoint(base, moduleConfigChan)
	reg(&handler{name: "ConfigHandler", path: "/config/",
		subject: noSubject, allowedMethods: httpGet | httpPost, ep: cfgEp})

	for i := range owner.modules {
		desc := owner.modules[i].Descriptor()
		actions := desc.Actions
		prefix := "/state/" + desc.ID + "/"
		modEp := newModuleStateEndpoint(base, actions, api.ModuleIndex(i),
			len(prefix))
		reg(&handler{name: "ModuleStateEndpoint", path: prefix,
			allowedMethods: httpPost, subject: pathSubject, ep: modEp})
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
