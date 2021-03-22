package config

import (
	"bytes"
	"encoding/json"

	"github.com/QuestScreen/QuestScreen/web"
	"github.com/QuestScreen/QuestScreen/web/comms"
	"github.com/QuestScreen/QuestScreen/web/site"
	"github.com/QuestScreen/api/server"
	api "github.com/QuestScreen/api/web"
	askew "github.com/flyx/askew/runtime"
)

// BaseView shows the base configuration for all modules.
type BaseView struct {
	*Page
}

// Title implements site.View, returns "Base"
func (bv BaseView) Title() string {
	return "Base"
}

// ID implements site.View, returns "b-base"
func (bv BaseView) ID() string {
	return "b-base"
}

// IsChild implements site.View, returns false
func (bv BaseView) IsChild() bool {
	return false
}

func genView(url string, ctx server.Context, p *Page) *view {
	var data [][]json.RawMessage
	if err := comms.Fetch(api.Get, url, nil, &data); err != nil {
		panic(err)
	}
	ret := newView()
	for i := range web.StaticData.Modules {
		m := &web.StaticData.Modules[i]
		mView := newModule(m.Name)
		mData := data[i]
		for j, item := range m.ConfigItems {
			wasEnabled := !bytes.HasPrefix(mData[j], []byte("null"))
			ctrl := item.Constructor(ctx)
			ui := newItem(ctrl, item.Name, wasEnabled, p)
			ctrl.SetEditHandler(ui)
			if wasEnabled {
				ctrl.Receive(mData[j], ctx)
				ui.enabled.Set(true)
			} else {
				ui.enabled.Set(false)
			}
			mView.items.Append(ui)
		}

		ret.modules.Append(mView)
	}
	return ret
}

// GenerateUI implements site.View, returns the UI for changing module base
// configuration.
func (bv BaseView) GenerateUI(ctx server.Context) askew.Component {
	bv.curID = bv.ID()
	bv.curUrl = "config/base"
	bv.curView = genView(bv.curUrl, ctx, bv.Page)
	return bv.curView
}

// SystemView shows the configuration of a given system.
type SystemView struct {
	*Page
	systemIndex int
}

// Title implements site.View, returns the system's name prefixed with "System "
func (sv *SystemView) Title() string {
	return web.Data.Systems[sv.systemIndex].Name
}

// ID implements site.View, returns the system's ID prefixed with "s-"
func (sv *SystemView) ID() string {
	return "s-" + web.Data.Systems[sv.systemIndex].ID
}

// IsChild implements site.View, returns false
func (sv *SystemView) IsChild() bool {
	return false
}

// GenerateUI implements site.View, returns the UI for changing module system
// configuration.
func (sv *SystemView) GenerateUI(ctx server.Context) askew.Component {
	sv.curID = sv.ID()
	sv.curUrl = "config/systems/" + web.Data.Systems[sv.systemIndex].ID
	sv.curView = genView(sv.curUrl, ctx, sv.Page)
	return sv.curView
}

// GroupView shows the configuration of the given group.
type GroupView struct {
	*Page
	groupIndex int
}

// Title implements site.View, returns the group's name prefixed with "Group "
func (gv *GroupView) Title() string {
	return web.Data.Groups[gv.groupIndex].Name
}

// ID implements site.View, returns the group's ID prefixed with "g-"
func (gv *GroupView) ID() string {
	return "g-" + web.Data.Groups[gv.groupIndex].ID
}

// IsChild implements site.View, returns false
func (gv *GroupView) IsChild() bool {
	return false
}

// GenerateUI implements site.View, returns the UI for changing module system
// configuration.
func (gv *GroupView) GenerateUI(ctx server.Context) askew.Component {
	gv.curID = gv.ID()
	gv.curUrl = "config/groups/" + web.Data.Groups[gv.groupIndex].ID
	gv.curView = genView(gv.curUrl, ctx, gv.Page)
	return gv.curView
}

