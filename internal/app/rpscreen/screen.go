package rpscreen

import (
	"fmt"
	"github.com/flyx/egl"
	"github.com/flyx/egl/platform"
	gl "github.com/remogatto/opengles2"
	"github.com/flyx/rpscreen/internal/pkg/background-image"
	"github.com/flyx/rpscreen/pkg/module"
)

const TexCoordMax = 1

type moduleListItem struct {
	module  module.Module
	enabled bool
}

type Screen struct {
	module.SceneCommon
	textureBuffer uint32
	modules       []moduleListItem
}

func NewRenderState(eglState *platform.EGLState) (*Screen, error) {
	if ok := egl.MakeCurrent(eglState.Display, eglState.Surface, eglState.Surface, eglState.Context); !ok {
		return nil, egl.NewError(egl.GetError())
	}

	screen := new(Screen)
	screen.modules = make([]moduleListItem, 16)

	var width, height int32
	egl.QuerySurface(eglState.Display, eglState.Surface, egl.WIDTH, &width)
	egl.QuerySurface(eglState.Display, eglState.Surface, egl.HEIGHT, &height)
	screen.Ratio = float32(width) / float32(height)

	screen.Square.Vertices = module.CreateFloatBuffer([]float32{
		1, -1, 1, 1, TexCoordMax, TexCoordMax,
		1, 1, 1, 1, TexCoordMax, 0,
		-1, 1, 1, 1, 0, 0,
		-1, -1, 1, 1, 0, TexCoordMax,
	})
	screen.Square.Indices = module.CreateByteBuffer([]byte{
		0, 1, 2,
		2, 3, 0,
	})

	fragmentShader := (module.FragmentShader)(`
			#version 101
			precision mediump float;
			uniform sampler2D tx;
			varying vec2 texOut;
			void main() {
				gl_FragColor = texture2D(tx, texOut);
				//gl_FragColor = vec4(1,0,0,1);
			}
        `)
	vertexShader := (module.VertexShader)(`
 				#version 101
				precision mediump float;
        uniform mat4 model;
        uniform mat4 projection_view;
        attribute vec4 pos;
        attribute vec2 texIn;
        varying vec2 texOut;
        void main() {
          gl_Position = projection_view*model*pos;
          texOut = texIn;
        }
        `)

	fsh := fragmentShader.Compile()
	vsh := vertexShader.Compile()
	screen.TextureRenderProgram.GlId = module.CreateProgram(fsh, vsh)

	screen.TextureRenderProgram.AttributeIds.Pos = gl.GetAttribLocation(screen.TextureRenderProgram.GlId, "pos")
	screen.TextureRenderProgram.AttributeIds.Color = gl.GetAttribLocation(screen.TextureRenderProgram.GlId, "color")
	screen.TextureRenderProgram.AttributeIds.TexIn = gl.GetAttribLocation(screen.TextureRenderProgram.GlId, "texIn")

	screen.TextureRenderProgram.UniformIds.Texture =
		gl.GetUniformLocation(screen.TextureRenderProgram.GlId, "texture")
	screen.TextureRenderProgram.UniformIds.Model = gl.GetUniformLocation(screen.TextureRenderProgram.GlId, "model")
	screen.TextureRenderProgram.UniformIds.ProjectionView =
		gl.GetUniformLocation(screen.TextureRenderProgram.GlId, "projection_view")

	background := new(background_image.Background)
	if err := background.Init(&screen.SceneCommon); err != nil {
		panic(err)
	}
	screen.modules = append(screen.modules, moduleListItem{module: background, enabled: true})
	return screen, nil
}

func (s *Screen) Render() {
	fmt.Println("rendering...")
	for _, item := range s.modules {
		if item.enabled {
			fmt.Println("rendering ", item.module.Name())
			item.module.Render(&s.SceneCommon)
		}
	}
	gl.Flush()
	gl.Finish()
}
