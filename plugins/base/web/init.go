package web

import (
	background "github.com/QuestScreen/QuestScreen/plugins/base/background/web"
	herolist "github.com/QuestScreen/QuestScreen/plugins/base/herolist/web"
	overlays "github.com/QuestScreen/QuestScreen/plugins/base/overlays/web"
	title "github.com/QuestScreen/QuestScreen/plugins/base/title/web"
	"github.com/QuestScreen/api/web"
)

// InitPluginWebUI initializes the plugin's modules for the Web UI.
func InitPluginWebUI(reg web.PluginRegistrator) {
	reg.RegisterModule("background", background.NewState)
	reg.RegisterModule("herolist", herolist.NewState)
	reg.RegisterModule("overlays", overlays.NewState)
	reg.RegisterModule("title", title.NewState)
}
