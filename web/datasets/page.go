package datasets

import (
	"github.com/QuestScreen/QuestScreen/web"
	"github.com/QuestScreen/QuestScreen/web/site"
	"github.com/QuestScreen/api/server"
	askew "github.com/flyx/askew/runtime"
)

// BaseView is the default view of the dataset page.
type BaseView struct{}

// Title returns "Base"
func (bv BaseView) Title() string {
	return "Base"
}

// ID returns "b-base"
func (bv BaseView) ID() string {
	return "b-base"
}

// IsChild returns false
func (bv BaseView) IsChild() bool {
	return false
}

// SwitchTo implements site.View
func (bv BaseView) SwitchTo(ctx server.Context) askew.Component {
	return newBase()
}

// SystemView is the view of a selected system.
type SystemView struct {
	systemIndex int
}

// Title returns the system's name
func (sv *SystemView) Title() string {
	return web.Data.Systems[sv.systemIndex].Name
}

// ID returns the system's id, prefixed by "s-"
func (sv *SystemView) ID() string {
	return "s-" + web.Data.Systems[sv.systemIndex].ID
}

// IsChild returns false
func (sv *SystemView) IsChild() bool {
	return false
}

// SwitchTo implements site.View
func (sv *SystemView) SwitchTo(ctx server.Context) askew.Component {
	s := &web.Data.Systems[sv.systemIndex]
	return newSystem(s)
}

// GroupView is the view of the selected group.
type GroupView struct {
	groupIndex int
}

// Title returns the group's name.
func (gv *GroupView) Title() string {
	return web.Data.Groups[gv.groupIndex].Name
}

// ID returns the group's ID, prefixed by "g-"
func (gv *GroupView) ID() string {
	return "g-" + web.Data.Groups[gv.groupIndex].ID
}

// IsChild returns false
func (gv *GroupView) IsChild() bool {
	return false
}

// SwitchTo implements site.View
func (gv *GroupView) SwitchTo(ctx server.Context) askew.Component {
	g := &web.Data.Groups[gv.groupIndex]
	return newGroup(g)
}

// SceneView is the view of the selected scene.
type SceneView struct {
	groupIndex, sceneIndex int
}

// Title returns the scene's name.
func (sv *SceneView) Title() string {
	return web.Data.Groups[sv.groupIndex].Scenes[sv.sceneIndex].Name
}

// ID returns the scene's ID, prefixed by "gs-"
func (sv *SceneView) ID() string {
	return "gs-" + web.Data.Groups[sv.groupIndex].Scenes[sv.sceneIndex].ID
}

// IsChild returns true.
func (sv *SceneView) IsChild() bool {
	return true
}

// SwitchTo implements site.View.
func (sv *SceneView) SwitchTo(ctx server.Context) askew.Component {
	g := &web.Data.Groups[sv.groupIndex]
	return newScene(g, sv.sceneIndex)
}

// Page is the controller for the Datasets page and implements site.Page.
type Page struct{}

// Title returns "Datasets"
func (p Page) Title() string {
	return "Datasets"
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

func (Page) IconOffset() int {
	return 1
}

// Register registers this page with the site.
func Register() {
	site.RegisterPage(site.DataPage, &Page{})
}
