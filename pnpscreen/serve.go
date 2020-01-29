package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/flyx/pnpscreen/api"
	"github.com/flyx/pnpscreen/web"

	"github.com/flyx/pnpscreen/display"
)

// this file implements a RESTful server. The server's API is as follows:
//
// /[index.html]
//   GET: Returns the web client. The file name index.html is optional.
// /static
//   GET: Returns static data, i.e. data that will not change during the runtime
//        of the server.
// /data
//   GET: Returns the structure of all existing systems, groups, scenes and
//        heroes.
// /data/systems
//   POST: Creates a new system from the payload. Returns list of all systems.
// /data/systems/<system-id>
//   PUT: Updates system metadata
//   DELETE: Deletes the system with the id <system-id>
// /data/groups
//   POST: Creates a new group from the payload. Returns list of all groups.
// /data/groups/<group-id>
//   PUT: Updates group metadata
//   DELETE: Deletes the group with the id <group-id>
// /data/groups/<group-id>/scenes
//   POST: Creates a new scene from the payload in the group with the given id.
// /data/groups/<group-id>/scenes/<scene-id>
//   PUT: Updates scene metadata
//   DELETE: Deletes the scene with the given id from its group.
// /data/groups/<group-id>/heroes
//   POST: Creates a new hero from the payload in the group with the given id.
// /data/groups/<group-id>/heroes/<hero-id>
//   PUT: Updates hero metadata
//   DELETE: Deletes the hero with the given id from its group.
// /state
//   GET: Returns the current group, scene, and for each active module its
//        state.
//   POST: Changes active group or scene, returns same data as GET.
// /state/<module-id>/<entity-id>
//   PUT: Trigger an animation by changing the state of the given module.
// /config/base
//   GET: Returns the base configuration.
//   PUT: Updates the base configuration.
// /config/systems/<system-id>
//   GET: Returns the configuration of the system with the id <system-id>.
//   PUT: Updates said configuration.
// /config/groups/<group-id>
//   GET: Returns the configuration of the group with the id <group-id>.
//   PUT: Updates said configuration.
// /config/groups/<group-id>/<scene-id>
//   GET: Returns the configuration of the scene with the id <scene-id>
//        within the group with the id <group-id>.
//   PUT: Updates said configuration.

type staticResource struct {
	contentType string
	content     []byte
}

type staticResourceHandler struct {
	resources map[string]staticResource
}

func (srh *staticResourceHandler) ServeHTTP(
	w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Clacks-Overhead", "GNU Terry Pratchett")
	method := parseMethod(r.Method)
	if method != httpGet {
		http.Error(w, fmt.Sprintf(
			"[StaticResourceHandler] 405: Method not allowed (supports GET, got %s)",
			method), http.StatusMethodNotAllowed)
		return
	}
	res, ok := srh.resources[r.URL.Path]
	if ok {
		w.Header().Set("Content-Type", res.contentType)
		w.Write(res.content)
	} else {
		http.NotFound(w, r)
	}
}

func newStaticResourceHandler(a *app) *staticResourceHandler {
	ep := &staticResourceHandler{
		resources: make(map[string]staticResource)}

	indexRes := staticResource{
		contentType: "text/html; charset=utf-8", content: a.html}
	ep.resources["/"] = indexRes
	ep.resources["/index.html"] = indexRes
	ep.resources["/all.js"] = staticResource{
		contentType: "application/javascript", content: a.js}
	ep.resources["/style.css"] = staticResource{
		contentType: "text/css", content: a.css}
	return ep
}

func (srh *staticResourceHandler) add(path string, contentType string) {
	srh.resources[path] = staticResource{
		contentType: contentType, content: web.MustAsset("web" + path)}
}

type endpointEnv struct {
	a      *app
	events display.Events
}

func (env *endpointEnv) sendConfigsToDisplay() api.SendableError {
	if env.a.activeGroupIndex != -1 {
		req, err := env.a.display.StartRequest(env.events.ModuleConfigID, 0)
		if err != nil {
			return err
		}
		defer req.Close()
		mergeAndSendConfigs(env.a, &req)
		req.Commit()
	}
	return nil
}

func sendScene(a *app, req *display.Request) {
	data := make([]bool, len(a.modules))
	scene := a.activeGroup().Scene(a.data.ActiveScene())
	for i := api.ModuleIndex(0); i < api.ModuleIndex(len(a.modules)); i++ {
		data[i] = scene.UsesModule(i)
		if data[i] {
			req.SendModuleData(i, a.data.StateOf(i).CreateModuleData())
		}
	}
	req.SendEnabledModulesList(data)
}

