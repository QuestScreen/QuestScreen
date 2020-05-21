package display

// #ifdef __APPLE__
// #cgo LDFLAGS: -framework OpenGL
// #endif
// #include "renderer.h"
import "C"
import (
	"log"
	"unsafe"

	"github.com/QuestScreen/api/colors"
	"github.com/QuestScreen/api/fonts"
	"github.com/QuestScreen/api/render"
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
)

func toArr(c colors.RGBA) [4]C.uint8_t {
	return [4]C.uint8_t{
		C.uint8_t(c.R), C.uint8_t(c.G), C.uint8_t(c.B), C.uint8_t(c.A)}
}

// this struct contains data used for implementing api.Renderer within Display.
type renderer struct {
	engine        C.engine_t
	width, height int32
	unit          int32
	textureCache  []render.Image
}

func (r *renderer) init(width int32, height int32, numTextures int) {
	if !C.engine_init(&r.engine) {
		panic("couldn't initialize rendering engine")
	}
	r.width = width
	r.height = height
	if width < height {
		r.unit = width / 144
	} else {
		r.unit = height / 144
	}
	C.glViewport(0, 0, C.GLsizei(width), C.GLsizei(height))
	r.textureCache = make([]render.Image, numTextures)
}

func (r *renderer) close() {
	C.engine_close(&r.engine)
}

func (r *renderer) clear() {
	C.glClearColor(1.0, 1.0, 1.0, 1.0)
	C.glClear(C.GL_COLOR_BUFFER_BIT)
}

// toInternalCoords alters a transformation which transforms a square
// (-0.5, -0.5) -- (0.5, 0.5) in a coordinate system where the screen
// is (0,0) -- (d.r.width,d.r.height), to a transformation which transforms a
// square (0,0) -- (1,1) in a coordinate system where the screen is
// (-1.0, -1.0) -- (1.0, 1.0) (the OpenGL viewport).
func (d *Display) toInternalCoords(t render.Transform, flip bool) render.Transform {
	ret := render.Identity().Translate(-1.0, -1.0).Scale(
		2.0/float32(d.r.width), 2.0/float32(d.r.height)).Compose(t)
	if flip {
		ret = ret.Scale(1.0, -1.0)
	}
	return ret.Translate(-0.5, -0.5)
}

// OutputSize returns a rectangle that describes the dimensions in pixels
// of the current rendering area. X and Y are always 0.
func (d *Display) OutputSize() render.Rectangle {
	return render.Rectangle{X: 0, Y: 0, Width: d.r.width, Height: d.r.height}
}

// FillRect fills the rectangle with the specified dimensions with the
// specified color. The rectangle is positions via the given transformation.
func (d *Display) FillRect(t render.Transform, color colors.RGBA) {
	t = d.toInternalCoords(t, false)
	cArr := toArr(color)
	C.draw_rect(&d.r.engine, (*C.float)(&t[0]), &cArr[0], false)
}

func (r *renderer) surfaceToTexture(surface *sdl.Surface) (render.Image, error) {
	var glFormat C.GLenum
	switch surface.Format.Format {
	case sdl.PIXELFORMAT_ABGR8888:
		glFormat = C.GL_RGBA
		log.Println("surface format: ABGR")
	case sdl.PIXELFORMAT_RGBA8888:
		glFormat = C.GL_RGBA
		rgba, err := surface.ConvertFormat(sdl.PIXELFORMAT_ABGR8888, 0)
		surface.Free()
		if err != nil {
			return render.EmptyImage(), err
		}
		surface = rgba
		log.Println("surface format: RGBA (did convert to ABGR)")
	case sdl.PIXELFORMAT_BGR888:
		glFormat = C.GL_RGB
		log.Println("surface format: BGR")
	case sdl.PIXELFORMAT_RGB888:
		glFormat = C.GL_RGB
		rgba, err := surface.ConvertFormat(sdl.PIXELFORMAT_BGR888, 0)
		surface.Free()
		if err != nil {
			return render.EmptyImage(), err
		}
		surface = rgba
		log.Println("surface format: RBG (did convert to BGR)")
	default:
		log.Printf("surface format: converting to ABGR from %d\n", surface.Format.Format)
		rgba, err := surface.ConvertFormat(sdl.PIXELFORMAT_ABGR8888, 0)
		surface.Free()
		if err != nil {
			return render.EmptyImage(), err
		}
		surface = rgba
		glFormat = C.GL_RGBA
	}
	var ret render.Image
	var expectedPixelBytes uint8
	switch glFormat {
	case C.GL_RGBA:
		expectedPixelBytes = 4
		ret.HasAlpha = true
	case C.GL_RGB:
		expectedPixelBytes = 3
		ret.HasAlpha = false
	default:
		panic("unexpected format")
	}
	if surface.Format.BytesPerPixel != expectedPixelBytes {
		panic("surface has wrong number of BytesPerPixel")
	}
	ret.TextureID = uint32(C.gen_texture(&r.engine, glFormat, C.GLsizei(surface.W),
		C.GLsizei(surface.H), unsafe.Pointer(&surface.Pixels()[0])))
	ret.Width = surface.W
	ret.Height = surface.H
	ret.Flipped = false
	surface.Free()
	return ret, nil
}

