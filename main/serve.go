package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/QuestScreen/QuestScreen/assets"
	"github.com/QuestScreen/QuestScreen/shared"
	"github.com/QuestScreen/api/comms"
	"github.com/QuestScreen/api/groups"
	"github.com/QuestScreen/api/modules"
	"github.com/QuestScreen/api/server"

	"github.com/QuestScreen/QuestScreen/display"
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
// /state/<module-id>[/<endpoint-path>][/<entity-id>]
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

type contentTypesS struct {
	HTML, JS, CSS, OctetStream, PNG, SVG, XML, ICO, EOT, TTF, WOFF, WOFF2,
	Webmanifest string
}

func (ct *contentTypesS) get(extension string) string {
	switch extension {
	case ".html":
		return ct.HTML
	case ".js":
		return ct.JS
	case ".css":
		return ct.CSS
	case ".png":
		return ct.PNG
	case ".svg":
		return ct.SVG
	case ".xml":
		return ct.XML
	case ".ico":
		return ct.ICO
	case ".eot":
		return ct.EOT
	case ".ttf":
		return ct.TTF
	case ".woff":
		return ct.WOFF
	case ".woff2":
		return ct.WOFF2
	case ".webmanifest":
		return ct.Webmanifest
	default:
		return ct.OctetStream
	}
}

var contentTypes = contentTypesS{
	HTML: "text/html", JS: "application/javascript", CSS: "text/css",
	PNG: "image/png", XML: "application/xml", ICO: "image/vnd.microsoft.icon",
	SVG: "image/svg+xml", EOT: "application/vnd.ms-fontobject", TTF: "font/ttf",
	WOFF: "font/woff", WOFF2: "font/woff2",
	Webmanifest: "application/manifest+json",
	OctetStream: "application/octet-stream",
}

func newStaticResourceHandler(qs *QuestScreen) *staticResourceHandler {
	srh := &staticResourceHandler{
		resources: make(map[string]staticResource)}

	for _, assetName := range assets.AssetNames() {
		res := staticResource{
			contentType: contentTypes.get(filepath.Ext(assetName)),
			content:     assets.MustAsset(assetName),
		}
		srh.resources["/"+assetName] = res
		if assetName == "index.html" {
			srh.resources["/"] = res
		}
	}

	return srh
}

type endpointEnv struct {
	qs     *QuestScreen
	events display.Events
}

func (env *endpointEnv) sendConfigsToDisplay() server.Error {
	if env.qs.activeGroupIndex != -1 {
		req, err := env.qs.display.StartRequest(env.events.ModuleConfigID, 0)
		if err != nil {
			return err
		}
		defer req.Close()
		mergeAndSendConfigs(env.qs, &req)
		req.Commit()
	}
	return nil
}

func sendScene(qs *QuestScreen, req *display.Request) {
	data := make([]bool, len(qs.modules))
	scene := qs.activeGroup().Scene(qs.data.ActiveScene())
	for i := shared.FirstModule; i < qs.NumModules(); i++ {
		data[i] = scene.UsesModule(i)
		if data[i] {
			req.SendRendererData(i, qs.data.StateOf(i).CreateRendererData(
				qs.ServerContext(i)))
		}
	}
	req.SendEnabledModulesList(data)
}

func propagateHeroesChange(action groups.HeroChangeAction,
	heroIndex int, qs *QuestScreen, req *display.Request) {
	g := qs.activeGroup()
	if g == nil {
		return
	}
	for i := 0; i < g.NumScenes(); i++ {
		scene := g.Scene(i)
		for j := shared.FirstModule; j < qs.NumModules(); j++ {
			if scene.UsesModule(j) {
				state := qs.data.State.StateOfScene(i, j)
				hams, ok := state.(modules.HeroAwareState)
				if ok {
					ctx := qs.ServerContext(j)
					hams.HeroListChanged(ctx, action, heroIndex)
					if i == qs.data.State.ActiveScene() {
						req.SendRendererData(j, state.CreateRendererData(ctx))
					}
				}
			}
		}
	}
}

func mergeAndSendConfigs(qs *QuestScreen, req *display.Request) {
	g := qs.activeGroup()
	if g != nil {
		scene := g.Scene(qs.data.ActiveScene())
		for i := shared.FirstModule; i < qs.NumModules(); i++ {
			if scene.UsesModule(i) {
				req.SendModuleConfig(i, qs.data.MergeConfig(i,
					qs.activeSystemIndex, qs.activeGroupIndex, qs.data.ActiveScene()))
			}
		}
	}
}

type staticDataEndpoint struct {
	*endpointEnv
}