func mergeAndSendConfigs(a *app, req *display.Request) {
	g := a.activeGroup()
	if g != nil {
		scene := g.Scene(a.data.ActiveScene())
		for i := api.ModuleIndex(0); i < api.ModuleIndex(len(a.modules)); i++ {
			if scene.UsesModule(i) {
				req.SendModuleConfig(i, a.data.MergeConfig(i,
					a.activeSystemIndex, a.activeGroupIndex, a.data.ActiveScene()))
			}
		}
	}
}

type staticDataEndpoint struct {
	*endpointEnv
}

func (sd staticDataEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, api.SendableError) {
	return sd.a.communication.StaticData(sd.a, sd.a.plugins), nil
}

type stateEndpoint struct {
	*endpointEnv
}

type stateAction int

const (
	setgroup stateAction = iota
	setscene
)

type validatedStateAction struct {
	Value struct {
		Action string `json:"action"`
		Index  int    `json:"index"`
	}
	Action               stateAction
	MaxGroups, MaxScenes int
}

func (vsa *validatedStateAction) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &api.ValidatedStruct{
		Value: &vsa.Value}); err != nil {
		return err
	}

	switch vsa.Value.Action {
	case "setgroup":
		vsa.Action = setgroup
		if vsa.Value.Index < 0 || vsa.Value.Index > vsa.MaxGroups {
			return fmt.Errorf("index out of range [0..%d]", vsa.Value.Index)
		}
	case "setscene":
		vsa.Action = setscene
		if vsa.Value.Index < 0 || vsa.Value.Index > vsa.MaxScenes {
			return fmt.Errorf("index out of range [0..%d]", vsa.Value.Index)
		}
	default:
		return fmt.Errorf("unknown action: %s", vsa.Value.Action)
	}
	return nil
}

func (se stateEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, api.SendableError) {
	activeScene := -1
	var modules interface{} = nil
	if method == httpPost {
		g := se.a.activeGroup()
		maxScenes := -1
		if g != nil {
			maxScenes = g.NumScenes() - 1
		}
		value := validatedStateAction{MaxGroups: se.a.data.NumGroups() - 1,
			MaxScenes: maxScenes}
		if err := api.ReceiveData(raw, &value); err != nil {
			return nil, err
		}

		req, err := se.a.display.StartRequest(se.events.SceneChangeID, 0)
		if err != nil {
			return nil, err
		}
		defer req.Close()

		switch value.Action {
		case setgroup:
			activeScene, err = se.a.setActiveGroup(value.Value.Index)
			if err != nil {
				return nil, err
			}
			if err = se.a.data.SetScene(activeScene); err != nil {
				return nil, err
			}
		case setscene:
			if g == nil {
				return nil, &api.BadRequest{Message: "No active group"}
			}

			if err := se.a.data.SetScene(value.Value.Index); err != nil {
				return nil, err
			}
			se.a.persistence.WriteState()
		}

		sendScene(se.a, &req)
		mergeAndSendConfigs(se.a, &req)
		req.Commit()
		modules = se.a.communication.ViewSceneState(se.a)
	} else {
		if se.a.activeGroupIndex != -1 {
			activeScene = se.a.data.ActiveScene()
			modules = se.a.communication.ViewSceneState(se.a)
		}
	}

	return struct {
		ActiveGroup int         `json:"activeGroup"`
		ActiveScene int         `json:"activeScene"`
		Modules     interface{} `json:"modules"`
	}{
		ActiveGroup: se.a.activeGroupIndex,
		ActiveScene: activeScene,
		Modules:     modules,
	}, nil
}

type baseConfigEndpoint struct {
	*endpointEnv
}

func (bce baseConfigEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, api.SendableError) {
	if method == httpPut {
		if err := bce.a.communication.UpdateBaseConfig(raw); err != nil {
			return nil, err
		}
		bce.a.persistence.WriteBase()
		return nil, bce.sendConfigsToDisplay()
	}
	return bce.a.communication.ViewBaseConfig(), nil
}

type systemConfigEndpoint struct {
	*endpointEnv
}

func (sce systemConfigEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, api.SendableError) {
	_, s := sce.a.data.SystemByID(ids[0])
	if s == nil {
		return nil, &api.NotFound{Name: ids[0]}
	}
	if method == httpPut {
		if err := sce.a.communication.UpdateSystemConfig(raw, s); err != nil {
			return nil, err
		}
		sce.a.persistence.WriteSystem(s)
		return nil, sce.sendConfigsToDisplay()
	}
	return sce.a.communication.ViewSystemConfig(s)
}

