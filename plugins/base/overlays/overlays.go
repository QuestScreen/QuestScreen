package overlays

import (
	"log"
	"time"

	"github.com/QuestScreen/api"
	"github.com/QuestScreen/api/modules"
	"github.com/QuestScreen/api/render"
	"github.com/QuestScreen/api/resources"
	"github.com/QuestScreen/api/server"
)

type config struct{}

type textureData struct {
	tex           render.Image
	resourceIndex int
}

type showRequest struct {
	resource      resources.Resource
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
	alphaMod          uint8
}

const duration = time.Second

func newRenderer(r render.Renderer,
	ms server.MessageSender) (modules.Renderer, error) {
	return &Overlays{status: resting, shownTexWidth: 0, curActive: -1}, nil
}

// Descriptor describes the Overlays module
var Descriptor = modules.Module{
	Name: "Overlays",
	ID:   "overlays",
	ResourceCollections: []resources.Selector{
		{Subdirectory: "", Suffixes: nil}},
	EndpointPaths:  []string{""},
	DefaultConfig:  &config{},
	CreateRenderer: newRenderer, CreateState: newState,
}

func (o *Overlays) loadTexture(r render.Renderer, td *textureData,
	resource resources.Resource, resourceIndex int) (loadedWidth int32) {
	tex, err := r.LoadImageFile(resource.Location, true)
	if err != nil {
		log.Println(err)
		return
	}
	frame := r.OutputSize()
	targetScale := float32(1.0)
	if tex.Height > frame.Height*2/3 {
		targetScale = float32(frame.Height*2/3) / float32(tex.Height)
	} else if tex.Height < frame.Height/2 {
		targetScale = float32(frame.Height/2) / float32(tex.Height)
	}
	if (float32(tex.Width) * targetScale) > float32(frame.Width/2) {
		targetScale = float32(frame.Width/2) / (float32(tex.Width) * targetScale)
	}
	if targetScale < 1.0 {
		loadedWidth = int32(float32(tex.Width) * targetScale)
		canvas, frame := r.CreateCanvas(loadedWidth,
			int32(float32(tex.Height)*targetScale),
			api.RGBA{R: 0, G: 0, B: 0, A: 255}.AsBackground(), render.Nowhere)
		tex.Draw(r, frame, 255)
		tex = canvas.Finish()
	} else {
		loadedWidth = tex.Width
	}
	td.tex = tex
	td.resourceIndex = resourceIndex
	return
}

func (o *Overlays) calcScale(r render.Renderer) {
	frame := r.OutputSize()
	wholeWidth := o.shownTexWidth + int32(len(o.textures)-1)*r.Unit()
	if wholeWidth > frame.Width*9/10 {
		o.targetScale = float32(frame.Width*9/10) / float32(o.shownTexWidth)
		o.targetXOffset = frame.Width / 20
	} else {
		o.targetScale = 1.0
		o.targetXOffset = (frame.Width - wholeWidth) / 2
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
func (o *Overlays) InitTransition(r render.Renderer,
	data interface{}) time.Duration {
	shown, ok := data.(*showRequest)
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
		width := o.loadTexture(r, newTexture, shown.resource, shown.resourceIndex)
		o.shownTexWidth += width
		o.calcScale(r)
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
			o.shownTexWidth -= o.textures[o.curActive].tex.Width
			o.calcScale(r)
			o.status = fadeOut
		} else {
			o.sendToRest()
			return -1
		}
	}
	return duration
}

// TransitionStep advances the transition.
func (o *Overlays) TransitionStep(r render.Renderer, elapsed time.Duration) {
	pos := render.TransitionCurve{Duration: duration}.Cubic(elapsed)
	o.curInactiveScale = o.startScale + pos*(o.targetScale-o.startScale)
	o.curXOffset = o.startXOffset + int32(pos*float32(o.targetXOffset-o.startXOffset))

	unit := r.Unit()
	if o.status == fadeIn {
		o.alphaMod = uint8(pos * 255)
		o.curActiveScale = pos * o.targetScale
		if o.curActive == 0 || o.curActive == len(o.textures)-1 {
			o.activeBorderWidth = int32(pos * float32(unit))
		} else {
			o.activeBorderWidth = unit/2 + int32(pos*float32(unit/2))
		}
	} else {
		o.alphaMod = 255 - uint8(pos*255)
		o.curActiveScale = (1.0 - pos) * o.startScale
		if o.curActive == 0 || o.curActive == len(o.textures)-1 {
			o.activeBorderWidth = int32((1.0 - pos) * float32(unit))
		} else {
			o.activeBorderWidth = unit/2 + int32((1.0-pos)*float32(unit/2))
		}
	}
}

// FinishTransition finalizes the transition.
func (o *Overlays) FinishTransition(r render.Renderer) {
	if o.status == fadeOut {
		r.FreeImage(&o.textures[o.curActive].tex)
		copy(o.textures[o.curActive:len(o.textures)-1], o.textures[o.curActive+1:len(o.textures)])
		o.textures = o.textures[:len(o.textures)-1]
	} else {
		o.alphaMod = 255
	}
	o.sendToRest()
}

// Render renders the module.
func (o *Overlays) Render(r render.Renderer) {
	//frame := r.OutputSize()
	curX := o.curXOffset

	for i := range o.textures {
		cur := &o.textures[i]
		var targetWidth, targetHeight int32
		if i == o.curActive {
			targetHeight = int32(float32(cur.tex.Height)*o.curActiveScale) - 1
			targetWidth = int32(float32(cur.tex.Width)*o.curActiveScale) - 1
		} else {
			targetHeight = int32(float32(cur.tex.Height)*o.curInactiveScale) - 1
			targetWidth = int32(float32(cur.tex.Width)*o.curInactiveScale) - 1
		}
		rect := render.Rectangle{X: curX, Y: 0,
			Width: targetWidth, Height: targetHeight}
		if i == o.curActive {
			cur.tex.Draw(r, rect, o.alphaMod)
		} else {
			cur.tex.Draw(r, rect, 255)
		}
		curX += targetWidth
		if i == o.curActive || i+1 == o.curActive {
			curX += o.activeBorderWidth
		} else {
			curX += r.Unit()
		}
	}
}

// Rebuild receives state data and config and immediately updates everything.
func (o *Overlays) Rebuild(r render.Renderer, data interface{},
	configVal interface{}) {
	o.config = configVal.(*config)
	if data == nil {
		return
	}
	req := data.(*fullRequest)
	for i := range o.textures {
		r.FreeImage(&o.textures[i].tex)
	}
	o.shownTexWidth = 0
	o.textures = make([]textureData, len(req.resources))
	for i := range o.textures {
		res := req.resources[i]
		o.shownTexWidth += o.loadTexture(
			r, &o.textures[i], res.resource, res.resourceIndex)
	}
	o.calcScale(r)
	o.sendToRest()
}
