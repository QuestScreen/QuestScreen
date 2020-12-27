package info

import "github.com/QuestScreen/QuestScreen/web"

// ConstructInfoPage constructs an info page filled with static data.
func ConstructInfoPage() *Page {
	ret := NewPage(web.StaticData.AppVersion)
	for _, module := range web.StaticData.Modules {
		ret.Modules.Append(NewModule(
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
		ret.Messages.Set(container)
	}

	return ret
}
