package overlays

import (
	"log"
	"time"

	"github.com/flyx/pnpscreen/api"

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

type itemRequest struct {
	index   int
	visible bool
}

type fullRequest struct {
	visible   []bool
	resources []api.Resource
}

// The Overlays module can show pictures above the current background.
type Overlays struct {
	*config
	status
	textures     []textureData
	curIndex     int
	curScale     float32
	curOrigWidth int32
}

func newModule(renderer *sdl.Renderer, env api.StaticEnvironment) (api.Module, error) {
	return &Overlays{curScale: 1, status: resting}, nil
}

// Descriptor describes the Overlays module
var Descriptor = api.ModuleDescriptor{
	Name: "Overlays",
	ID:   "overlays",
	ResourceCollections: []api.ResourceSelector{
		{Subdirectory: "", Suffixes: nil}},
	EndpointPaths: []string{""},
	DefaultConfig: &config{},
	CreateModule:  newModule, CreateState: newState,
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
func (o *Overlays) InitTransition(ctx api.RenderContext, data interface{}) time.Duration {
	req := data.(*itemRequest)
	if req.visible {
		o.loadTexture(ctx.Renderer, &o.textures[req.index])
		o.status = fadeIn
		if err := o.textures[req.index].tex.SetBlendMode(sdl.BLENDMODE_BLEND); err != nil {
			log.Println(err)
		}
		o.textures[req.index].shown = true
		o.curIndex = req.index
	} else {
		o.textures[req.index].shown = false
		o.status = fadeOut
		if err := o.textures[req.index].tex.SetBlendMode(sdl.BLENDMODE_BLEND); err != nil {
			log.Println(err)
		}
		o.textures[req.index].shown = false
		o.curIndex = req.index
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

// RebuildState queries the new state through the channel and immediately
// updates everything.
func (o *Overlays) RebuildState(ctx api.RenderContext, data interface{}) {
	if data == nil {
		return
	}
	req := data.(*fullRequest)
	o.curOrigWidth = 0
	o.curScale = 1
	for i := range o.textures {
		if o.textures[i].tex != nil {
			o.textures[i].tex.Destroy()
		}
	}
	o.textures = make([]textureData, len(req.resources))
	for i := range o.textures {
		o.textures[i].file = req.resources[i]
		if req.visible[i] {
			o.loadTexture(ctx.Renderer, &o.textures[i])
			o.textures[i].shown = true
		} else {
			o.textures[i].scale = 1
			o.textures[i].shown = false
		}
	}
}