// LoadImageFile loads an image file from the specified path.
// if an error is returned, the returned image is empty.
func (d *Display) LoadImageFile(path string) (render.Image, error) {
	surface, err := img.Load(path)
	if err != nil {
		return render.EmptyImage(), err
	}
	return d.r.surfaceToTexture(surface)
}

// LoadImageMem loads an image from data in memory.
// if an error is returned, the returned image is empty.
func (d *Display) LoadImageMem(data []byte) (render.Image, error) {
	logoStream, err := sdl.RWFromMem(data)
	if err != nil {
		panic(err)
	}
	logo, err := img.LoadRW(logoStream, true)
	if err != nil {
		panic(err)
	}
	return d.r.surfaceToTexture(logo)
}

// FreeImage destroys the texture associated with the image (if one exists)
// and sets i to be the empty image. Does nothing on empty images.
func (d *Display) FreeImage(i *render.Image) {
	if !i.IsEmpty() {
		C.glDeleteTextures(1, (*C.GLuint)(&i.TextureID))
		i.Width = 0
	}
}

// DrawImage renders the given image if it is not empty, using the given
// transformation. alpha modifies the image's opacity.
func (d *Display) DrawImage(image render.Image, t render.Transform, alpha uint8) {
	t = d.toInternalCoords(t, image.Flipped)
	C.draw_image(&d.r.engine, C.GLuint(image.TextureID),
		(*C.float)(&t[0]), C.uint8_t(alpha), C.bool(image.HasAlpha))
}

// Unit is the scaled smallest unit in pixels.
func (d *Display) Unit() int32 {
	return d.r.unit
}

// RenderText renders the given text with the given font into an image with
// transparent background.
// Returns an empty image if it wasn't able to create the texture.
func (d *Display) RenderText(text string, font fonts.Config) render.Image {
	face := d.owner.Font(font.FamilyIndex, font.Style, font.Size)
	bottomText, err := face.RenderUTF8Blended(
		text, sdl.Color{R: font.Color.R, G: font.Color.G, B: font.Color.B,
			A: font.Color.A})
	if err != nil {
		panic(err)
	}
	ret, err := d.r.surfaceToTexture(bottomText)
	if err != nil {
		panic(err)
	}
	return ret
}

type canvas struct {
	*renderer
	prevFb       C.GLuint
	fb, tex      C.GLuint
	alpha        bool
	prevW, prevH int32
}

func (c *canvas) Finish() (ret render.Image) {
	if c.fb == 0 {
		panic("tried to finish already closed canvas")
	}
	C.finish_canvas(&c.renderer.engine, c.fb, c.prevFb)
	c.fb = 0
	ret = render.Image{Width: c.width, Height: c.height, TextureID: uint32(c.tex),
		Flipped: true, HasAlpha: c.alpha}
	c.renderer.width = c.prevW
	c.renderer.height = c.prevH
	C.glViewport(0, 0, C.GLsizei(c.width), C.GLsizei(c.height))
	return
}