type groupConfigEndpoint struct {
	*endpointEnv
}

func (gce groupConfigEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, api.SendableError) {
	_, g := gce.a.data.GroupByID(ids[0])
	if g == nil {
		return nil, &api.NotFound{Name: ids[0]}
	}
	if method == httpPut {

		if err := gce.a.communication.UpdateGroupConfig(raw, g); err != nil {
			return nil, err
		}
		gce.a.persistence.WriteGroup(g)
		return nil, gce.sendConfigsToDisplay()
	}
	return gce.a.communication.ViewGroupConfig(g), nil
}

type sceneConfigEndpoint struct {
	*endpointEnv
}

func (sce sceneConfigEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, api.SendableError) {
	_, g := sce.a.data.GroupByID(ids[0])
	if g == nil {
		return nil, &api.NotFound{Name: ids[0]}
	}
	_, s := g.SceneByID(ids[1])
	if s == nil {
		return nil, &api.NotFound{Name: ids[1]}
	}
	if method == httpPut {
		if err := sce.a.communication.UpdateSceneConfig(raw, s); err != nil {
			return nil, err
		}
		sce.a.persistence.WriteScene(g, s)
		return nil, sce.sendConfigsToDisplay()
	}
	return sce.a.communication.ViewSceneConfig(s), nil
}

type moduleEndpoint struct {
	*endpointEnv
	moduleIndex   api.ModuleIndex
	endpointIndex int
	pure          bool
}

func (me *moduleEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, api.SendableError) {
	state := me.a.data.State.StateOf(me.moduleIndex)
	if state == nil {
		return nil, &api.BadRequest{
			Message: fmt.Sprintf("module \"%s\" not enabled for current scene",
				me.a.modules[me.moduleIndex].Descriptor().ID)}
	}

	req, err := me.a.display.StartRequest(
		me.events.ModuleUpdateID, int32(me.moduleIndex))
	if err != nil {
		return nil, err
	}
	defer req.Close()

	var responseObj, data interface{}
	if me.pure {
		ep := state.(api.PureEndpointProvider).PureEndpoint(me.endpointIndex)
		responseObj, data, err = ep.Put(raw)
	} else {
		ep := state.(api.IDEndpointProvider).IDEndpoint(me.endpointIndex)
		responseObj, data, err = ep.Put(ids[0], raw)
	}
	if err != nil {
		return nil, err
	}

	req.SendModuleData(me.moduleIndex, data)
	req.Commit()
	me.a.persistence.WriteState()
	return responseObj, nil
}

type dataEndpoint struct {
	*endpointEnv
}

func (de dataEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, api.SendableError) {
	return de.a.communication.ViewAll(de.a), nil
}

type systemEndpoint struct {
	*endpointEnv
}

func (se systemEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, api.SendableError) {
	index, s := se.a.data.SystemByID(ids[0])
	if s == nil {
		return nil, &api.NotFound{Name: ids[0]}
	}
	if method == httpPut {
		if err := se.a.communication.UpdateSystem(raw, s); err != nil {
			return nil, err
		}
		if err := se.a.persistence.WriteSystem(s); err != nil {
			log.Println("failed to persist system: " + err.Error())
		}
		return se.a.communication.ViewSystems(), nil
	}
	return nil, se.a.persistence.DeleteSystem(index)
}

type dataGroupEndpoint struct {
	*endpointEnv
}

func (dge dataGroupEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, api.SendableError) {
	index, g := dge.a.data.GroupByID(ids[0])
	if g == nil {
		return nil, &api.NotFound{Name: ids[0]}
	}
	if method == httpPut {
		if err := dge.a.communication.UpdateGroup(raw, g); err != nil {
			return nil, err
		}
		if err := dge.a.persistence.WriteGroup(g); err != nil {
			log.Println("failed to persist group: " + err.Error())
		}
		// TODO: check group index; if active group, update stuff since index could
		// have been changed due to reordering
	} else {
		dge.a.persistence.DeleteGroup(index)
	}
	return dge.a.communication.ViewGroups(), nil
}

type dataSystemsEndpoint struct {
	*endpointEnv
}