func (sd staticDataEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, server.Error) {
	return sd.qs.communication.StaticData(sd.qs, sd.qs.plugins), nil
}

type stateEndpoint struct {
	*endpointEnv
}

type stateAction int

const (
	setgroup stateAction = iota
	setscene
	leaveGroup
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
	if err := json.Unmarshal(data, &comms.ValidatedStruct{
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
	case "leavegroup":
		vsa.Action = leaveGroup
	default:
		return fmt.Errorf("unknown action: %s", vsa.Value.Action)
	}
	return nil
}

func (se stateEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, server.Error) {
	activeScene := -1
	var modules interface{} = nil
	if method == httpPost {
		g := se.qs.activeGroup()
		maxScenes := -1
		if g != nil {
			maxScenes = g.NumScenes() - 1
		}
		value := validatedStateAction{MaxGroups: se.qs.data.NumGroups() - 1,
			MaxScenes: maxScenes}
		if err := comms.ReceiveData(raw, &value); err != nil {
			return nil, &server.BadRequest{
				Inner: err, Message: "received invalid data"}
		}

		if value.Action == leaveGroup {
			se.qs.setActiveGroup(-1)
			req, err := se.qs.display.StartRequest(se.events.LeaveGroupID, 0)
			if err != nil {
				return nil, err
			}
			req.Commit()
		} else {
			req, err := se.qs.display.StartRequest(se.events.SceneChangeID, 0)
			if err != nil {
				return nil, err
			}
			defer req.Close()

			switch value.Action {
			case setgroup:
				activeScene, err = se.qs.setActiveGroup(value.Value.Index)
				if err != nil {
					return nil, err
				}
				if err = se.qs.data.SetScene(activeScene); err != nil {
					return nil, err
				}
			case setscene:
				if g == nil {
					return nil, &server.BadRequest{Message: "No active group"}
				}

				if err := se.qs.data.SetScene(value.Value.Index); err != nil {
					return nil, err
				}
				se.qs.persistence.WriteState()
			}

			sendScene(se.qs, &req)
			mergeAndSendConfigs(se.qs, &req)
			req.Commit()
			modules = se.qs.communication.ViewSceneState(se.qs)
		}
	} else {
		if se.qs.activeGroupIndex != -1 {
			activeScene = se.qs.data.ActiveScene()
			modules = se.qs.communication.ViewSceneState(se.qs)
		}
	}

	return struct {
		ActiveGroup int         `json:"activeGroup"`
		ActiveScene int         `json:"activeScene"`
		Modules     interface{} `json:"modules"`
	}{
		ActiveGroup: se.qs.activeGroupIndex,
		ActiveScene: activeScene,
		Modules:     modules,
	}, nil
}

type baseConfigEndpoint struct {
	*endpointEnv
}

func (bce baseConfigEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, server.Error) {
	if method == httpPut {
		if err := bce.qs.communication.UpdateBaseConfig(raw); err != nil {
			return nil, err
		}
		bce.qs.persistence.WriteBase()
		return nil, bce.sendConfigsToDisplay()
	}
	return bce.qs.communication.ViewBaseConfig(), nil
}

type systemConfigEndpoint struct {
	*endpointEnv
}

func (sce systemConfigEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, server.Error) {
	_, s := sce.qs.data.SystemByID(ids[0])
	if s == nil {
		return nil, &server.NotFound{Name: ids[0]}
	}
	if method == httpPut {
		if err := sce.qs.communication.UpdateSystemConfig(raw, s); err != nil {
			return nil, err
		}
		sce.qs.persistence.WriteSystem(s)
		return nil, sce.sendConfigsToDisplay()
	}
	return sce.qs.communication.ViewSystemConfig(s)
}

type groupConfigEndpoint struct {
	*endpointEnv
}

func (gce groupConfigEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, server.Error) {
	_, g := gce.qs.data.GroupByID(ids[0])
	if g == nil {
		return nil, &server.NotFound{Name: ids[0]}
	}
	if method == httpPut {

		if err := gce.qs.communication.UpdateGroupConfig(raw, g); err != nil {
			return nil, err
		}
		gce.qs.persistence.WriteGroup(g)
		return nil, gce.sendConfigsToDisplay()
	}
	return gce.qs.communication.ViewGroupConfig(g), nil
}

type sceneConfigEndpoint struct {
	*endpointEnv
}

func (sce sceneConfigEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, server.Error) {
	_, g := sce.qs.data.GroupByID(ids[0])
	if g == nil {
		return nil, &server.NotFound{Name: ids[0]}
	}
	_, s := g.SceneByID(ids[1])
	if s == nil {
		return nil, &server.NotFound{Name: ids[1]}
	}
	if method == httpPut {
		if err := sce.qs.communication.UpdateSceneConfig(raw, s); err != nil {
			return nil, err
		}
		sce.qs.persistence.WriteScene(g, s)
		return nil, sce.sendConfigsToDisplay()
	}
	return sce.qs.communication.ViewSceneConfig(s), nil
}

type moduleEndpoint struct {
	*endpointEnv
	moduleIndex   shared.ModuleIndex
	endpointIndex int
	pure          bool
}

func (me *moduleEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, server.Error) {
	state := me.qs.data.State.StateOf(me.moduleIndex)
	if state == nil {
		return nil, &server.BadRequest{
			Message: fmt.Sprintf("module \"%s\" not enabled for current scene",
				me.qs.modules[me.moduleIndex].ID)}
	}

	req, err := me.qs.display.StartRequest(
		me.events.ModuleUpdateID, int32(me.moduleIndex))
	if err != nil {
		return nil, err
	}
	defer req.Close()

	var responseObj, data interface{}
	if me.pure {
		ep := state.(modules.PureEndpointProvider).PureEndpoint(me.endpointIndex)
		responseObj, data, err = ep.Post(raw)
	} else {
		ep := state.(modules.IDEndpointProvider).IDEndpoint(me.endpointIndex)
		responseObj, data, err = ep.Post(ids[0], raw)
	}
	if err != nil {
		return nil, err
	}

	req.SendRendererData(me.moduleIndex, data)
	req.Commit()
	me.qs.persistence.WriteState()
	return responseObj, nil
}

type dataEndpoint struct {
	*endpointEnv
}

func (de dataEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, server.Error) {
	return de.qs.communication.ViewAll(de.qs), nil
}

type systemEndpoint struct {
	*endpointEnv
}

func (se systemEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, server.Error) {
	index, s := se.qs.data.SystemByID(ids[0])
	if s == nil {
		return nil, &server.NotFound{Name: ids[0]}
	}
	if method == httpPut {
		if err := se.qs.communication.UpdateSystem(raw, s); err != nil {
			return nil, err
		}
		if err := se.qs.persistence.WriteSystem(s); err != nil {
			log.Println("failed to persist system: " + err.Error())
		}
		return se.qs.communication.ViewSystems(), nil
	}
	return nil, se.qs.persistence.DeleteSystem(index)
}

type dataGroupEndpoint struct {
	*endpointEnv
}

func (dge dataGroupEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, server.Error) {
	index, g := dge.qs.data.GroupByID(ids[0])
	if g == nil {
		return nil, &server.NotFound{Name: ids[0]}
	}
	if method == httpPut {
		if err := dge.qs.communication.UpdateGroup(raw, g); err != nil {
			return nil, err
		}
		if err := dge.qs.persistence.WriteGroup(g); err != nil {
			log.Println("failed to persist group: " + err.Error())
		}
		// TODO: check group index; if active group, update stuff since index could
		// have been changed due to reordering
	} else {
		dge.qs.persistence.DeleteGroup(index)
	}
	return dge.qs.communication.ViewGroups(), nil
}

type dataSystemsEndpoint struct {
	*endpointEnv
}

func (sc dataSystemsEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, server.Error) {
	name := comms.ValidatedString{MinLen: 1, MaxLen: -1}
	if err := comms.ReceiveData(raw, &name); err != nil {
		return nil, &server.BadRequest{Inner: err, Message: "received invalid data"}
	}
	if err := sc.qs.persistence.CreateSystem(name.Value); err != nil {
		return nil, err
	}
	return sc.qs.communication.ViewSystems(), nil
}

type dataGroupsEndpoint struct {
	*endpointEnv
}

type groupCreationReceiver struct {
	shared.GroupCreationRequest
	plugins []pluginData
}

func (gcr *groupCreationReceiver) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data,
		&comms.ValidatedStruct{Value: &gcr.GroupCreationRequest}); err != nil {
		return err
	}
	if gcr.Name == "" {
		return errors.New("name must not be empty")
	} else if gcr.PluginIndex < 0 ||
		gcr.PluginIndex >= len(gcr.plugins) {
		return fmt.Errorf("pluginIndex out of range [0..%d]", len(gcr.plugins)-1)
	} else if gcr.GroupTemplateIndex < 0 ||
		gcr.GroupTemplateIndex >= len(gcr.plugins[gcr.PluginIndex].GroupTemplates) {
		return fmt.Errorf("groupTemplateIndex out of range [0..%d]",
			len(gcr.plugins[gcr.PluginIndex].GroupTemplates)-1)
	}
	return nil
}