// SceneView shows the configuration of the given scene.
type SceneView struct {
	*Page
	groupIndex, sceneIndex int
}

// Title implements site.View, returns the scene's name prefixed with "System "
func (sv *SceneView) Title() string {
	return web.Data.Groups[sv.groupIndex].Scenes[sv.sceneIndex].Name
}

// ID implements site.View, returns the scene's ID prefixed with "gs-"
func (sv *SceneView) ID() string {
	return "gs-" + web.Data.Groups[sv.groupIndex].Scenes[sv.sceneIndex].ID
}

// IsChild implements site.View, returns true
func (sv *SceneView) IsChild() bool {
	return true
}

// GenerateUI implements site.View, returns the UI for changing module system
// configuration.
func (sv *SceneView) GenerateUI(ctx server.Context) askew.Component {
	g := &web.Data.Groups[sv.groupIndex]
	sv.curID = sv.ID()
	sv.curUrl = "config/groups/" + g.ID + "/scenes/" + g.Scenes[sv.sceneIndex].ID
	sv.curView = genView(sv.curUrl, ctx, sv.Page)
	return sv.curView
}

// Page is the controller for the Configuration page and implements site.Page.
type Page struct {
	site.PageEditHandler
	curView       *view
	curID, curUrl string
}

// Title returns "Datasets"
func (p Page) Title() string {
	return "Configuration"
}

func (p *Page) RegisterEditHandler(handler site.PageEditHandler) {
	p.PageEditHandler = handler
}

func (p *Page) Commit() {
	data := make([][]interface{}, len(web.StaticData.Modules))
	for i := 0; i < len(web.StaticData.Modules); i++ {
		m := web.StaticData.Modules[i]
		mView := p.curView.modules.Item(i)
		data[i] = make([]interface{}, len(m.ConfigItems))
		for j := 0; j < len(m.ConfigItems); j++ {
			if mView.items.Item(j).enabled.Get() {
				data[i][j] = mView.items.Item(j).content.Send(
					&comms.ServerState{site.State(), ""})
			}
		}
	}
	if err := comms.Fetch(api.Put, p.curUrl, data, nil); err != nil {
		panic(err)
	}
	site.Refresh(p.curID)
}

func (p *Page) Reset() {
	site.Refresh(p.curID)
}

// GenViews implements site.Page
func (p *Page) GenViews() []site.ViewCollection {
	ret := make([]site.ViewCollection, 3)
	ret[0].Items = []site.View{BaseView{Page: p}}

	systemItems := make([]site.View, len(web.Data.Systems))
	for index := range web.Data.Systems {
		systemItems[index] = &SystemView{Page: p, systemIndex: index}
	}
	ret[1].Title = "Systems"
	ret[1].Items = systemItems

	groupItems := make([]site.View, 0, len(web.Data.Groups)*4)
	for index, g := range web.Data.Groups {
		groupItems = append(groupItems, &GroupView{Page: p, groupIndex: index})
		for sIndex := range g.Scenes {
			groupItems = append(groupItems, &SceneView{Page: p, groupIndex: index, sceneIndex: sIndex})
		}
	}
	ret[2].Title = "Groups"
	ret[2].Items = groupItems
	return ret
}

func (p *Page) updateEdited(force bool) {
	if force {
		p.PageEditHandler.SetEdited(true)
	} else {
		for i := 0; i < p.curView.modules.Len(); i++ {
			m := p.curView.modules.Item(i)
			for j := 0; j < m.items.Len(); j++ {
				item := m.items.Item(j)
				if item.wasEnabled != item.enabled.Get() ||
					(item.wasEnabled && item.valuesEdited) {
					p.PageEditHandler.SetEdited(true)
					return
				}
			}
		}
		p.PageEditHandler.SetEdited(false)
	}
}

// Register registers this page with the site.
func Register() {
	site.RegisterPage(site.ConfigPage, &Page{})
}
