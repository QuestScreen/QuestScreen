package overlays

import (
	"log"
	"time"

	"github.com/QuestScreen/QuestScreen/api"

	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
)

type config struct{}

type textureData struct {
	tex           *sdl.Texture
	resourceIndex int
}

type showRequest struct {
	resource      api.Resource
	resourceIndex int
}

type hideRequest struct {
	resourceIndex int
}

type status int

const (
	resting status = iota
	fadeIn
	fadeOut
)

type fullRequest struct {
	resources []showRequest
}

// The Overlays module can show pictures above the current background.
type Overlays struct {
	*config
	status
	// holds data for all currently shown textures.
	// textures are always sorted by resource index
	textures []textureData
	// index into textures slice of texture currently fading in/out
	// (not the resource index!)
	curActive int
	// summed width of all textures that are currently shown
	shownTexWidth int32
	// startXOffset is the X offset of the first texture at the start of the
	// the transition, targetXOffset the one at its end.
	startXOffset, targetXOffset int32
	// startScale is the scaling factor of all textures at the start of the
	// transition, targetScale the one at the end of the transition
	startScale, targetScale float32
	// curXOffset is the current X offset for the first texture
	curXOffset int32
	// curInactiveScale is the current scaling factor of all but the active
	// texture during transition, curActiveScale is the scaling factor of the
	// activeTexture
	curInactiveScale, curActiveScale float32
	// current width of the border around the active texture
	// (grows/shrinks while fadeIn/fadeOut from/to the border of the two adjacent
	// textures)
	activeBorderWidth int32
}

const duration = time.Second

func newRenderer(
	backend *sdl.Renderer, ms api.MessageSender) (api.ModuleRenderer, error) {
	return &Overlays{status: resting, shownTexWidth: 0, curActive: -1}, nil
}

// Descriptor describes the Overlays module
var Descriptor = api.Module{
	Name: "Overlays",
	ID:   "overlays",
	ResourceCollections: []api.ResourceSelector{
		{Subdirectory: "", Suffixes: nil}},
	EndpointPaths:  []string{""},
	DefaultConfig:  &config{},
	CreateRenderer: newRenderer, CreateState: newState,
}

// Descriptor returns the descriptor of the Overlays module
func (o *Overlays) Descriptor() *api.Module {
	return &Descriptor
}

func (td *textureData) loadTexture(renderer *sdl.Renderer,
	resource api.Resource, resourceIndex int) (loadedWidth int32) {
	tex, err := img.LoadTexture(renderer, resource.Path())
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
	if targetScale < 1.0 {
		loadedWidth = int32(float32(texWidth) * targetScale)
		// create a smaller texture
		smallerTex, err := renderer.CreateTexture(sdl.PIXELFORMAT_RGB888,
			sdl.TEXTUREACCESS_TARGET, loadedWidth,
			int32(float32(texHeight)*targetScale))
		if err == nil {
			renderer.SetRenderTarget(smallerTex)
			err = renderer.Copy(tex, nil, nil)
			if err == nil {
				tex.Destroy()
				tex = smallerTex
			} else {
				loadedWidth = texWidth
				smallerTex.Destroy()
				log.Println(err)
			}
			renderer.SetRenderTarget(nil)
		} else {
			loadedWidth = texWidth
			log.Println(err)
		}
	} else {
		loadedWidth = texWidth
	}
	td.tex = tex
	td.resourceIndex = resourceIndex
	return
}

func (o *Overlays) calcScale(ctx api.RenderContext) {
	winWidth, _, _ := ctx.Renderer().GetOutputSize()
	wholeWidth := o.shownTexWidth + int32(len(o.textures)-1)*ctx.DefaultBorderWidth()
	if wholeWidth > winWidth*9/10 {
		o.targetScale = float32(winWidth*9/10) / float32(o.shownTexWidth)
		o.targetXOffset = winWidth / 20
	} else {
		o.targetScale = 1.0
		o.targetXOffset = (winWidth - wholeWidth) / 2
	}
}

func (o *Overlays) sendToRest() {
	o.startScale = o.targetScale
	o.curInactiveScale = o.targetScale
	o.startXOffset = o.targetXOffset
	o.curXOffset = o.targetXOffset
	o.curActive = -1
	o.status = resting
}

// InitTransition initializes a transition.
func (o *Overlays) InitTransition(ctx api.RenderContext, data interface{}) time.Duration {
	shown, ok := data.(*showRequest)
	r := ctx.Renderer()
	if ok {
		o.textures = append(o.textures, textureData{})
		var newTexture *textureData
		for i := range o.textures {
			if i == len(o.textures)-1 {
				newTexture = &o.textures[i]
				o.curActive = i
			} else if o.textures[i].resourceIndex > shown.resourceIndex {
				copy(o.textures[i+1:len(o.textures)], o.textures[i:len(o.textures)-1])
				newTexture = &o.textures[i]
				o.curActive = i
				break
			} else if o.textures[i].resourceIndex == shown.resourceIndex {
				o.textures = o.textures[:len(o.textures)-1]
				return -1
			}
		}
		width := newTexture.loadTexture(r, shown.resource, shown.resourceIndex)
		o.shownTexWidth += width
		if err := newTexture.tex.SetBlendMode(sdl.BLENDMODE_BLEND); err != nil {
			log.Println(err)
		}
		o.calcScale(ctx)
		o.status = fadeIn
	} else {
		hide := data.(*hideRequest)
		o.curActive = -1
		for i := range o.textures {
			if o.textures[i].resourceIndex == hide.resourceIndex {
				o.curActive = i
				break
			}
		}
		if o.curActive != -1 {
			o.status = fadeOut
			if err := o.textures[o.curActive].tex.SetBlendMode(sdl.BLENDMODE_BLEND); err != nil {
				log.Println(err)
			}
			_, _, texWidth, _, _ := o.textures[o.curActive].tex.Query()
			o.shownTexWidth -= texWidth
			o.calcScale(ctx)
			o.status = fadeOut
		} else {
			o.sendToRest()
			return -1
		}
	}
	return duration
}

