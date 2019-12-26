package overlays

import (
	"log"
	"sync"
	"time"

	"github.com/flyx/pnpscreen/api"
	"gopkg.in/yaml.v3"

	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
)

type config struct{}

type textureData struct {
	tex   *sdl.Texture
	scale float32
	shown bool
	file  api.Resource
}

type status int

const (
	resting status = iota
	fadeIn
	fadeOut
)

type requestKind int

const (
	noRequest requestKind = iota
	itemRequest
	stateRequest
)

type sharedData struct {
	moduleIndex api.ModuleIndex
	mutex       sync.Mutex
	kind        requestKind
	itemIndex   int
	itemShown   bool
	state       []bool
	resources   []api.Resource
}

// The Overlays module can show pictures above the current background.
type Overlays struct {
	*config
	status
	sharedData
	textures     []textureData
	curIndex     int
	curScale     float32
	curOrigWidth int32
}

// CreateModule creates the Overlays module.
func CreateModule(renderer *sdl.Renderer, env api.StaticEnvironment,
	index api.ModuleIndex) (api.Module, error) {
	o := new(Overlays)
	o.moduleIndex = index
	o.curScale = 1
	o.status = resting
	return o, nil
}

// Descriptor describes the Overlays module
var Descriptor = api.ModuleDescriptor{
	Name: "Overlays",
	ID:   "overlays",
	ResourceCollections: []api.ResourceSelector{
		{Subdirectory: "", Suffixes: nil}},
	Actions:       []string{"switch"},
	DefaultConfig: &config{},
	CreateModule:  CreateModule,
}

// Descriptor returns the descriptor of the Overlays module
func (o *Overlays) Descriptor() *api.ModuleDescriptor {
	return &Descriptor
}

func (o *Overlays) loadTexture(renderer *sdl.Renderer, data *textureData) {
	tex, err := img.LoadTexture(renderer, data.file.Path())
	if err != nil {
		log.Println(err)
		return
	}
	_, _, texWidth, texHeight, _ := tex.Query()
	winWidth, winHeight, _ := renderer.GetOutputSize()
	targetScale := float32(1.0)
	if texHeight > winHeight*2/3 {
		targetScale = float32(winHeight*2/3) / float32(texHeight)
	} else if texHeight < winHeight/2 {
		targetScale = float32(winHeight/2) / float32(texHeight)
	}
	if (float32(texWidth) * targetScale) > float32(winWidth/2) {
		targetScale = float32(winWidth/2) / (float32(texWidth) * targetScale)
	}
	o.curOrigWidth = o.curOrigWidth + int32(float32(texWidth)*targetScale)
	if o.curOrigWidth > winWidth*9/10 {
		o.curScale = float32(winWidth*9/10) / float32(o.curOrigWidth)
	} else {
		o.curScale = 1
	}
	data.tex = tex
	data.scale = targetScale
}

// InitTransition initializes a transition.
func (o *Overlays) InitTransition(ctx api.RenderContext) time.Duration {
	o.sharedData.mutex.Lock()
	if o.sharedData.kind != itemRequest {
		o.sharedData.mutex.Unlock()
		return -1
	}
	o.sharedData.kind = noRequest
	index := o.sharedData.itemIndex
	shown := o.sharedData.itemShown
	o.sharedData.mutex.Unlock()
	if shown {
		o.loadTexture(ctx.Renderer, &o.textures[index])
		o.status = fadeIn
		if err := o.textures[index].tex.SetBlendMode(sdl.BLENDMODE_BLEND); err != nil {
			log.Println(err)
		}
		o.textures[index].shown = true
		o.curIndex = index
	} else {
		o.textures[index].shown = false
		o.status = fadeOut
		if err := o.textures[index].tex.SetBlendMode(sdl.BLENDMODE_BLEND); err != nil {
			log.Println(err)
		}
		o.textures[index].shown = false
		o.curIndex = index
	}
	return time.Second
}

