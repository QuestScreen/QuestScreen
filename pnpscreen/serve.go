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
	*endpointEnv
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
		endpointEnv: env, resources: make(map[string]staticResource)}

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

func sendScene(a *app, req *display.Request) {
	data := make([]bool, len(a.modules))
	scene := a.activeGroup().Scene(a.groupState.ActiveScene())
	for i := api.ModuleIndex(0); i < api.ModuleIndex(len(a.modules)); i++ {
		data[i] = scene.UsesModule(i)
		if data[i] {
			req.SendModuleData(i, a.groupState.State(i).CreateModuleData())
		}
	}
	req.SendEnabledModulesList(data)
}

func mergeAndSendConfigs(a *app, req *display.Request) {
	g := a.activeGroup()
	if g != nil {
		scene := g.Scene(a.groupState.ActiveScene())
		for i := api.ModuleIndex(0); i < api.ModuleIndex(len(a.modules)); i++ {
			if scene.UsesModule(i) {
				req.SendModuleConfig(i, a.config.MergeConfig(i,
					a.activeSystemIndex, a.activeGroupIndex, a.groupState.ActiveScene()))
			}
		}
	}
}

func sendJSON(w http.ResponseWriter, data interface{}) {
	content, err := json.Marshal(data)
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	} else {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

type staticDataEndpoint struct {
	*endpointEnv
}

func (sd *staticDataEndpoint) Handle(
	method httpMethods, subject string, w http.ResponseWriter, r *http.Request) {
	sendJSON(w, sd.a.communication.StaticData(sd.a, sd.a.plugins))
}

func newStaticDataEndpoint(env *endpointEnv) *staticDataEndpoint {
	return &staticDataEndpoint{endpointEnv: env}
}

type stateEndpoint struct {
	*endpointEnv
}

func newStateEndpoint(env *endpointEnv) *stateEndpoint {
	return &stateEndpoint{endpointEnv: env}
}

func (se *stateEndpoint) Handle(
	method httpMethods, idParam string, w http.ResponseWriter, r *http.Request) {
	activeScene := -1
	var modules interface{} = nil
	if method == httpPost {
		var request struct {
			Action string `json:"action"`
			Index  int    `json:"index"`
		}
		raw, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "500: could not read request body", http.StatusInternalServerError)
			return
		}
		if err = json.Unmarshal(raw, &request); err != nil {
			http.Error(w, "400: could not unmarshal request body: "+err.Error(),
				http.StatusBadRequest)
			return
		}

		req, err := se.a.display.StartRequest(se.events.SceneChangeID, 0)
		if err != nil {
			http.Error(w, "503: Previous request still pending",
				http.StatusServiceUnavailable)
			return
		}
		defer req.Close()

		switch request.Action {
		case "setgroup":
			if request.Index < 0 || request.Index >= se.a.config.NumGroups() {
				http.Error(w, "400: group index out of range", http.StatusBadRequest)
				return
			}

			activeScene, err = se.a.setActiveGroup(request.Index)
			if err != nil {
				http.Error(w, "400: Could not set group: "+err.Error(),
					http.StatusBadRequest)
				return
			}
			if err = se.a.groupState.SetScene(activeScene); err != nil {
				http.Error(w, "500: Failed to load active scene for target group",
					http.StatusInternalServerError)
				return
			}
		case "setscene":
			g := se.a.activeGroup()
			if g == nil || request.Index < 0 || request.Index >= g.NumScenes() {
				http.Error(w, "400: scene index out of range", http.StatusBadRequest)
				return
			}
			activeScene = request.Index
			if err := se.a.groupState.SetScene(activeScene); err != nil {
				http.Error(w, "500: Failed to load active scene for target group",
					http.StatusInternalServerError)
				return
			}
			se.a.groupState.Persist()
		default:
			http.Error(w, "400: unknown action: "+request.Action, http.StatusBadRequest)
			return
		}

		sendScene(se.a, &req)
		mergeAndSendConfigs(se.a, &req)
		req.Commit()
		modules = se.a.groupState.CommunicateSceneState(se.a)
	} else {
		if se.a.groupState != nil {
			activeScene = se.a.groupState.ActiveScene()
			modules = se.a.groupState.CommunicateSceneState(se.a)
		}
	}

	sendJSON(w, struct {
		ActiveGroup int         `json:"activeGroup"`
		ActiveScene int         `json:"activeScene"`
		Modules     interface{} `json:"modules"`
	}{
		ActiveGroup: se.a.activeGroupIndex,
		ActiveScene: activeScene,
		Modules:     modules,
	})
}

type configEndpoint struct {
	*endpointEnv
}

func newConfigEndpoint(env *endpointEnv) *configEndpoint {
	return &configEndpoint{endpointEnv: env}
}

