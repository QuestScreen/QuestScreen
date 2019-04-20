package module

import "html/template"

type Module interface {
	Init(common *SceneCommon) error
	Render(common *SceneCommon)
	Name() string
	UI() template.HTML
}
