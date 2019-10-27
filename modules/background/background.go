package background

import (
	"log"
	"time"

	"github.com/flyx/rpscreen/data"
	"github.com/flyx/rpscreen/display"
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
)

type backgroundConfig struct{}

// Background is a module for painting background images
type Background struct {
	state
	display                *display.Display
	config                 *backgroundConfig
	curTexture, newTexture *sdl.Texture
	curTextureIndex        int
	curTextureSplit        float32
	reqTextureIndex        chan int
}

// Init initializes the module
func (bg *Background) Init(display *display.Display, store *data.Store) error {
	bg.display = display
	bg.curTexture = nil
	bg.newTexture = nil
	bg.curTextureSplit = 0
	bg.curTextureIndex = -1
	bg.reqTextureIndex = make(chan int, 4)
	bg.state.owner = bg
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

func offsets(inRatio float32, outRatio float32, winWidth int32, winHeight int32) sdl.Rect {
	if inRatio > outRatio {
		return sdl.Rect{X: 0, Y: int32(float32(winHeight) * (1.0 - (outRatio / inRatio)) / 2.0),
			W: winWidth, H: int32(float32(winHeight) * (outRatio / inRatio))}
	}
	return sdl.Rect{X: int32(float32(winWidth) * (1.0 - (inRatio / outRatio)) / 2.0),
		Y: 0, W: int32(float32(winWidth) * (inRatio / outRatio)), H: winHeight}
}

func (bg *Background) genTexture(index int) *sdl.Texture {
	file := bg.state.resources[index]
	tex, err := img.LoadTexture(bg.display.Renderer, file.Path)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer tex.Destroy()
	_, _, texWidth, texHeight, err := tex.Query()
	if err != nil {
		panic(err)
	}
	winWidth, winHeight := bg.display.Window.GetSize()
	newTexture, err := bg.display.Renderer.CreateTexture(
		sdl.PIXELFORMAT_RGB888, sdl.TEXTUREACCESS_TARGET, winWidth, winHeight)
	if err != nil {
		panic(err)
	}
	bg.display.Renderer.SetRenderTarget(newTexture)
	defer bg.display.Renderer.SetRenderTarget(nil)
	bg.display.Renderer.Clear()
	bg.display.Renderer.SetDrawColor(0, 0, 0, 255)
	bg.display.Renderer.FillRect(nil)
	dst := offsets(float32(texWidth)/float32(texHeight),
		float32(winWidth)/float32(winHeight), winWidth, winHeight)
	bg.display.Renderer.Copy(tex, nil, &dst)
	return newTexture
}

// InitTransition initializes a transition
func (bg *Background) InitTransition() time.Duration {
	var ret time.Duration = -1
	var newTextureIndex int
	// retrieve newest requested texture index
	for available := true; available; {
		select {
		case newTextureIndex, available = <-bg.reqTextureIndex:
			break
		default:
			available = false
			break
		}
	}

	if newTextureIndex != -1 {
		if newTextureIndex != bg.curTextureIndex {
			if newTextureIndex < len(bg.state.resources) {
				bg.newTexture = bg.genTexture(newTextureIndex)
			}
			bg.curTextureIndex = newTextureIndex
			bg.curTextureSplit = 0
			ret = time.Second
		}
	}
	return ret
}

// TransitionStep advances the transition.
func (bg *Background) TransitionStep(elapsed time.Duration) {
	bg.curTextureSplit = float32(elapsed) / float32(time.Second)
}

// FinishTransition finalizes the transition.
func (bg *Background) FinishTransition() {
	if bg.curTexture != nil {
		bg.curTexture.Destroy()
	}
	bg.curTexture = bg.newTexture
	bg.curTextureSplit = 0.0
	bg.newTexture = nil
}

// Render renders the module
func (bg *Background) Render() {
	if bg.curTexture != nil || bg.curTextureSplit != 0 {
		winWidth, winHeight := bg.display.Window.GetSize()
		curSplit := int32(bg.curTextureSplit * float32(winWidth))
		if bg.curTexture != nil {
			rect := sdl.Rect{X: curSplit, Y: 0, W: winWidth - curSplit, H: winHeight}
			bg.display.Renderer.Copy(bg.curTexture, &rect, &rect)
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
func (bg *Background) SetConfig(config interface{}) bool {
	bg.config = config.(*backgroundConfig)
	return false
}

// GetConfig retrieves the current configuration of the item.
func (bg *Background) GetConfig() interface{} {
	return bg.config
}

// GetState returns the current state.
func (bg *Background) GetState() data.ModuleState {
	return &bg.state
}

// RebuildState queries the texture index through the channel and immediately
// sets that texture as background.
func (bg *Background) RebuildState() {
	select {
	case bg.curTextureIndex = <-bg.reqTextureIndex:
		for empty := false; !empty; {
			select {
			case bg.curTextureIndex = <-bg.reqTextureIndex:
			default:
				empty = true
			}
			if bg.curTexture != nil {
				bg.curTexture.Destroy()
			}
			bg.curTexture = bg.genTexture(bg.curTextureIndex)
		}
	default:
	}
}
