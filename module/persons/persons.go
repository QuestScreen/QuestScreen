package persons

import (
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/flyx/rpscreen/data"
	"github.com/flyx/rpscreen/module"
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
)

type personsConfig struct{}

// The Persons module can show pictures of persons and other stuff.
type Persons struct {
	common          *module.SceneCommon
	config          *personsConfig
	textures        []*sdl.Texture
	textureScale    []float32
	reqTextureIndex int
	reqShow         bool
	files           []data.Resource
	shown           []bool
	curScale        float32
	curOrigWidth    int32
	transitioning   bool
}

// Init initializes the module.
func (p *Persons) Init(common *module.SceneCommon) error {
	p.common = common
	p.files = common.ListFiles(p, "")
	p.textures = make([]*sdl.Texture, len(p.files))
	p.textureScale = make([]float32, len(p.files))
	for index := range p.textures {
		p.textures[index] = nil
		p.textureScale[index] = 1
	}

	p.reqTextureIndex = -1
	p.curScale = 1
	p.shown = make([]bool, len(p.files))
	for index := range p.shown {
		p.shown[index] = false
	}
	return nil
}

// Name returns "Persons"
func (*Persons) Name() string {
	return "Persons"
}

// InternalName returns "persons"
func (*Persons) InternalName() string {
	return "persons"
}

// UI renders the HTML UI of the module.
func (p *Persons) UI() template.HTML {
	var builder module.UIBuilder

	for index, file := range p.files {
		if file.Enabled(&p.common.Store) {
			builder.StartForm(p, "switch", "", true)
			builder.HiddenValue("index", strconv.Itoa(index))
			if p.shown[index] {
				builder.SubmitButton("Hide", file.Name, true)
			} else {
				builder.SecondarySubmitButton("Show", file.Name, true)
			}
			builder.EndForm()
		}
	}
	return builder.Finish()
}

// EndpointHandler implements the endpoint handler of the module.
func (p *Persons) EndpointHandler(suffix string, values url.Values, w http.ResponseWriter, returnPartial bool) bool {
	if suffix == "switch" {
		index, err := strconv.Atoi(values["index"][0])
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return false
		} else if index < 0 || index >= len(p.files) {
			http.Error(w, "index out of range", http.StatusBadRequest)
			return false
		}
		p.reqTextureIndex = index
		p.reqShow = !p.shown[index]
		p.shown[index] = true

		var returns module.EndpointReturn
		if returnPartial {
			returns = module.EndpointReturnEmpty
		} else {
			returns = module.EndpointReturnRedirect
		}
		module.WriteEndpointHeader(w, returns)
		return true
	}
	http.Error(w, "404 not found: "+suffix, http.StatusNotFound)
	return false
}

// InitTransition initializes a transition.
func (p *Persons) InitTransition() time.Duration {
	var ret time.Duration = -1
	if p.reqShow {
		file := p.files[p.reqTextureIndex]
		tex, err := img.LoadTexture(p.common.Renderer, file.Path)
		if err != nil {
			log.Println(err)
		} else {
			p.textures[p.reqTextureIndex] = tex
			_, _, texWidth, texHeight, _ := tex.Query()
			winWidth, winHeight := p.common.Window.GetSize()
			targetScale := float32(1.0)
			if texHeight > winHeight*2/3 {
				targetScale = float32(winHeight*2/3) / float32(texHeight)
			} else if texHeight < winHeight/2 {
				targetScale = float32(winHeight/2) / float32(texHeight)
			}
			if (float32(texWidth) * targetScale) > float32(winWidth/2) {
				targetScale = float32(winWidth/2) / (float32(texWidth) * targetScale)
			}
			p.textureScale[p.reqTextureIndex] = targetScale
			p.curOrigWidth = p.curOrigWidth + int32(float32(texWidth)*targetScale)
			if p.curOrigWidth > winWidth*9/10 {
				p.curScale = float32(winWidth*9/10) / float32(p.curOrigWidth)
			} else {
				p.curScale = 1
			}
			ret = time.Second
			p.transitioning = true
			if err := p.textures[p.reqTextureIndex].SetBlendMode(sdl.BLENDMODE_BLEND); err != nil {
				log.Println(err)
			}
			p.shown[p.reqTextureIndex] = true
		}
	} else {
		ret = time.Second
		p.transitioning = true
		if err := p.textures[p.reqTextureIndex].SetBlendMode(sdl.BLENDMODE_BLEND); err != nil {
			log.Println(err)
		}
		p.shown[p.reqTextureIndex] = false
	}
	return ret
}

// TransitionStep advances the transition.
func (p *Persons) TransitionStep(elapsed time.Duration) {
	if p.reqShow {
		err := p.textures[p.reqTextureIndex].SetAlphaMod(uint8((elapsed * 255) / time.Second))
		if err != nil {
			log.Println(err)
		}
	} else {
		err := p.textures[p.reqTextureIndex].SetAlphaMod(255 - uint8((elapsed*255)/time.Second))
		if err != nil {
			log.Println(err)
		}
	}
}

// FinishTransition finalizes the transition.
func (p *Persons) FinishTransition() {
	if !p.reqShow {
		_, _, texWidth, _, _ := p.textures[p.reqTextureIndex].Query()
		winWidth, _ := p.common.Window.GetSize()
		_ = p.textures[p.reqTextureIndex].Destroy()
		p.textures[p.reqTextureIndex] = nil
		p.curOrigWidth = p.curOrigWidth - int32(float32(texWidth)*p.textureScale[p.reqTextureIndex])
		if p.curOrigWidth > winWidth*9/10 {
			p.curScale = float32(winWidth*9/10) / float32(p.curOrigWidth)
		} else {
			p.curScale = 1
		}
	}
	if err := p.textures[p.reqTextureIndex].SetBlendMode(sdl.BLENDMODE_NONE); err != nil {
		log.Println(err)
	}
	if err := p.textures[p.reqTextureIndex].SetAlphaMod(255); err != nil {
		log.Println(err)
	}
	p.transitioning = false
}

// Render renders the module.
func (p *Persons) Render() {
	winWidth, winHeight := p.common.Window.GetSize()
	curX := (winWidth - int32(float32(p.curOrigWidth)*p.curScale)) / 2
	for i := range p.textures {
		if p.shown[i] || (i == p.reqTextureIndex && p.transitioning) {
			_, _, texWidth, texHeight, _ := p.textures[i].Query()
			targetHeight := int32(float32(texHeight) * p.textureScale[i] * p.curScale)
			targetWidth := int32(float32(texWidth) * p.textureScale[i] * p.curScale)
			rect := sdl.Rect{X: curX, Y: winHeight - targetHeight, W: targetWidth, H: targetHeight}
			curX += targetWidth
			err := p.common.Renderer.Copy(p.textures[i], nil, &rect)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

// EmptyConfig returns an empty configuration
func (*Persons) EmptyConfig() interface{} {
	return &personsConfig{}
}

// DefaultConfig returns the default configuration
func (*Persons) DefaultConfig() interface{} {
	return &personsConfig{}
}

// SetConfig sets the module's configuration
func (p *Persons) SetConfig(config interface{}) {
	p.config = config.(*personsConfig)
}

// GetConfig retrieves the current configuration of the item.
func (p *Persons) GetConfig() interface{} {
	return p.config
}

// NeedsTransition returns false
func (*Persons) NeedsTransition() bool {
	return false
}