func (c *canvas) Close() {
	if c.fb != 0 {
		c.fb = 0
		c.renderer.width = c.prevW
		c.renderer.height = c.prevH
		C.glViewport(0, 0, C.GLsizei(c.width), C.GLsizei(c.height))
		C.destroy_canvas(&c.renderer.engine, c.fb, c.tex, c.prevFb)
	}
}

func (d *Display) drawMasked(
	bg colors.Background, content render.Rectangle) bool {
	if bg.TextureIndex == -1 {
		return false
	}
	loadedTexture := &d.r.textureCache[bg.TextureIndex]
	if loadedTexture.IsEmpty() {
		path := d.owner.GetTextures()[bg.TextureIndex].Path()
		surface, err := img.Load(path)
		if err != nil {
			log.Printf("unable to load %s: %s\n", path, err.Error())
			return false
		}
		if surface.Format.Format != sdl.PIXELFORMAT_INDEX8 {
			grayscale, err := surface.ConvertFormat(sdl.PIXELFORMAT_INDEX8, 0)
			if err != nil {
				log.Printf("could not convert %s to grayscale: %s\n", path,
					err.Error())
				return false
			}
			surface.Free()
			surface = grayscale
		}
		if surface.Format.BytesPerPixel != 1 {
			panic("grayscale image has wrong number of bytes per pixel")
		}
		*loadedTexture = render.Image{
			TextureID: uint32(C.gen_texture(&d.r.engine, C.GL_SINGLE_VALUE_COLOR,
				C.GLsizei(surface.W), C.GLsizei(surface.H),
				unsafe.Pointer(&surface.Pixels()[0]))),
			Width: surface.W, Height: surface.H, Flipped: false, HasAlpha: true}
		surface.Free()
	}
	posTrans := d.toInternalCoords(content.Transformation(), false)
	texTrans := render.Identity().Scale(
		float32(d.r.width)/float32(loadedTexture.Width),
		float32(d.r.height)/float32(loadedTexture.Height))
	pColor := toArr(bg.Primary)
	sColor := toArr(bg.Secondary)
	C.draw_masked(&d.r.engine, C.GLuint(loadedTexture.TextureID),
		(*C.float)(&posTrans[0]), (*C.float)(&texTrans[0]), &pColor[0], &sColor[0])
	return true
}

// CreateCanvas creates a canvas to draw content into, and fills it with the
// given background.
func (d *Display) CreateCanvas(innerWidth, innerHeight int32,
	bg colors.Background, borders render.Directions) (c render.Canvas, content render.Rectangle) {
	ret := &canvas{renderer: &d.r, prevW: d.r.width, prevH: d.r.height}
	width, height := innerWidth, innerHeight
	content = render.Rectangle{
		X: 0, Y: 0, Width: innerWidth, Height: innerHeight}
	if borders&render.East != 0 {
		width += d.r.unit
	}
	if borders&render.West != 0 {
		width += d.r.unit
		content.X += d.r.unit
	}
	if borders&render.North != 0 {
		height += d.r.unit
	}
	if borders&render.South != 0 {
		height += d.r.unit
		content.Y += d.r.unit
	}
	ret.alpha = bg.Primary.A != 255 ||
		(bg.TextureIndex != -1 && bg.Secondary.A != 255)

	C.create_canvas(&d.r.engine, C.GLsizei(width), C.GLsizei(height),
		&ret.prevFb, &ret.fb, &ret.tex, C.bool(ret.alpha))
	if ret.fb == 0 {
		panic("failed to create canvas")
	}
	d.r.width, d.r.height = width, height

	C.glViewport(0, 0, C.GLsizei(width), C.GLsizei(height))

	if !d.drawMasked(bg, content) {
		t := d.toInternalCoords(content.Transformation(), false)
		pArr := toArr(bg.Primary)
		C.draw_rect(&d.r.engine, (*C.float)(&t[0]), &pArr[0], true)
	}

	c = ret
	return
}

func (r *renderer) canvasCount() uint8 {
	return uint8(C.canvas_count(&r.engine))
}
