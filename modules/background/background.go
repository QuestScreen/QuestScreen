package background

import (
	"log"
	"sync"
	"time"

	"github.com/flyx/pnpscreen/api"

	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
)

type backgroundConfig struct{}

type requests struct {
	mutex         sync.Mutex
	activeRequest bool
	index         int
}

// Background is a module for painting background images
type Background struct {
	state
	requests
	env                    api.Environment
	moduleIndex            api.ModuleIndex
	config                 *backgroundConfig
	curTexture, newTexture *sdl.Texture
	curTextureIndex        int
	curTextureSplit        float32
}

// Init initializes the module
func (bg *Background) Init(renderer *sdl.Renderer, env api.Environment, index api.ModuleIndex) error {
	bg.env = env
	bg.moduleIndex = index
	bg.curTexture = nil
	bg.newTexture = nil
	bg.curTextureSplit = 0
	bg.curTextureIndex = -1
	bg.state.owner = bg
	return nil
}

// Name returns "Background Image"
func (*Background) Name() string {
	return "Background Image"
}

// ID returns "background"
func (*Background) ID() string {
	return "background"
}

func offsets(inRatio float32, outRatio float32, winWidth int32, winHeight int32) sdl.Rect {
	if inRatio > outRatio {
		return sdl.Rect{X: 0, Y: int32(float32(winHeight) * (1.0 - (outRatio / inRatio)) / 2.0),
			W: winWidth, H: int32(float32(winHeight) * (outRatio / inRatio))}
	}
	return sdl.Rect{X: int32(float32(winWidth) * (1.0 - (inRatio / outRatio)) / 2.0),
		Y: 0, W: int32(float32(winWidth) * (inRatio / outRatio)), H: winHeight}
}

func (bg *Background) genTexture(renderer *sdl.Renderer, index int) *sdl.Texture {
	file := bg.state.resources[index]
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
func (bg *Background) InitTransition(renderer *sdl.Renderer) time.Duration {
	var ret time.Duration = -1

	bg.requests.mutex.Lock()
	reqIndex, available := bg.requests.index, bg.requests.activeRequest
	bg.requests.activeRequest = false
	bg.requests.mutex.Unlock()

	if available && reqIndex != bg.curTextureIndex {
		bg.curTextureIndex = reqIndex
		if bg.curTextureIndex != -1 {
			bg.newTexture = bg.genTexture(renderer, bg.curTextureIndex)
		}
		bg.curTextureSplit = 0
		ret = time.Second
	} else {
		log.Println("background transition skiping, available=", available, ", reqIndex=", reqIndex)
	}
	return ret
}

// TransitionStep advances the transition.
func (bg *Background) TransitionStep(renderer *sdl.Renderer, elapsed time.Duration) {
	bg.curTextureSplit = float32(elapsed) / float32(time.Second)
}

// FinishTransition finalizes the transition.
func (bg *Background) FinishTransition(renderer *sdl.Renderer) {
	if bg.curTexture != nil {
		bg.curTexture.Destroy()
	}
	bg.curTexture = bg.newTexture
	bg.curTextureSplit = 0.0
	bg.newTexture = nil
}

// Render renders the module
func (bg *Background) Render(renderer *sdl.Renderer) {
	if bg.curTexture != nil || bg.curTextureSplit != 0 {
		winWidth, winHeight, _ := renderer.GetOutputSize()
		curSplit := int32(bg.curTextureSplit * float32(winWidth))
		if bg.curTexture != nil {
			rect := sdl.Rect{X: curSplit, Y: 0, W: winWidth - curSplit, H: winHeight}
			renderer.Copy(bg.curTexture, &rect, &rect)
		}
		if bg.newTexture != nil {
			rect := sdl.Rect{X: 0, Y: 0, W: curSplit, H: winHeight}
			renderer.Copy(bg.newTexture, &rect, &rect)
		}
	}
}

// EmptyConfig returns an empty configuration
func (*Background) EmptyConfig() interface{} {
	return &backgroundConfig{}
}

// DefaultConfig returns the default configuration
func (*Background) DefaultConfig() interface{} {
	return &backgroundConfig{}
}

// SetConfig sets the module's configuration
func (bg *Background) SetConfig(config interface{}) {
	bg.config = config.(*backgroundConfig)
}

// State returns the current state.
func (bg *Background) State() api.ModuleState {
	return &bg.state
}

// RebuildState queries the texture index through the channel and immediately
// sets that texture as background.
func (bg *Background) RebuildState(renderer *sdl.Renderer) {
	bg.requests.mutex.Lock()
	reqIndex, available := bg.requests.index, bg.requests.activeRequest
	bg.activeRequest = false
	bg.requests.mutex.Unlock()

	if available {
		if reqIndex != bg.curTextureIndex {
			if bg.curTexture != nil {
				bg.curTexture.Destroy()
			}
			bg.curTextureIndex = reqIndex
			if bg.curTextureIndex != -1 {
				bg.curTexture = bg.genTexture(renderer, bg.curTextureIndex)
			} else {
				bg.curTexture = nil
			}
		}
		bg.curTextureSplit = 0
	}
}

// ResourceCollections returns a singleton list describing the selector for
// background images.
func (bg *Background) ResourceCollections() []api.ResourceSelector {
	return []api.ResourceSelector{
		api.ResourceSelector{Subdirectory: "", Suffixes: nil}}
}
