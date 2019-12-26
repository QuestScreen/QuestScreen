package background

import (
	"log"
	"sync"
	"time"

	"github.com/flyx/pnpscreen/api"
	"gopkg.in/yaml.v3"

	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
)

type backgroundConfig struct{}

type sharedData struct {
	moduleIndex   api.ModuleIndex // read-only, therefore not protected with mutex
	mutex         sync.Mutex
	activeRequest bool
	file          api.Resource
}

// Background is a module for painting background images
type Background struct {
	sharedData
	config                 *backgroundConfig
	curTexture, newTexture *sdl.Texture
	curFile                api.Resource
	curTextureSplit        float32
}

// CreateModule creates a new Background module
func CreateModule(renderer *sdl.Renderer,
	env api.StaticEnvironment, index api.ModuleIndex) (api.Module, error) {
	bg := new(Background)
	bg.moduleIndex = index
	bg.curTexture = nil
	bg.newTexture = nil
	bg.curTextureSplit = 0
	bg.curFile = nil
	return bg, nil
}

// Descriptor describes the Background module.
var Descriptor = api.ModuleDescriptor{
	Name: "Background Image",
	ID:   "background",
	ResourceCollections: []api.ResourceSelector{
		api.ResourceSelector{Subdirectory: "", Suffixes: nil}},
	Actions:       []string{"set"},
	DefaultConfig: &backgroundConfig{},
	CreateModule:  CreateModule,
}

// Descriptor returns the Background's descriptor
func (bg *Background) Descriptor() *api.ModuleDescriptor {
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
func (bg *Background) InitTransition(ctx api.RenderContext) time.Duration {
	var ret time.Duration = -1

	bg.sharedData.mutex.Lock()
	reqFile, available := bg.sharedData.file, bg.sharedData.activeRequest
	bg.sharedData.activeRequest = false
	bg.sharedData.mutex.Unlock()

	if available && reqFile != bg.curFile {
		bg.curFile = reqFile
		if bg.curFile != nil {
			bg.newTexture = bg.genTexture(ctx.Renderer, bg.curFile)
		}
		bg.curTextureSplit = 0
		ret = time.Second
	} else {
		log.Println("background transition skiping, available=", available,
			", reqIndex=", reqFile)
	}
	return ret
}

// TransitionStep advances the transition.
func (bg *Background) TransitionStep(
	ctx api.RenderContext, elapsed time.Duration) {
	bg.curTextureSplit = float32(elapsed) / float32(time.Second)
}

// FinishTransition finalizes the transition.
func (bg *Background) FinishTransition(ctx api.RenderContext) {
	if bg.curTexture != nil {
		bg.curTexture.Destroy()
	}
	bg.curTexture = bg.newTexture
	bg.curTextureSplit = 0.0
	bg.newTexture = nil
}

// Render renders the module
func (bg *Background) Render(ctx api.RenderContext) {
	if bg.curTexture != nil || bg.curTextureSplit != 0 {
		winWidth, winHeight, _ := ctx.Renderer.GetOutputSize()
		curSplit := int32(bg.curTextureSplit * float32(winWidth))
		if bg.curTexture != nil {
			rect := sdl.Rect{X: curSplit, Y: 0, W: winWidth - curSplit, H: winHeight}
			ctx.Renderer.Copy(bg.curTexture, &rect, &rect)
		}
		if bg.newTexture != nil {
			rect := sdl.Rect{X: 0, Y: 0, W: curSplit, H: winHeight}
			ctx.Renderer.Copy(bg.newTexture, &rect, &rect)
		}
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

// CreateState returns the current state.
func (bg *Background) CreateState(
	input *yaml.Node, env api.Environment) (api.ModuleState, error) {
	return newState(input, env, &bg.sharedData)
}

// RebuildState queries the texture index through the channel and immediately
// sets that texture as background.
func (bg *Background) RebuildState(ctx api.RenderContext) {
	bg.sharedData.mutex.Lock()
	reqFile, available := bg.sharedData.file, bg.sharedData.activeRequest
	bg.activeRequest = false
	bg.sharedData.mutex.Unlock()

	if available {
		if reqFile != bg.curFile {
			if bg.curTexture != nil {
				bg.curTexture.Destroy()
			}
			bg.curFile = reqFile
			if bg.curFile != nil {
				bg.curTexture = bg.genTexture(ctx.Renderer, bg.curFile)
			} else {
				bg.curTexture = nil
			}
		}
		bg.curTextureSplit = 0
	}
}