// TransitionStep advances the transition.
func (o *Overlays) TransitionStep(ctx api.RenderContext, elapsed time.Duration) {
	if o.status == fadeIn {
		err := o.textures[o.curIndex].tex.SetAlphaMod(uint8((elapsed * 255) / time.Second))
		if err != nil {
			log.Println(err)
		}
	} else {
		err := o.textures[o.curIndex].tex.SetAlphaMod(255 - uint8((elapsed*255)/time.Second))
		if err != nil {
			log.Println(err)
		}
	}
}

// FinishTransition finalizes the transition.
func (o *Overlays) FinishTransition(ctx api.RenderContext) {
	if o.status == fadeOut {
		_, _, texWidth, _, _ := o.textures[o.curIndex].tex.Query()
		winWidth, _, _ := ctx.Renderer.GetOutputSize()
		_ = o.textures[o.curIndex].tex.Destroy()
		o.textures[o.curIndex].tex = nil
		o.curOrigWidth = o.curOrigWidth - int32(float32(texWidth)*o.textures[o.curIndex].scale)
		if o.curOrigWidth > winWidth*9/10 {
			o.curScale = float32(winWidth*9/10) / float32(o.curOrigWidth)
		} else {
			o.curScale = 1
		}
	}
	if err := o.textures[o.curIndex].tex.SetBlendMode(sdl.BLENDMODE_NONE); err != nil {
		log.Println(err)
	}
	if err := o.textures[o.curIndex].tex.SetAlphaMod(255); err != nil {
		log.Println(err)
	}
	o.status = resting
}

// Render renders the module.
func (o *Overlays) Render(ctx api.RenderContext) {
	winWidth, winHeight, _ := ctx.Renderer.GetOutputSize()
	curX := (winWidth - int32(float32(o.curOrigWidth)*o.curScale)) / 2
	for i := range o.textures {
		if o.textures[i].shown || (i == o.curIndex && o.status != resting) {
			_, _, texWidth, texHeight, _ := o.textures[i].tex.Query()
			targetHeight := int32(float32(texHeight) * o.textures[i].scale * o.curScale)
			targetWidth := int32(float32(texWidth) * o.textures[i].scale * o.curScale)
			rect := sdl.Rect{X: curX, Y: winHeight - targetHeight, W: targetWidth, H: targetHeight}
			curX += targetWidth
			err := ctx.Renderer.Copy(o.textures[i].tex, nil, &rect)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

// SetConfig sets the module's configuration
func (o *Overlays) SetConfig(value interface{}) {
	o.config = value.(*config)
}

// CreateState creates a new state for this module.
func (o *Overlays) CreateState(
	input *yaml.Node, env api.Environment) (api.ModuleState, error) {
	return newState(input, env, &o.sharedData)
}

// RebuildState queries the new state through the channel and immediately
// updates everything.
func (o *Overlays) RebuildState(ctx api.RenderContext) {
	var newState []bool = nil
	var newResources []api.Resource
	o.sharedData.mutex.Lock()
	switch o.sharedData.kind {
	case stateRequest:
		newState = o.sharedData.state
		newResources = o.sharedData.resources
	case noRequest:
		break
	default:
		panic("RebuildState() called on something which is not stateRequest")
	}
	o.sharedData.kind = noRequest
	o.sharedData.mutex.Unlock()
	if newState == nil {
		return
	}
	o.curOrigWidth = 0
	o.curScale = 1
	for i := range o.textures {
		if o.textures[i].tex != nil {
			o.textures[i].tex.Destroy()
		}
	}
	o.textures = make([]textureData, len(newState))
	for i := range o.textures {
		o.textures[i].file = newResources[i]
		if newState[i] {
			o.loadTexture(ctx.Renderer, &o.textures[i])
			o.textures[i].shown = true
		} else {
			o.textures[i].scale = 1
			o.textures[i].shown = false
		}
	}
}