func (sc dataSystemsEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, api.SendableError) {
	name := api.ValidatedString{MinLen: 1, MaxLen: -1}
	if err := api.ReceiveData(raw, &name); err != nil {
		return nil, err
	}
	if err := sc.a.persistence.CreateSystem(name.Value); err != nil {
		return nil, err
	}
	return sc.a.communication.ViewSystems(), nil
}

type dataGroupsEndpoint struct {
	*endpointEnv
}

type groupCreationReceiver struct {
	data struct {
		Name               string `json:"name"`
		PluginIndex        int    `json:"pluginIndex"`
		GroupTemplateIndex int    `json:"groupTemplateIndex"`
	}
	plugins []*api.Plugin
}

func (gcr *groupCreationReceiver) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data,
		&api.ValidatedStruct{Value: &gcr.data}); err != nil {
		return err
	}
	if gcr.data.Name == "" {
		return errors.New("name must not be empty")
	} else if gcr.data.PluginIndex < 0 ||
		gcr.data.PluginIndex >= len(gcr.plugins) {
		return fmt.Errorf("pluginIndex out of range [0..%d]", len(gcr.plugins)-1)
	} else if gcr.data.GroupTemplateIndex < 0 ||
		gcr.data.GroupTemplateIndex >= len(gcr.plugins[gcr.data.PluginIndex].GroupTemplates) {
		return fmt.Errorf("groupTemplateIndex out of range [0..%d]",
			len(gcr.plugins[gcr.data.PluginIndex].GroupTemplates)-1)
	}
	return nil
}

func (dge dataGroupsEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, api.SendableError) {
	value := groupCreationReceiver{plugins: dge.a.plugins}
	if err := api.ReceiveData(raw, &value); err != nil {
		return nil, err
	}
	if err := dge.a.persistence.CreateGroup(value.data.Name,
		&dge.a.plugins[value.data.PluginIndex].GroupTemplates[value.data.GroupTemplateIndex],
		dge.a.plugins[value.data.PluginIndex].SceneTemplates); err != nil {
		return nil, &api.InternalError{
			Description: "while creating group", Inner: err}
	}
	return dge.a.communication.ViewGroups(), nil
}

type dataScenesEndpoint struct {
	*endpointEnv
}

type sceneCreationReceiver struct {
	data struct {
		Name               string `json:"name"`
		PluginIndex        int    `json:"pluginIndex"`
		SceneTemplateIndex int    `json:"sceneTemplateIndex"`
	}
	plugins []*api.Plugin
}

func (scr *sceneCreationReceiver) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data,
		&api.ValidatedStruct{Value: &scr.data}); err != nil {
		return err
	}
	if scr.data.Name == "" {
		return errors.New("name must not be empty")
	} else if scr.data.PluginIndex < 0 ||
		scr.data.PluginIndex >= len(scr.plugins) {
		return fmt.Errorf("pluginIndex out of range [0..%d]", len(scr.plugins)-1)
	} else if scr.data.SceneTemplateIndex < 0 ||
		scr.data.SceneTemplateIndex >= len(scr.plugins[scr.data.PluginIndex].SceneTemplates) {
		return fmt.Errorf("groupTemplateIndex out of range [0..%d]",
			len(scr.plugins[scr.data.PluginIndex].SceneTemplates)-1)
	}
	return nil
}

func (dse dataScenesEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, api.SendableError) {
	_, group := dse.a.data.GroupByID(ids[0])
	if group == nil {
		return nil, &api.NotFound{Name: ids[0]}
	}
	value := sceneCreationReceiver{plugins: dse.a.plugins}
	if err := api.ReceiveData(raw, &value); err != nil {
		return nil, err
	}

	if err := dse.a.persistence.CreateScene(group, value.data.Name,
		&dse.a.plugins[value.data.PluginIndex].SceneTemplates[value.data.SceneTemplateIndex]); err != nil {
		return nil, &api.InternalError{Description: "while creating group", Inner: err}
	}
	return dse.a.communication.ViewScenes(group), nil
}

type dataSceneEndpoint struct {
	*endpointEnv
}

func (dse dataSceneEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, api.SendableError) {
	_, group := dse.a.data.GroupByID(ids[0])
	if group == nil {
		return nil, &api.NotFound{Name: ids[0]}
	}
	sceneIndex, scene := group.SceneByID(ids[1])
	if scene == nil {
		return nil, &api.NotFound{Name: ids[0]}
	}
	if method == httpPut {
		if err := dse.a.communication.UpdateScene(raw, group, scene); err != nil {
			return nil, err
		}
		if err := dse.a.persistence.WriteScene(group, scene); err != nil {
			log.Println("failed to persist scene: " + err.Error())
		}
		// TODO: check scene index; if active scene, update stuff since index could
		// have been changed due to reordering
	} else {
		dse.a.persistence.DeleteScene(group, sceneIndex)
	}
	return dse.a.communication.ViewScenes(group), nil
}

