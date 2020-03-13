package background

import (
	"log"
	"time"

	"github.com/QuestScreen/QuestScreen/api"

	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
)

type backgroundConfig struct{}

type request struct {
	file api.Resource
}

// Background is a module for painting background images
type Background struct {
	config                 *backgroundConfig
	curTexture, newTexture *sdl.Texture
	curFile                api.Resource
}

func newRenderer(backend *sdl.Renderer,
	ms api.MessageSender) (api.ModuleRenderer, error) {
	bg := new(Background)
	bg.curTexture = nil
	bg.newTexture = nil
	bg.curFile = nil
	return bg, nil
}

// Descriptor describes the Background module.
var Descriptor = api.Module{
	Name: "Background Image",
	ID:   "background",
	ResourceCollections: []api.ResourceSelector{
		api.ResourceSelector{Subdirectory: "", Suffixes: nil}},
	EndpointPaths:  []string{""},
	DefaultConfig:  &backgroundConfig{},
	CreateRenderer: newRenderer,
	CreateState:    newState,
}

// Descriptor returns the Background's descriptor
func (bg *Background) Descriptor() *api.Module {
	return &Descriptor
}

func offsets(inRatio float32, outRatio float32,
	winWidth int32, winHeight int32) sdl.Rect {
	if inRatio > outRatio {
		return sdl.Rect{
			X: 0, Y: int32(float32(winHeight) * (1.0 - (outRatio / inRatio)) / 2.0),
			W: winWidth, H: int32(float32(winHeight) * (outRatio / inRatio))}
	}
	return sdl.Rect{
		X: int32(float32(winWidth) * (1.0 - (inRatio / outRatio)) / 2.0),
		Y: 0, W: int32(float32(winWidth) * (inRatio / outRatio)), H: winHeight}
}

func (bg *Background) genTexture(
	renderer *sdl.Renderer, file api.Resource) *sdl.Texture {
	tex, err := img.LoadTexture(renderer, file.Path())
	if err != nil {
		log.Println(err)
		return nil
	}
	defer tex.Destroy()
	_, _, texWidth, texHeight, err := tex.Query()
	if err != nil {
		panic(err)
	}
	winWidth, winHeight, _ := renderer.GetOutputSize()
	newTexture, err := renderer.CreateTexture(
		sdl.PIXELFORMAT_RGB888, sdl.TEXTUREACCESS_TARGET, winWidth, winHeight)
	if err != nil {
		panic(err)
	}
	renderer.SetRenderTarget(newTexture)
	defer renderer.SetRenderTarget(nil)
	renderer.Clear()
	renderer.SetDrawColor(0, 0, 0, 255)
	renderer.FillRect(nil)
	dst := offsets(float32(texWidth)/float32(texHeight),
		float32(winWidth)/float32(winHeight), winWidth, winHeight)
	renderer.Copy(tex, nil, &dst)
	return newTexture
}

// InitTransition initializes a transition
func (bg *Background) InitTransition(ctx api.RenderContext, data interface{}) time.Duration {
	var ret time.Duration = -1
	req := data.(*request)

	if req.file != bg.curFile {
		bg.curFile = req.file
		if bg.curFile != nil {
			bg.newTexture = bg.genTexture(ctx.Renderer(), bg.curFile)
			if err := bg.newTexture.SetBlendMode(sdl.BLENDMODE_BLEND); err != nil {
				log.Println(err)
			}
		}
		ret = time.Second
	}
	return ret
}

// TransitionStep advances the transition.
func (bg *Background) TransitionStep(
	ctx api.RenderContext, elapsed time.Duration) {
	if bg.newTexture != nil {
		if err := bg.newTexture.SetAlphaMod(uint8((elapsed * 255) / time.Second)); err != nil {
			log.Println(err)
		}
	}
}

// FinishTransition finalizes the transition.
func (bg *Background) FinishTransition(ctx api.RenderContext) {
	if bg.curTexture != nil {
		bg.curTexture.Destroy()
	}
	bg.curTexture = bg.newTexture
	bg.newTexture = nil
	if bg.curTexture != nil {
		if err := bg.curTexture.SetBlendMode(sdl.BLENDMODE_NONE); err != nil {
			log.Println(err)
		}
		if err := bg.curTexture.SetAlphaMod(255); err != nil {
			log.Println(err)
		}
	}
}

// Render renders the module
func (bg *Background) Render(ctx api.RenderContext) {
	if bg.curTexture != nil {
		ctx.Renderer().Copy(bg.curTexture, nil, nil)
	}
	if bg.newTexture != nil {
		ctx.Renderer().Copy(bg.newTexture, nil, nil)
	}
}

// EmptyConfig returns an empty configuration
func (*Background) EmptyConfig() interface{} {
	return &backgroundConfig{}
}

// SetConfig sets the module's configuration
func (bg *Background) SetConfig(config interface{}) {
	bg.config = config.(*backgroundConfig)
}

// RebuildState queries the texture index through the channel and immediately
// sets that texture as background.
func (bg *Background) RebuildState(
	ctx api.ExtendedRenderContext, data interface{}) {
	if data != nil {
		req := data.(*request)
		if req.file != bg.curFile {
			if bg.curTexture != nil {
				bg.curTexture.Destroy()
			}
			bg.curFile = req.file
			if bg.curFile != nil {
				bg.curTexture = bg.genTexture(ctx.Renderer(), bg.curFile)
			} else {
				bg.curTexture = nil
			}
		}
		if bg.newTexture != nil {
			bg.newTexture.Destroy()
			bg.newTexture = nil
		}
	}
}
