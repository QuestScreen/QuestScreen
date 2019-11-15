package persons

import (
	"log"
	"sync"
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

type requests struct {
	mutex     sync.Mutex
	kind      requestKind
	itemIndex int
	itemShown bool
	state     []bool
}

// The Persons module can show pictures of persons and other stuff.
type Persons struct {
	*config
	state
	status
	requests

	env          api.Environment
	moduleIndex  api.ModuleIndex
	textures     []textureData
	curIndex     int
	curScale     float32
	curOrigWidth int32
}

// Init initializes the module.
func (p *Persons) Init(
	renderer *sdl.Renderer, env api.Environment, index api.ModuleIndex) error {
	p.env = env
	p.moduleIndex = index
	p.curScale = 1
	p.status = resting
	p.state.owner = p
	return nil
}

// Name returns "Persons"
func (*Persons) Name() string {
	return "Persons"
}

// ID returns "persons"
func (*Persons) ID() string {
	return "persons"
}

func (p *Persons) loadTexture(renderer *sdl.Renderer, index int) textureData {
	file := p.state.resources[index]
	tex, err := img.LoadTexture(renderer, file.Path())
	if err != nil {
		log.Println(err)
		return textureData{tex: nil, shown: true, scale: 1}
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
	p.curOrigWidth = p.curOrigWidth + int32(float32(texWidth)*targetScale)
	if p.curOrigWidth > winWidth*9/10 {
		p.curScale = float32(winWidth*9/10) / float32(p.curOrigWidth)
	} else {
		p.curScale = 1
	}
	return textureData{tex: tex, shown: true, scale: targetScale}
}

// InitTransition initializes a transition.
func (p *Persons) InitTransition(renderer *sdl.Renderer) time.Duration {
	p.requests.mutex.Lock()
	if p.requests.kind != itemRequest {
		p.requests.mutex.Unlock()
		return -1
	}
	p.requests.kind = noRequest
	index := p.requests.itemIndex
	shown := p.requests.itemShown
	p.requests.mutex.Unlock()
	if shown {
		p.textures[index] = p.loadTexture(renderer, index)
		p.status = fadeIn
		if err := p.textures[index].tex.SetBlendMode(sdl.BLENDMODE_BLEND); err != nil {
			log.Println(err)
		}
		p.textures[index].shown = true
		p.curIndex = index
	} else {
		p.textures[index].shown = false
		p.status = fadeOut
		if err := p.textures[index].tex.SetBlendMode(sdl.BLENDMODE_BLEND); err != nil {
			log.Println(err)
		}
		p.textures[index].shown = false
		p.curIndex = index
	}
	return time.Second
}

// TransitionStep advances the transition.
func (p *Persons) TransitionStep(renderer *sdl.Renderer, elapsed time.Duration) {
	if p.status == fadeIn {
		err := p.textures[p.curIndex].tex.SetAlphaMod(uint8((elapsed * 255) / time.Second))
		if err != nil {
			log.Println(err)
		}
	} else {
		err := p.textures[p.curIndex].tex.SetAlphaMod(255 - uint8((elapsed*255)/time.Second))
		if err != nil {
			log.Println(err)
		}
	}
}

// FinishTransition finalizes the transition.
func (p *Persons) FinishTransition(renderer *sdl.Renderer) {
	if p.status == fadeOut {
		_, _, texWidth, _, _ := p.textures[p.curIndex].tex.Query()
		winWidth, _, _ := renderer.GetOutputSize()
		_ = p.textures[p.curIndex].tex.Destroy()
		p.textures[p.curIndex].tex = nil
		p.curOrigWidth = p.curOrigWidth - int32(float32(texWidth)*p.textures[p.curIndex].scale)
		if p.curOrigWidth > winWidth*9/10 {
			p.curScale = float32(winWidth*9/10) / float32(p.curOrigWidth)
		} else {
			p.curScale = 1
		}
	}
	if err := p.textures[p.curIndex].tex.SetBlendMode(sdl.BLENDMODE_NONE); err != nil {
		log.Println(err)
	}
	if err := p.textures[p.curIndex].tex.SetAlphaMod(255); err != nil {
		log.Println(err)
	}
	p.status = resting
}

// Render renders the module.
func (p *Persons) Render(renderer *sdl.Renderer) {
	winWidth, winHeight, _ := renderer.GetOutputSize()
	curX := (winWidth - int32(float32(p.curOrigWidth)*p.curScale)) / 2
	for i := range p.textures {
		if p.textures[i].shown || (i == p.curIndex && p.status != resting) {
			_, _, texWidth, texHeight, _ := p.textures[i].tex.Query()
			targetHeight := int32(float32(texHeight) * p.textures[i].scale * p.curScale)
			targetWidth := int32(float32(texWidth) * p.textures[i].scale * p.curScale)
			rect := sdl.Rect{X: curX, Y: winHeight - targetHeight, W: targetWidth, H: targetHeight}
			curX += targetWidth
			err := renderer.Copy(p.textures[i].tex, nil, &rect)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

// EmptyConfig returns an empty configuration
func (*Persons) EmptyConfig() interface{} {
	return &config{}
}

// DefaultConfig returns the default configuration
func (*Persons) DefaultConfig() interface{} {
	return &config{}
}

// SetConfig sets the module's configuration
func (p *Persons) SetConfig(value interface{}) {
	p.config = value.(*config)
}

// State returns the current state.
func (p *Persons) State() api.ModuleState {
	return &p.state
}

// RebuildState queries the new state through the channel and immediately
// updates everything.
func (p *Persons) RebuildState(renderer *sdl.Renderer) {
	p.requests.mutex.Lock()
	if p.requests.kind != stateRequest {
		panic("RebuildState() called on something which is not stateRequest")
	}
	newState := p.requests.state
	p.requests.kind = noRequest
	p.requests.mutex.Unlock()
	p.curOrigWidth = 0
	p.curScale = 1
	for i := range p.textures {
		if p.textures[i].tex != nil {
			p.textures[i].tex.Destroy()
		}
	}
	p.textures = make([]textureData, len(newState))
	for i := range p.textures {
		if newState[i] {
			p.textures[i] = p.loadTexture(renderer, i)
		} else {
			p.textures[i].scale = 1
			p.textures[i].shown = false
		}
	}
}

// ResourceCollections returns a singleton list describing the selector for
// overlay images.
func (p *Persons) ResourceCollections() []api.ResourceSelector {
	return []api.ResourceSelector{
		{Subdirectory: "", Suffixes: nil}}
}
