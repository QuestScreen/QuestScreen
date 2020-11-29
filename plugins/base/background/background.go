package background

import (
	"log"
	"time"

	"github.com/QuestScreen/api/colors"
	"github.com/QuestScreen/api/modules"
	"github.com/QuestScreen/api/render"
	"github.com/QuestScreen/api/resources"
	"github.com/QuestScreen/api/server"
)

type backgroundConfig struct{}

type request struct {
	file resources.Resource
}

// Background is a module for painting background images
type Background struct {
	config                 *backgroundConfig
	curTexture, newTexture render.Image
	curFile                resources.Resource
	alphaMod               uint8
}

func newRenderer(backend render.Renderer,
	ms server.MessageSender) (modules.Renderer, error) {
	bg := new(Background)
	return bg, nil
}

// Descriptor describes the Background module.
var Descriptor = modules.Module{
	Name: "Background Image",
	ID:   "background",
	ResourceCollections: []resources.Selector{
		{Subdirectory: "", Suffixes: nil}},
	EndpointPaths:  []string{""},
	DefaultConfig:  &backgroundConfig{},
	CreateRenderer: newRenderer,
	CreateState:    newState,
}

func (bg *Background) genTexture(
	renderer render.Renderer, file resources.Resource) render.Image {
	tex, err := renderer.LoadImageFile(file.Path(), true)
	if err != nil {
		log.Println(err)
		return render.Image{}
	}
	defer renderer.FreeImage(&tex)

	window := renderer.OutputSize()
	canvas, _ := renderer.CreateCanvas(window.Width, window.Height,
		colors.RGB{R: 0, G: 0, B: 0}.AsBackground(), render.Nowhere)
	scaleFactor := float32(1.0)
	texRatio := float32(tex.Width) / float32(tex.Height)
	winRatio := float32(window.Width) / float32(window.Height)
	if texRatio > winRatio {
		scaleFactor = float32(window.Width) / float32(tex.Width)
	} else {
		scaleFactor = float32(window.Height) / float32(tex.Height)
	}
	resW := int32(scaleFactor * float32(tex.Width))
	resH := int32(scaleFactor * float32(tex.Height))
	frame := window.Position(resW, resH, render.Center, render.Middle)
	tex.Draw(renderer, frame, 255)

	return canvas.Finish()
}

// InitTransition initializes a transition
func (bg *Background) InitTransition(r render.Renderer, data interface{}) time.Duration {
	var ret time.Duration = -1
	req := data.(*request)

	if req.file != bg.curFile {
		bg.curFile = req.file
		if bg.curFile != nil {
			bg.newTexture = bg.genTexture(r, bg.curFile)
		}
		ret = time.Second
	}
	return ret
}

// TransitionStep advances the transition.
func (bg *Background) TransitionStep(r render.Renderer, elapsed time.Duration) {
	if !bg.newTexture.IsEmpty() {
		bg.alphaMod = uint8((elapsed * 255) / time.Second)
	}
}

// FinishTransition finalizes the transition.
func (bg *Background) FinishTransition(r render.Renderer) {
	if !bg.curTexture.IsEmpty() {
		r.FreeImage(&bg.curTexture)
	}
	bg.curTexture = bg.newTexture
	bg.newTexture.Width = 0
}

// Render renders the module
func (bg *Background) Render(r render.Renderer) {
	window := r.OutputSize()
	if !bg.curTexture.IsEmpty() {
		bg.curTexture.Draw(r, window, 255)
	}
	if !bg.newTexture.IsEmpty() {
		bg.newTexture.Draw(r, window, bg.alphaMod)
	}
}

// EmptyConfig returns an empty configuration
func (*Background) EmptyConfig() interface{} {
	return &backgroundConfig{}
}

// Rebuild receives state data and config and immediately updates everything.
func (bg *Background) Rebuild(
	r render.Renderer, data interface{}, configVal interface{}) {
	bg.config = configVal.(*backgroundConfig)
	if data != nil {
		req := data.(*request)
		if req.file != bg.curFile {
			r.FreeImage(&bg.curTexture)
			bg.curFile = req.file
			if bg.curFile != nil {
				bg.curTexture = bg.genTexture(r, bg.curFile)
			}
		}
		r.FreeImage(&bg.newTexture)
	}
}
