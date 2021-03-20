package session

import (
	"encoding/json"

	"github.com/QuestScreen/QuestScreen/shared"
	"github.com/QuestScreen/QuestScreen/web"
	"github.com/QuestScreen/QuestScreen/web/comms"
	"github.com/QuestScreen/QuestScreen/web/site"
	"github.com/QuestScreen/api/server"
	api "github.com/QuestScreen/api/web"
	"github.com/QuestScreen/api/web/modules"
	askew "github.com/flyx/askew/runtime"
)

type View struct {
	*Page
	scene *shared.Scene
}

func (v View) Title() string {
	return v.scene.Name
}

func (v View) ID() string {
	return v.scene.ID
}

func (View) IsChild() bool {
	return false
}

func (v View) GenerateUI(ctx server.Context) askew.Component {
	modules := make([]modules.State, len(v.modules))
	for i := range v.modules {
		descr := &web.StaticData.Modules[i]
		server := &comms.ServerState{State: p.State, Base: descr.BasePath()}
		var err error
		modules[i], err = descr.Constructor(v.modules[i], server)
		if err != nil {
			panic("invalid data for module " +
				web.StaticData.Modules[i].Name + ": " + err.Error())
		}
	}

	return newViewContent(modules)
}

// Page implements site.EndablePage
type Page struct {
	*shared.State
	modules []json.RawMessage
}

func (p *Page) End() {
	// TODO
}

func (p *Page) Title() string {
	return "Session: " + web.Data.Groups[p.ActiveGroup].Name
}

func (p *Page) GenViews() []site.ViewCollection {
	ret := make([]site.ViewCollection, 1)
	scenes := web.Data.Groups[p.ActiveGroup].Scenes
	ret[0].Title = "Scenes"
	ret[0].Items = make([]site.View, 0, len(scenes))
	for i := range scenes {
		ret[0].Items = append(ret[0].Items, View{p, &scenes[i]})
	}
	return ret
}

func (p *Page) loadState(data *shared.StateResponse) error {
	site.UpdateSession(data.ActiveGroup, data.ActiveScene)
	p.modules = data.Modules
	return nil
}

var p Page

func StartSession(groupIndex int) {
	var state shared.StateResponse
	if err := comms.Fetch(api.Post, "/state", shared.StateRequest{
		Action: "setgroup", Index: groupIndex}, &state); err != nil {
		panic(err)
	}
	p.loadState(&state)
	site.ShowHome()
}

func CheckSession() {
	var state shared.StateResponse
	if err := comms.Fetch(api.Get, "/state", nil, &state); err != nil {
		panic(err)
	}
	if state.ActiveGroup == -1 {
		site.UpdateSession(-1, -1)
	} else {
		p.loadState(&state)
	}
	site.ShowHome()
}

// Register registers this page with the site.
func Register() {
	site.RegisterPage(site.SessionPage, &p)
	p.State = site.State()
}
