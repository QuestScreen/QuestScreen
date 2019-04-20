package module

type Module interface {
	Init(common *SceneCommon) error
	Render(common *SceneCommon)
	Name() string
}
