package background

import (
	"github.com/flyx/rpscreen/module"
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Background struct {
	texture, newTexture              module.Texture
	reqTextureIndex, curTextureIndex int
	images                           []module.Resource
	curTextureSplit                  float32
}

func (bg *Background) Init(common *module.SceneCommon) error {
	bg.images = common.ListFiles(bg, "")
	bg.texture = module.Texture{Ratio: 1}
	bg.newTexture = module.Texture{Ratio: 1}

	bg.reqTextureIndex = -1
	bg.curTextureIndex = len(bg.images)
	bg.curTextureSplit = 0
	return nil
}

func (*Background) Name() string {
	return "Background Image"
}

func (*Background) InternalName() string {
	return "background"
}

func (bg *Background) UI() template.HTML {
	var builder module.UIBuilder
	shownIndex := bg.reqTextureIndex
	if shownIndex == -1 {
		shownIndex = bg.curTextureIndex
	}
	builder.StartForm(bg, "image", "Select Image")
	builder.StartSelect("", "image", "value")
	builder.Option("", shownIndex == len(bg.images), "None")
	for index, file := range bg.images {
		builder.Option(strconv.Itoa(index), shownIndex == index, file.Name)
	}
	builder.EndSelect()
	builder.SubmitButton("Update")
	builder.EndForm()

	return builder.Finish()
}

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
		var returns module.EndpointReturn
		if returnPartial {
			returns = module.EndpointReturnEmpty
		} else {
			returns = module.EndpointReturnRedirect
		}
		module.WriteEndpointHeader(w, returns)
		return true
	} else {
		http.Error(w, "404 not found: "+suffix, http.StatusNotFound)
		return false
	}
}

func offsets(inRatio float32, outRatio float32, winWidth int32, winHeight int32) sdl.Rect {
	if inRatio > outRatio {
		return sdl.Rect{X: 0, Y: int32(float32(winHeight) * (1.0 - (outRatio / inRatio)) / 2.0),
			W: winWidth, H: int32(float32(winHeight) * (outRatio / inRatio))}
	} else {
		return sdl.Rect{X: int32(float32(winWidth) * (1.0 - (inRatio / outRatio)) / 2.0),
			Y: 0, W: int32(float32(winWidth) * (inRatio / outRatio)), H: winHeight}
	}
}

func (bg *Background) InitTransition(common *module.SceneCommon) time.Duration {
	var ret time.Duration = -1
	if bg.reqTextureIndex != -1 {
		if bg.reqTextureIndex != bg.curTextureIndex {
			if bg.reqTextureIndex < len(bg.images) {
				file := bg.images[bg.reqTextureIndex]
				tex, err := img.LoadTexture(common.Renderer, file.Path)
				if err != nil {
					log.Println(err)
					bg.newTexture.Tex = nil
				} else {
					defer tex.Destroy()
					_, _, texWidth, texHeight, err := tex.Query()
					if err != nil {
						panic(err)
					}
					winWidth, winHeight := common.Window.GetSize()
					bg.newTexture.Tex, err = common.Renderer.CreateTexture(sdl.PIXELFORMAT_RGB888, sdl.TEXTUREACCESS_TARGET,
						winWidth, winHeight)
					if err != nil {
						panic(err)
					}
					common.Renderer.SetRenderTarget(bg.newTexture.Tex)
					defer common.Renderer.SetRenderTarget(nil)
					common.Renderer.Clear()
					common.Renderer.SetDrawColor(255, 255, 255, 255)
					common.Renderer.FillRect(&sdl.Rect{X: 0, Y: 0, W: int32(winWidth), H: int32(winHeight)})
					dst := offsets(float32(texWidth)/float32(texHeight), float32(common.Width)/float32(common.Height),
						winWidth, winHeight)
					common.Renderer.Copy(tex, nil, &dst)
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

func (bg *Background) TransitionStep(common *module.SceneCommon, elapsed time.Duration) {
	bg.curTextureSplit = float32(elapsed) / float32(time.Second)
}

func (bg *Background) FinishTransition(common *module.SceneCommon) {
	if bg.texture.Tex != nil {
		bg.texture.Tex.Destroy()
	}
	bg.texture = bg.newTexture
	bg.curTextureSplit = 0.0
	bg.newTexture = module.Texture{Ratio: 1}
}

func (bg *Background) Render(common *module.SceneCommon) {
	if bg.texture.Tex != nil || bg.curTextureSplit != 0 {
		winWidth, winHeight := common.Window.GetSize()
		curSplit := int32(bg.curTextureSplit * float32(winWidth))
		if bg.texture.Tex != nil {
			rect := sdl.Rect{X: curSplit, Y: 0, W: winWidth - curSplit, H: winHeight}
			common.Renderer.Copy(bg.texture.Tex, &rect, &rect)
		}
		if bg.newTexture.Tex != nil {
			rect := sdl.Rect{X: 0, Y: 0, W: curSplit, H: winHeight}
			common.Renderer.Copy(bg.newTexture.Tex, &rect, &rect)
		}
	}
}
