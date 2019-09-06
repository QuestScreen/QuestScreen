package background

import (
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/flyx/rpscreen/data"
	"github.com/flyx/rpscreen/display"
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
)

type backgroundConfig struct{}

// Background is a module for painting background images
type Background struct {
	display *display.Display
	// TODO: remove
	store                            *data.Store
	config                           *backgroundConfig
	texture, newTexture              *sdl.Texture
	reqTextureIndex, curTextureIndex int
	images                           []data.Resource
	curTextureSplit                  float32
}

// Init initializes the module
func (bg *Background) Init(display *display.Display, store *data.Store) error {
	bg.display = display
	bg.store = store
	bg.images = store.ListFiles(bg, "")
	bg.texture = nil
	bg.newTexture = nil

	bg.reqTextureIndex = -1
	bg.curTextureIndex = len(bg.images)
	bg.curTextureSplit = 0
	return nil
}

// Name returns "Background Image"
func (*Background) Name() string {
	return "Background Image"
}

// InternalName returns "background"
func (*Background) InternalName() string {
	return "background"
}

// UI renders the HTML UI of the module.
func (bg *Background) UI() template.HTML {
	var builder display.UIBuilder
	shownIndex := bg.reqTextureIndex
	if shownIndex == -1 {
		shownIndex = bg.curTextureIndex
	}
	builder.StartForm(bg, "image", "Select Image", false)
	builder.StartSelect("", "image", "value")
	builder.Option("", shownIndex == len(bg.images), "None")
	for index, file := range bg.images {
		if file.Enabled(bg.store) {
			builder.Option(strconv.Itoa(index), shownIndex == index, file.Name)
		}
	}
	builder.EndSelect()
	builder.SubmitButton("Update", "", true)
	builder.EndForm()

	return builder.Finish()
}

// EndpointHandler implements the endpoint of the module.
func (bg *Background) EndpointHandler(suffix string, values url.Values, w http.ResponseWriter, returnPartial bool) bool {
	if suffix == "image" {
		value := values["value"][0]
		if value == "" {
			bg.reqTextureIndex = len(bg.images)
		} else {
			index, err := strconv.Atoi(value)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return false
			}
			if index < 0 || index >= len(bg.images) {
				http.Error(w, "image index out of range", http.StatusBadRequest)
				return false
			}
			bg.reqTextureIndex = index
		}
		var returns display.EndpointReturn
		if returnPartial {
			returns = display.EndpointReturnEmpty
		} else {
			returns = display.EndpointReturnRedirect
		}
		display.WriteEndpointHeader(w, returns)
		return true
	}
	http.Error(w, "404 not found: "+suffix, http.StatusNotFound)
	return false
}

func offsets(inRatio float32, outRatio float32, winWidth int32, winHeight int32) sdl.Rect {
	if inRatio > outRatio {
		return sdl.Rect{X: 0, Y: int32(float32(winHeight) * (1.0 - (outRatio / inRatio)) / 2.0),
			W: winWidth, H: int32(float32(winHeight) * (outRatio / inRatio))}
	}
	return sdl.Rect{X: int32(float32(winWidth) * (1.0 - (inRatio / outRatio)) / 2.0),
		Y: 0, W: int32(float32(winWidth) * (inRatio / outRatio)), H: winHeight}
}

// InitTransition initializes a transition
func (bg *Background) InitTransition() time.Duration {
	var ret time.Duration = -1
	if bg.reqTextureIndex != -1 {
		if bg.reqTextureIndex != bg.curTextureIndex {
			if bg.reqTextureIndex < len(bg.images) {
				file := bg.images[bg.reqTextureIndex]
				tex, err := img.LoadTexture(bg.display.Renderer, file.Path)
				if err != nil {
					log.Println(err)
					bg.newTexture = nil
				} else {
					defer tex.Destroy()
					_, _, texWidth, texHeight, err := tex.Query()
					if err != nil {
						panic(err)
					}
					winWidth, winHeight := bg.display.Window.GetSize()
					bg.newTexture, err = bg.display.Renderer.CreateTexture(sdl.PIXELFORMAT_RGB888, sdl.TEXTUREACCESS_TARGET,
						winWidth, winHeight)
					if err != nil {
						panic(err)
					}
					bg.display.Renderer.SetRenderTarget(bg.newTexture)
					defer bg.display.Renderer.SetRenderTarget(nil)
					bg.display.Renderer.Clear()
					bg.display.Renderer.SetDrawColor(0, 0, 0, 255)
					bg.display.Renderer.FillRect(nil)
					dst := offsets(float32(texWidth)/float32(texHeight), float32(winWidth)/float32(winHeight),
						winWidth, winHeight)
					bg.display.Renderer.Copy(tex, nil, &dst)
				}
			}
			bg.curTextureIndex = bg.reqTextureIndex
			bg.curTextureSplit = 0
			ret = time.Second
		}
		bg.reqTextureIndex = -1
	}
	return ret
}

// TransitionStep advances the transition.
func (bg *Background) TransitionStep(elapsed time.Duration) {
	bg.curTextureSplit = float32(elapsed) / float32(time.Second)
}

// FinishTransition finalizes the transition.
func (bg *Background) FinishTransition() {
	if bg.texture != nil {
		bg.texture.Destroy()
	}
	bg.texture = bg.newTexture
	bg.curTextureSplit = 0.0
	bg.newTexture = nil
}

// Render renders the module
func (bg *Background) Render() {
	if bg.texture != nil || bg.curTextureSplit != 0 {
		winWidth, winHeight := bg.display.Window.GetSize()
		curSplit := int32(bg.curTextureSplit * float32(winWidth))
		if bg.texture != nil {
			rect := sdl.Rect{X: curSplit, Y: 0, W: winWidth - curSplit, H: winHeight}
			bg.display.Renderer.Copy(bg.texture, &rect, &rect)
		}
		if bg.newTexture != nil {
			rect := sdl.Rect{X: 0, Y: 0, W: curSplit, H: winHeight}
			bg.display.Renderer.Copy(bg.newTexture, &rect, &rect)
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

// GetConfig retrieves the current configuration of the item.
func (bg *Background) GetConfig() interface{} {
	return bg.config
}

// NeedsTransition returns false
func (*Background) NeedsTransition() bool {
	return false
}
