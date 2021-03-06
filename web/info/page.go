package info

import (
	"github.com/QuestScreen/QuestScreen/web"
	"github.com/QuestScreen/QuestScreen/web/site"
	"github.com/QuestScreen/api/server"
	askew "github.com/flyx/askew/runtime"
)

// View implements site.View
type View struct{}

// Title returns "Home"
func (v View) Title() string {
	return "Home"
}

// ID returns "home"
func (v View) ID() string {
	return "home"
}

// SwitchTo creates the home view.
func (v View) SwitchTo(ctx server.Context) askew.Component {
	return newViewContent(web.StaticData.AppVersion)
}

// IsChild returns false.
func (v View) IsChild() bool {
	return false
}

// Page implements site.Page
type Page struct{}

// Title returns "Home"
func (Page) Title() string {
	return "Home"
}

// GenViews returns a view list containing the only view of the home page.
func (Page) GenViews() []site.ViewCollection {
	return []site.ViewCollection{{Title: "", Items: []site.View{View{}}}}
}

func (Page) IconOffset() int {
	return 1
}

// Register registers this page with the site.
func Register() {
	site.RegisterPage(site.InfoPage, &Page{})
}

func newViewContent(version string) *viewContent {
	ret := new(viewContent)
	ret.init(version)
	return ret
}

func (c *viewContent) init(version string) {
	state := 0
	if len(web.Data.Groups) > 0 {
		state = 1
	}
	if len(web.StaticData.Messages) > 0 {
		state = 2
	}
	c.askewInit(version, state)
	for i, group := range web.Data.Groups {
		c.groups.Append(NewChooseableGroup(group.Name, i))
	}
	for _, module := range web.StaticData.Modules {
		p := web.StaticData.Plugins[module.PluginIndex]
		c.Modules.Append(NewModule(
			p.Name, p.ID, module.Name, module.ID))
	}
	if len(web.StaticData.Messages) > 0 {
		container := NewMessageContainer()
		for _, msg := range web.StaticData.Messages {
			if msg.ModuleIndex == -1 {
				container.Items.Append(
					NewMessage("", "", "", msg.Text, msg.IsError))
			} else {
				mod := &web.StaticData.Modules[msg.ModuleIndex]
				container.Items.Append(
					NewMessage("unknown", mod.Name, mod.ID, msg.Text, msg.IsError))
			}
		}
		c.Messages.Set(container)
	}
}