func startServer(owner *app, events display.Events, port uint16) *http.Server {
	server := &http.Server{Addr: ":" + strconv.Itoa(int(port))}
	env := &endpointEnv{a: owner, events: events}

	sep := newStaticResourceHandler(owner)
	sep.add("/css/pure-min.css", "text/css")
	sep.add("/css/grids-responsive-min.css", "text/css")
	sep.add("/css/fontawesome.min.css", "text/css")
	sep.add("/css/solid.min.css", "text/css")
	sep.add("/webfonts/fa-solid-900.eot", "application/vnd.ms-fontobject")
	sep.add("/webfonts/fa-solid-900.svg", "image/svg+xml")
	sep.add("/webfonts/fa-solid-900.ttf", "font/ttf")
	sep.add("/webfonts/fa-solid-900.woff", "font/woff")
	sep.add("/webfonts/fa-solid-900.woff2", "font/woff2")
	http.Handle("/", sep)

	reg("StaticDataHandler", "/static",
		endpoint{httpGet, &staticDataEndpoint{env}})
	reg("StateHandler", "/state",
		endpoint{httpGet | httpPost, &stateEndpoint{env}})
	reg("BaseConfigHandler", "/config/base",
		endpoint{httpGet | httpPut, &baseConfigEndpoint{env}})
	reg("SystemConfigHandler", "/config/systems/",
		idCapture{}, endpoint{httpGet | httpPut, &systemConfigEndpoint{env}})
	reg("GroupConfigHandler", "/config/groups/",
		idCapture{}, endpoint{httpGet | httpPut, &groupConfigEndpoint{env}},
		pathFragment("scenes"), idCapture{},
		endpoint{httpGet | httpPut, &sceneConfigEndpoint{env}})
	reg("DataHandler", "/data", endpoint{httpGet, &dataEndpoint{env}})
	reg("DataSystemsHandler", "/data/systems",
		endpoint{httpPost, &dataSystemsEndpoint{env}})
	reg("DataSystemHandler", "/data/systems/", idCapture{},
		endpoint{httpPut | httpDelete, &systemEndpoint{env}})
	reg("DataGroupsHandler", "/data/groups",
		endpoint{httpPost, &dataGroupsEndpoint{env}})
	reg("DataGroupHandler", "/data/groups/", idCapture{},
		endpoint{httpPut | httpDelete, &dataGroupEndpoint{env}},
		pathFragment("scenes"), endpoint{httpPost, &dataScenesEndpoint{env}},
		idCapture{}, endpoint{httpPut | httpDelete, &dataSceneEndpoint{env}})

	for i := api.ModuleIndex(0); i < owner.NumModules(); i++ {
		seenSlash := false
		seenOthers := false

		desc := owner.modules[i].Descriptor()
		seen := make(map[string]struct{})

		for j := range desc.EndpointPaths {
			path := desc.EndpointPaths[j]
			_, ok := seen[path]
			if ok {
				panic("module " + desc.Name + " has duplicate endpoint path " + path)
			}
			seen[path] = struct{}{}
			if path == "" {
			} else if path == "/" {
				if seenOthers {
					panic("module " + desc.Name +
						" has \"/\" endpoint path besides non-empty paths")
				}
				seenSlash = true
			} else {
				if seenSlash {
					panic("module " + desc.Name +
						" has \"/\" endpoint path besides non-empty paths")
				}
				seenOthers = true
			}
			var builder strings.Builder
			builder.WriteString("/state/")
			builder.WriteString(desc.ID)
			if len(path) != 0 && path[0] != '/' {
				builder.WriteByte('/')
			}
			builder.WriteString(path)
			if len(path) != 0 && path[len(path)-1] == '/' {
				reg("ModuleEndpoint("+path+")", builder.String(),
					idCapture{}, endpoint{httpPut,
						&moduleEndpoint{endpointEnv: env, moduleIndex: i, endpointIndex: j,
							pure: false}})
			} else {
				reg("ModuleEndpoint("+path+")", builder.String(),
					endpoint{httpPut,
						&moduleEndpoint{endpointEnv: env, moduleIndex: i, endpointIndex: j,
							pure: true}})
			}
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
