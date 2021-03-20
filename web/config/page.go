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
type BaseView struct{}

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

func genView(url string, ctx server.Context) *view {
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
			ui := newItem(ctrl, wasEnabled)
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
	return genView("config/base", ctx)
}

// SystemView shows the configuration of a given system.
type SystemView struct {
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
	return genView("config/system/"+web.Data.Systems[sv.systemIndex].ID, ctx)
}

// GroupView shows the configuration of the given group.
type GroupView struct {
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
	return genView("config/group/"+web.Data.Groups[gv.groupIndex].ID, ctx)
}

// SceneView shows the configuration of the given scene.
type SceneView struct {
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
	return genView("config/system/"+
		web.Data.Groups[sv.groupIndex].Scenes[sv.sceneIndex].ID, ctx)
}

// Page is the controller for the Configuration page and implements site.Page.
type Page struct{}

// Title returns "Datasets"
func (p Page) Title() string {
	return "Configuration"
}

// GenViews implements site.Page
func (p Page) GenViews() []site.ViewCollection {
	ret := make([]site.ViewCollection, 3)
	ret[0].Items = []site.View{BaseView{}}

	systemItems := make([]site.View, len(web.Data.Systems))
	for index := range web.Data.Systems {
		systemItems[index] = &SystemView{systemIndex: index}
	}
	ret[1].Title = "Systems"
	ret[1].Items = systemItems

	groupItems := make([]site.View, 0, len(web.Data.Groups)*4)
	for index, g := range web.Data.Groups {
		groupItems = append(groupItems, &GroupView{groupIndex: index})
		for sIndex := range g.Scenes {
			groupItems = append(groupItems, &SceneView{groupIndex: index, sceneIndex: sIndex})
		}
	}
	ret[2].Title = "Groups"
	ret[2].Items = groupItems
	return ret
}

// Register registers this page with the site.
func Register() {
	site.RegisterPage(site.ConfigPage, &Page{})
}
