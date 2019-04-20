package background_image

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/flyx/rpscreen/pkg/module"
)

type Background struct {
	texture module.Texture
}

func (me *Background) Init(common *module.SceneCommon) error {
	var err error
	me.texture, err = module.LoadTextureFromFile("Kerker.png")
	return err
}

func (me *Background) Render(common *module.SceneCommon) {
	model := mgl32.Ident4()
	if me.texture.Ratio > common.Ratio {
		model = mgl32.Scale3D(1, common.Ratio/me.texture.Ratio, 1)
	} else {
		model = mgl32.Scale3D(me.texture.Ratio/common.Ratio, 1, 1)
	}
	common.DrawSquare(me.texture.GlId, model)
}

func (*Background) Name() string {
	return "Background Image"
}