// ServeHTTP implements the HTTP server
func (ce *configEndpoint) Handle(
	method httpMethods, idParam string, w http.ResponseWriter, r *http.Request) {
	post := method == httpPost
	var raw []byte
	var err error
	var view interface{}
	if post {
		post = true
		raw, err = ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	if len(r.URL.Path) <= len("/config/") {
		http.Error(w, "404: not found", http.StatusNotFound)
		return
	}

	item, isLast := nextPathItem(r.URL.Path[len("/config/"):])
	switch item {
	case "base":
		if !isLast {
			http.Error(w, "404: \""+r.URL.Path+"\" not found", http.StatusNotFound)
			return
		}
		if post {
			if err := ce.a.communication.LoadBase(raw); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			ce.a.persistence.WriteBase()
			break
		}
		view = ce.a.communication.Base()
	case "groups":
		if isLast {
			http.Error(w, "400: group missing", http.StatusBadRequest)
			return
		}
		groupName, isLast := nextPathItem(r.URL.Path[len("/config/groups/"):])
		if isLast {
			if post {
				group, err := ce.a.communication.LoadGroup(raw, groupName)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				ce.a.persistence.WriteGroup(group)
			} else {
				view, err = ce.a.communication.Group(groupName)
			}
			break
		}
		sceneName, isLast :=
			nextPathItem(r.URL.Path[len("/config/groups//")+len(groupName):])
		if !isLast {
			http.Error(w, "404: not found", http.StatusNotFound)
			return
		}
		if post {
			group, scene, err := ce.a.communication.LoadScene(raw, groupName, sceneName)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			ce.a.persistence.WriteScene(group, scene)
		} else {
			view, err = ce.a.communication.Scene(groupName, sceneName)
		}
	case "systems":
		if isLast {
			http.Error(w, "400: system missing", http.StatusBadRequest)
		} else {
			systemName, isLast := nextPathItem(r.URL.Path[len("/config/systems/"):])
			if !isLast {
				http.Error(w, "404: not found", http.StatusNotFound)
				return
			}
			if post {
				system, err := ce.a.communication.LoadSystem(raw, systemName)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				ce.a.persistence.WriteSystem(system)
			} else {
				view, err = ce.a.communication.System(systemName)
			}
		}
	default:
		http.Error(w, "404: \""+r.URL.Path+"\" not found", http.StatusNotFound)
		return
	}

	if post {
		if ce.a.activeGroupIndex != -1 {
			req, err := ce.a.display.StartRequest(ce.events.ModuleConfigID, 0)
			if err != nil {
				http.Error(w, "503: Previous request still pending",
					http.StatusServiceUnavailable)
				return
			}
			defer req.Close()
			mergeAndSendConfigs(ce.a, &req)
			req.Commit()
		}
		w.WriteHeader(http.StatusNoContent)
	} else if err != nil {
		http.Error(w, fmt.Sprintf("400: %s", err.Error()), http.StatusBadRequest)
	} else {
		sendJSON(w, view)
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

func (ep *moduleStateEndpoint) Handle(
	method httpMethods, subject string, w http.ResponseWriter, r *http.Request) {
	actionIndex, ok := ep.actions[subject]
	if !ok {
		http.Error(w, "404: unknown action \""+subject+"\"",
			http.StatusNotFound)
		return
	}
	a := ep.env.a
	state := a.groupState.State(ep.moduleIndex)
	if state == nil {
		http.Error(w, "400: module \""+
			ep.env.a.modules[ep.moduleIndex].Descriptor().ID+
			"\" not enabled for current scene", http.StatusBadRequest)
		return
	}

	req, err := a.display.StartRequest(
		ep.env.events.ModuleUpdateID, int32(ep.moduleIndex))
	if err != nil {
		http.Error(w, "503: Previous request still pending",
			http.StatusServiceUnavailable)
		return
	}
	defer req.Close()

	payload, _ := ioutil.ReadAll(r.Body)
	responseObj, data, err := state.HandleAction(actionIndex, payload)
	var response []byte
	if err == nil {
		response, err = json.Marshal(responseObj)
	}
	if err != nil {
		http.Error(w, "500: "+err.Error(), http.StatusBadRequest)
	} else {
		req.SendModuleData(ep.moduleIndex, data)
		req.Commit()
		if response == nil {
			w.WriteHeader(http.StatusNoContent)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(response)
		}
		ep.env.a.groupState.Persist()
	}
}

type datasetsEndpoint struct {
	*endpointEnv
}

func (de *datasetsEndpoint) Handle(
	method httpMethods, subject string, w http.ResponseWriter, r *http.Request) {
	sendJSON(w, de.a.communication.Datasets(de.a))
}

func newDatasetsEndpoint(env *endpointEnv) *datasetsEndpoint {
	return &datasetsEndpoint{endpointEnv: env}
}

type systemDeleteEndpoint struct {
	env *endpointEnv
}

func newSystemDeleteEndpoint(env *endpointEnv) *systemDeleteEndpoint {
	return &systemDeleteEndpoint{env: env}
}

func (sd *systemDeleteEndpoint) Handle(
	method httpMethods, subject string, w http.ResponseWriter, r *http.Request) {
	a := sd.env.a
	if err := a.persistence.DeleteSystem(subject); err != nil {
		http.Error(w, "400: "+err.Error(), http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

type groupDeleteEndpoint struct {
	env *endpointEnv
}

func newGroupDeleteEndpoint(env *endpointEnv) *groupDeleteEndpoint {
	return &groupDeleteEndpoint{env: env}
}

func (gd *groupDeleteEndpoint) Handle(
	method httpMethods, subject string, w http.ResponseWriter, r *http.Request) {
	a := gd.env.a
	if err := a.persistence.DeleteGroup(subject); err != nil {
		http.Error(w, "400: "+err.Error(), http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

type systemCreateEndpoint struct {
	env *endpointEnv
}

func newSystemCreateEndpoint(env *endpointEnv) *systemCreateEndpoint {
	return &systemCreateEndpoint{env: env}
}

func (sc *systemCreateEndpoint) Handle(
	method httpMethods, subject string, w http.ResponseWriter, r *http.Request) {
	a := sc.env.a
	if err := a.persistence.CreateSystem(subject); err != nil {
		http.Error(w, "500: "+err.Error(), http.StatusInternalServerError)
	} else {
		sendJSON(w, a.communication.Systems())
	}
}

type groupCreateEndpoint struct {
	env *endpointEnv
}

func newGroupCreateEndpoint(env *endpointEnv) *groupCreateEndpoint {
	return &groupCreateEndpoint{env: env}
}

func (gc *groupCreateEndpoint) Handle(
	method httpMethods, subject string, w http.ResponseWriter, r *http.Request) {
	a := gc.env.a
	raw, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("[GroupCreateHandler] 500: unable to read body: %s",
			err), http.StatusInternalServerError)
		return
	}

	data := struct {
		Name               string `json:"name"`
		PluginIndex        int    `json:"pluginIndex"`
		GroupTemplateIndex int    `json:"groupTemplateIndex"`
	}{Name: "", PluginIndex: -1, GroupTemplateIndex: -1}
	if err := json.Unmarshal(raw, &data); err != nil {
		http.Error(w, "400: cannot deserialize input data: %s"+err.Error(),
			http.StatusBadRequest)
	} else if data.Name == "" {
		http.Error(w, "400: `name` must not be empty", http.StatusBadRequest)
	} else if data.PluginIndex < 0 || data.PluginIndex >= len(a.plugins) {
		http.Error(w, "400: `pluginIndex` out of range 0.."+
			strconv.Itoa(len(a.plugins)-1), http.StatusBadRequest)
	} else if data.GroupTemplateIndex < 0 ||
		data.GroupTemplateIndex > len(a.plugins[data.PluginIndex].GroupTemplates) {
		http.Error(w, "400: `groupTemplateIndex` out of range 0.."+
			strconv.Itoa(len(a.plugins[data.PluginIndex].GroupTemplates)-1),
			http.StatusBadRequest)
	} else {
		if err := a.persistence.CreateGroup(data.Name,
			&a.plugins[data.PluginIndex].GroupTemplates[data.GroupTemplateIndex],
			a.plugins[data.PluginIndex].SceneTemplates); err != nil {
			http.Error(w, "500: "+err.Error(), http.StatusInternalServerError)
		} else {
			sendJSON(w, a.communication.Groups())
		}
	}
}

func reg(h *handler) {
	http.Handle(h.path, h)
}

func startServer(owner *app, events display.Events, port uint16) *http.Server {
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

	sdEp := newStaticDataEndpoint(base)
	reg(&handler{name: "StaticDataHandler", path: "/static", allowedMethods: httpGet, ep: sdEp})

	stEp := newStateEndpoint(base)
	reg(&handler{name: "StateHandler", path: "/state", allowedMethods: httpGet | httpPost, ep: stEp})

	cfgEp := newConfigEndpoint(base)
	reg(&handler{name: "ConfigHandler", path: "/config/",
		subject: noSubject, allowedMethods: httpGet | httpPost, ep: cfgEp})

	dsEp := newDatasetsEndpoint(base)
	reg(&handler{name: "DatasetsHandler", path: "/datasets",
		subject: noSubject, allowedMethods: httpGet, ep: dsEp})

	sdelEp := newSystemDeleteEndpoint(base)
	reg(&handler{name: "SystemDeleteHandler", path: "/datasets/system/delete",
		subject: jsonSubject, allowedMethods: httpPost, ep: sdelEp})

	gdelEp := newGroupDeleteEndpoint(base)
	reg(&handler{name: "GroupDeleteHandler", path: "/datasets/group/delete",
		subject: jsonSubject, allowedMethods: httpPost, ep: gdelEp})

	screEp := newSystemCreateEndpoint(base)
	reg(&handler{name: "SystemCreateHandler", path: "/datasets/system/create",
		subject: jsonSubject, allowedMethods: httpPost, ep: screEp})

	gcreEp := newGroupCreateEndpoint(base)
	reg(&handler{name: "GroupCreateHandler", path: "/datasets/group/create",
		subject: noSubject, allowedMethods: httpPost, ep: gcreEp})

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
