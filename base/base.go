package modules

import (
	"github.com/QuestScreen/QuestScreen/api"
	"github.com/QuestScreen/QuestScreen/base/background"
	"github.com/QuestScreen/QuestScreen/base/herolist"
	"github.com/QuestScreen/QuestScreen/base/overlays"
	"github.com/QuestScreen/QuestScreen/base/title"
	"github.com/QuestScreen/QuestScreen/generated"
)

// Base is a plugin providing the most common system-independent modules.
var Base = api.Plugin{
	Name: "Base",
	Modules: []*api.Module{
		&background.Descriptor, &herolist.Descriptor, &overlays.Descriptor,
		&title.Descriptor},
	AdditionalJS:    generated.MustAsset("web/js/base.js"),
	AdditionalHTML:  generated.MustAsset("web/html/base.html"),
	AdditionalCSS:   nil,
	SystemTemplates: nil,
	GroupTemplates: []api.GroupTemplate{
		{
			Name: "Minimal", Description: "Contains a „Main“ scene with no modules.",
			Config: []byte("name: Minimal"),
			Scenes: []api.SceneTmplRef{
				{Name: "Main", TmplIndex: 0},
			},
		}, {
			Name:        "Base",
			Description: "Contains a „Main“ scene with base modules.",
			Config:      []byte("name: Base"),
			Scenes: []api.SceneTmplRef{
				{Name: "Main", TmplIndex: 1},
			},
		},
	},
	SceneTemplates: []api.SceneTemplate{
		{
			Name: "Empty", Description: "A scene with no modules.",
			Config: []byte("name: Empty"),
		}, {
			Name:        "BaseMain",
			Description: "A scene with background, title, herolist and overlay enabled.",
			Config: []byte(`name: BaseMain
modules:
  background:
    enabled: true
  herolist:
    enabled: true
  overlays:
    enabled: true
  title:
    enabled: true`),
		},
	},
}