// TransitionStep advances the transition.
func (o *Overlays) TransitionStep(ctx api.RenderContext, elapsed time.Duration) {
	pos := api.TransitionCurve{Duration: duration}.Cubic(elapsed)
	o.curInactiveScale = o.startScale + pos*(o.targetScale-o.startScale)
	o.curXOffset = o.startXOffset + int32(pos*float32(o.targetXOffset-o.startXOffset))

	active := &o.textures[o.curActive]
	if o.status == fadeIn {
		err := active.tex.SetAlphaMod(uint8(pos * 255))
		if err != nil {
			log.Println(err)
		}
		o.curActiveScale = pos * o.targetScale
		if o.curActive == 0 || o.curActive == len(o.textures)-1 {
			o.activeBorderWidth = int32(pos * float32(ctx.DefaultBorderWidth()))
		} else {
			o.activeBorderWidth = ctx.DefaultBorderWidth()/2 + int32(pos*float32(ctx.DefaultBorderWidth()/2))
		}
	} else {
		err := o.textures[o.curActive].tex.SetAlphaMod(255 - uint8(pos*255))
		if err != nil {
			log.Println(err)
		}
		o.curActiveScale = (1.0 - pos) * o.startScale
		if o.curActive == 0 || o.curActive == len(o.textures)-1 {
			o.activeBorderWidth = int32((1.0 - pos) * float32(ctx.DefaultBorderWidth()))
		} else {
			o.activeBorderWidth = ctx.DefaultBorderWidth()/2 + int32((1.0-pos)*float32(ctx.DefaultBorderWidth()/2))
		}
	}
}

// FinishTransition finalizes the transition.
func (o *Overlays) FinishTransition(ctx api.RenderContext) {
	if o.status == fadeOut {
		_ = o.textures[o.curActive].tex.Destroy()
		copy(o.textures[o.curActive:len(o.textures)-1], o.textures[o.curActive+1:len(o.textures)])
		o.textures = o.textures[:len(o.textures)-1]
	} else {
		if err := o.textures[o.curActive].tex.SetBlendMode(sdl.BLENDMODE_NONE); err != nil {
			log.Println(err)
		}
		if err := o.textures[o.curActive].tex.SetAlphaMod(255); err != nil {
			log.Println(err)
		}
	}
	o.sendToRest()
}

// Render renders the module.
func (o *Overlays) Render(ctx api.RenderContext) {
	r := ctx.Renderer()
	_, winHeight, _ := r.GetOutputSize()
	curX := o.curXOffset

	for i := range o.textures {
		cur := &o.textures[i]
		_, _, texWidth, texHeight, _ := cur.tex.Query()
		var targetWidth, targetHeight int32
		if i == o.curActive {
			targetHeight = int32(float32(texHeight)*o.curActiveScale) - 1
			targetWidth = int32(float32(texWidth)*o.curActiveScale) - 1
		} else {
			targetHeight = int32(float32(texHeight)*o.curInactiveScale) - 1
			targetWidth = int32(float32(texWidth)*o.curInactiveScale) - 1
		}
		rect := sdl.Rect{X: curX, Y: winHeight - targetHeight, W: targetWidth, H: targetHeight}
		err := r.Copy(cur.tex, nil, &rect)
		if err != nil {
			log.Println(err)
		}
		curX += targetWidth
		if i == o.curActive || i+1 == o.curActive {
			curX += o.activeBorderWidth
		} else {
			curX += ctx.DefaultBorderWidth()
		}
	}
}

// SetConfig sets the module's configuration
func (o *Overlays) SetConfig(value interface{}) {
	o.config = value.(*config)
}

// RebuildState queries the new state through the channel and immediately
// updates everything.
func (o *Overlays) RebuildState(ctx api.ExtendedRenderContext, data interface{}) {
	if data == nil {
		return
	}
	req := data.(*fullRequest)
	for i := range o.textures {
		if o.textures[i].tex != nil {
			o.textures[i].tex.Destroy()
		}
	}
	o.shownTexWidth = 0
	o.textures = make([]textureData, len(req.resources))
	r := ctx.Renderer()
	for i := range o.textures {
		res := req.resources[i]
		o.shownTexWidth += o.textures[i].loadTexture(r, res.resource, res.resourceIndex)
	}
	o.calcScale(ctx)
	o.sendToRest()
}