func (dge dataGroupsEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, server.Error) {
	value := groupCreationReceiver{plugins: dge.qs.plugins}
	if err := comms.ReceiveData(raw, &value); err != nil {
		return nil, &server.BadRequest{Inner: err, Message: "received invalid data"}
	}
	if err := dge.qs.persistence.CreateGroup(value.Name,
		&dge.qs.plugins[value.PluginIndex].Plugin.GroupTemplates[value.GroupTemplateIndex],
		dge.qs.plugins[value.PluginIndex].Plugin.SceneTemplates); err != nil {
		return nil, &server.InternalError{
			Description: "while creating group", Inner: err}
	}
	return dge.qs.communication.ViewGroups(), nil
}

type dataScenesEndpoint struct {
	*endpointEnv
}

type sceneCreationReceiver struct {
	shared.SceneCreationRequest
	plugins []pluginData
}

func (scr *sceneCreationReceiver) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data,
		&comms.ValidatedStruct{Value: &scr.SceneCreationRequest}); err != nil {
		return err
	}
	if scr.Name == "" {
		return errors.New("name must not be empty")
	} else if scr.PluginIndex < 0 ||
		scr.PluginIndex >= len(scr.plugins) {
		return fmt.Errorf("pluginIndex out of range [0..%d]", len(scr.plugins)-1)
	} else if scr.SceneTemplateIndex < 0 ||
		scr.SceneTemplateIndex >= len(scr.plugins[scr.PluginIndex].SceneTemplates) {
		return fmt.Errorf("groupTemplateIndex out of range [0..%d]",
			len(scr.plugins[scr.PluginIndex].SceneTemplates)-1)
	}
	return nil
}

