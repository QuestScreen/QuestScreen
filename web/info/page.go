package info

import (
	"github.com/QuestScreen/QuestScreen/web"
	"github.com/QuestScreen/QuestScreen/web/site"
	"github.com/flyx/askew/runtime"
)

// View implements info.View
type View struct{}

// Title returns "Info"
func (v View) Title() string {
	return "Info"
}

// ID returns "info"
func (v View) ID() string {
	return "info"
}

// GenerateUI creates the info view.
func (v View) GenerateUI() runtime.Component {
	return newViewContent(web.StaticData.AppVersion)
}

// IsChild returns false.
func (v View) IsChild() bool {
	return false
}

// Page implements info.Page
type Page struct{}

// Title returns "Info"
func (p Page) Title() string {
	return "Info"
}

// BackButton returns NoBackButton.
func (p Page) BackButton() site.BackButtonKind {
	return site.NoBackButton
}

// GenViews returns a view list containing the only view of the info page.
func (p Page) GenViews() []site.ViewCollection {
	return []site.ViewCollection{{Title: "", Items: []site.View{View{}}}}
}

// Register registers this page with the site.
func Register() {
	site.RegisterPage(site.InfoPage, &Page{})
}

func (c *viewContent) init(version string) {
	for _, module := range web.StaticData.Modules {
		c.Modules.Append(NewModule(
			web.StaticData.Plugins[module.PluginIndex].Name, module.Name, module.ID))
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
