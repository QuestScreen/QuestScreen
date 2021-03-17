package session

import (
	"errors"

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
	return newViewContent(v.modules)
}

// Page implements site.Page
type Page struct {
	*shared.State
	modules []modules.State
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
	for i := range data.Modules {
		descr := &web.StaticData.Modules[i]
		server := &comms.ServerState{State: p.State, Base: descr.BasePath()}
		var err error
		p.modules[i], err = descr.Constructor(data.Modules[i], server)
		if err != nil {
			return errors.New("invalid data for module " +
				web.StaticData.Modules[i].Name + ": " + err.Error())
		}
	}
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
	p.modules = make([]modules.State, len(web.StaticData.Modules))
	site.RegisterPage(site.SessionPage, &p)
	p.State = site.State()
}