func (dse dataScenesEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, server.Error) {
	_, group := dse.qs.data.GroupByID(ids[0])
	if group == nil {
		return nil, &server.NotFound{Name: ids[0]}
	}
	value := sceneCreationReceiver{plugins: dse.qs.plugins}
	if err := comms.ReceiveData(raw, &value); err != nil {
		return nil, &server.BadRequest{Inner: err, Message: "received invalid data"}
	}

	if err := dse.qs.persistence.CreateScene(group, value.Name,
		&dse.qs.plugins[value.PluginIndex].SceneTemplates[value.SceneTemplateIndex]); err != nil {
		return nil, &server.InternalError{
			Description: "while creating scene", Inner: err}
	}
	return dse.qs.communication.ViewScenes(group), nil
}

type dataSceneEndpoint struct {
	*endpointEnv
}

func (dse dataSceneEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, server.Error) {
	_, group := dse.qs.data.GroupByID(ids[0])
	if group == nil {
		return nil, &server.NotFound{Name: ids[0]}
	}
	sceneIndex, scene := group.SceneByID(ids[1])
	if scene == nil {
		return nil, &server.NotFound{Name: ids[1]}
	}
	if method == httpPut {
		if err := dse.qs.communication.UpdateScene(raw, group, scene); err != nil {
			return nil, err
		}
		if err := dse.qs.persistence.WriteScene(group, scene); err != nil {
			log.Println("failed to persist scene: " + err.Error())
		}
		// TODO: check scene index; if active scene, update stuff since index could
		// have been changed due to reordering
	} else {
		dse.qs.persistence.DeleteScene(group, sceneIndex)
	}
	return dse.qs.communication.ViewScenes(group), nil
}

type dataHeroesEndpoint struct {
	*endpointEnv
}

func (dhe dataHeroesEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, server.Error) {
	_, group := dhe.qs.data.GroupByID(ids[0])
	if group == nil {
		return nil, &server.NotFound{Name: ids[0]}
	}
	value := struct {
		Name        comms.ValidatedString `json:"name"`
		Description string                `json:"description"`
	}{
		Name: comms.ValidatedString{MinLen: 1, MaxLen: -1},
	}
	if err := comms.ReceiveData(raw, &value); err != nil {
		return nil, &server.BadRequest{Inner: err, Message: "received invalid data"}
	}
	req, err := dhe.qs.display.StartRequest(dhe.events.HeroesChangedID, 0)
	if err != nil {
		return nil, err
	}
	defer req.Close()
	heroes := group.Heroes()
	if err := dhe.qs.persistence.CreateHero(
		group, heroes, value.Name.Value, value.Description); err != nil {
		return nil, &server.InternalError{Description: "while creating hero", Inner: err}
	}
	propagateHeroesChange(groups.HeroAdded, heroes.NumHeroes()-1, dhe.qs,
		&req)
	req.Commit()
	return dhe.qs.communication.ViewHeroes(heroes), nil
}

type dataHeroEndpoint struct {
	*endpointEnv
}

func (dhe dataHeroEndpoint) Handle(method httpMethods, ids []string,
	raw []byte) (interface{}, server.Error) {
	_, group := dhe.qs.data.GroupByID(ids[0])
	if group == nil {
		return nil, &server.NotFound{Name: ids[0]}
	}
	heroes := group.Heroes()
	heroIndex, hero := heroes.HeroByID(ids[1])
	if hero == nil {
		return nil, &server.NotFound{Name: ids[1]}
	}
	req, err := dhe.qs.display.StartRequest(dhe.events.HeroesChangedID, 0)
	if err != nil {
		return nil, err
	}
	defer req.Close()
	var action groups.HeroChangeAction
	if method == httpPut {
		if err := dhe.qs.communication.UpdateHero(raw, hero); err != nil {
			return nil, err
		}
		if err := dhe.qs.persistence.WriteHero(group, hero); err != nil {
			log.Println("failed to persist hero: " + err.Error())
		}
		action = groups.HeroModified
	} else {
		dhe.qs.persistence.DeleteHero(group, heroes, heroIndex)
		action = groups.HeroDeleted
	}
	propagateHeroesChange(action, heroIndex, dhe.qs, &req)
	req.Commit()

	return dhe.qs.communication.ViewHeroes(heroes), nil
}

func startServer(owner *QuestScreen, events display.Events,
	port uint16) *http.Server {
	server := &http.Server{Addr: ":" + strconv.Itoa(int(port))}
	env := &endpointEnv{qs: owner, events: events}
	mutex := &sync.Mutex{}

	sep := newStaticResourceHandler(owner)
	http.Handle("/", sep)

	reg("StaticDataHandler", "/static", mutex,
		endpoint{httpGet, &staticDataEndpoint{env}})

	// if no fonts are found, QuestScreen is not operable. We only provide static
	// data (telling the client no fonts are available) and the static resources.
	if len(owner.fonts) > 0 {
		reg("StateHandler", "/state", mutex,
			endpoint{httpGet | httpPost, &stateEndpoint{env}})
		reg("BaseConfigHandler", "/config/base", mutex,
			endpoint{httpGet | httpPut, &baseConfigEndpoint{env}})
		reg("SystemConfigHandler", "/config/systems/", mutex,
			idCapture{}, endpoint{httpGet | httpPut, &systemConfigEndpoint{env}})
		reg("GroupConfigHandler", "/config/groups/", mutex,
			idCapture{}, endpoint{httpGet | httpPut, &groupConfigEndpoint{env}},
			pathFragment("scenes"), idCapture{},
			endpoint{httpGet | httpPut, &sceneConfigEndpoint{env}})
		reg("DataHandler", "/data", mutex, endpoint{httpGet, &dataEndpoint{env}})
		reg("DataSystemsHandler", "/data/systems", mutex,
			endpoint{httpPost, &dataSystemsEndpoint{env}})
		reg("DataSystemHandler", "/data/systems/", mutex, idCapture{},
			endpoint{httpPut | httpDelete, &systemEndpoint{env}})
		reg("DataGroupsHandler", "/data/groups", mutex,
			endpoint{httpPost, &dataGroupsEndpoint{env}})
		reg("DataGroupHandler", "/data/groups/", mutex, idCapture{},
			endpoint{httpPut | httpDelete, &dataGroupEndpoint{env}},
			&branch{"scenes"}, endpoint{httpPost, &dataScenesEndpoint{env}},
			idCapture{}, endpoint{httpPut | httpDelete, &dataSceneEndpoint{env}},
			&branch{"heroes"}, endpoint{httpPost, &dataHeroesEndpoint{env}},
			idCapture{}, endpoint{httpPut | httpDelete, &dataHeroEndpoint{env}})

		for i := shared.FirstModule; i < owner.NumModules(); i++ {
			seenSlash := false
			seenOthers := false

			desc := owner.modules[i]
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
					reg("ModuleEndpoint("+path+")", builder.String(), mutex,
						idCapture{}, endpoint{httpPost,
							&moduleEndpoint{endpointEnv: env, moduleIndex: i, endpointIndex: j,
								pure: false}})
				} else {
					reg("ModuleEndpoint("+path+")", builder.String(), mutex,
						endpoint{httpPost,
							&moduleEndpoint{endpointEnv: env, moduleIndex: i, endpointIndex: j,
								pure: true}})
				}
			}
		}
	}

	log.Printf("Listening on port %d\n", port)
	go func() {
		if err := server.ListenAndServe(); err != nil {
			if err != http.ErrServerClosed {
				log.Fatal(err)
			}
		}
	}()

	return server
}
