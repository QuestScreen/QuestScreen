package display

import (
	"github.com/flyx/pnpscreen/api"
	"github.com/veandco/go-sdl2/sdl"
)

type keyOption struct {
	key  string
	desc string
}

func (d *Display) shrinkByBorder(rect *sdl.Rect) {
	rect.X += d.owner.DefaultBorderWidth()
	rect.Y += d.owner.DefaultBorderWidth()
	rect.W -= 2 * d.owner.DefaultBorderWidth()
	rect.H -= 2 * d.owner.DefaultBorderWidth()
}

func shrinkTo(rect *sdl.Rect, w int32, h int32) {
	xStep := (rect.W - w) / 2
	yStep := (rect.H - h) / 2
	rect.X += xStep
	rect.Y += yStep
	rect.W -= 2 * xStep
	rect.H -= 2 * yStep
}

func (d *Display) renderKeyOptions(frame *sdl.Rect, options ...keyOption) error {
	surfaces := make([]*sdl.Surface, len(options))
	fontFace := d.owner.Font(0, api.Standard, api.ContentFont)
	var err error
	var bottomText *sdl.Surface
	if bottomText, err = fontFace.RenderUTF8Blended(
		"any other key to close", sdl.Color{R: 0, G: 0, B: 0, A: 200}); err != nil {
		return err
	}
	defer bottomText.Free()

	maxHeight := bottomText.H
	for i := range options {
		if surfaces[i], err = fontFace.RenderUTF8Blended(
			options[i].desc, sdl.Color{R: 0, G: 0, B: 0, A: 230}); err != nil {
			for j := 0; j < i; j++ {
				surfaces[j].Free()
			}
			return err
		}
		//noinspection GoDeferInLoop
		if surfaces[i].H > maxHeight {
			maxHeight = surfaces[i].H
		}
	}
	defer func() {
		for i := range surfaces {
			surfaces[i].Free()
		}
	}()
	padding := (frame.H - maxHeight*int32(len(options)+1)) / (2 * int32(len(options)+1))
	curY := frame.Y + padding
	borderWidth := d.owner.DefaultBorderWidth()
	for i := range options {
		curRect := sdl.Rect{X: frame.X + padding - 2*borderWidth,
			Y: curY - 2*borderWidth, W: maxHeight + 4*borderWidth,
			H: maxHeight + 4*borderWidth}
		d.Renderer.SetDrawColor(0, 0, 0, 255)
		d.Renderer.FillRect(&curRect)
		d.shrinkByBorder(&curRect)
		d.Renderer.SetDrawColor(255, 255, 255, 255)
		d.Renderer.FillRect(&curRect)
		var keySurface *sdl.Surface
		if keySurface, err = fontFace.RenderUTF8Blended(
			options[i].key, sdl.Color{R: 0, G: 0, B: 0, A: 230}); err != nil {
			return err
		}
		keyTex, err := d.Renderer.CreateTextureFromSurface(keySurface)
		if err != nil {
			keySurface.Free()
			return err
		}
		shrinkTo(&curRect, keySurface.W, keySurface.H)
		d.Renderer.Copy(keyTex, nil, &curRect)
		keySurface.Free()
		keyTex.Destroy()

		textTex, err := d.Renderer.CreateTextureFromSurface(surfaces[i])
		if err != nil {
			return err
		}
		curRect = sdl.Rect{X: frame.X + padding + maxHeight + 4*borderWidth,
			Y: curY, W: surfaces[i].W, H: maxHeight}
		shrinkTo(&curRect, surfaces[i].W, surfaces[i].H)
		d.Renderer.Copy(textTex, nil, &curRect)
		textTex.Destroy()

		curY = curY + padding*2 + maxHeight
	}

	var bottomTextTex *sdl.Texture
	if bottomTextTex, err = d.Renderer.CreateTextureFromSurface(bottomText); err != nil {
		return err
	}
	bottomRect := sdl.Rect{X: frame.X, Y: curY, W: frame.W, H: maxHeight}
	shrinkTo(&bottomRect, bottomText.W, bottomText.H)
	d.Renderer.Copy(bottomTextTex, nil, &bottomRect)
	bottomTextTex.Destroy()
	return nil
}

func (d *Display) genPopup(width int32, height int32) {
	var err error
	d.popupTexture, err = d.Renderer.CreateTexture(sdl.PIXELFORMAT_RGB888, sdl.TEXTUREACCESS_TARGET,
		width, height)
	if err != nil {
		panic(err)
	}
	d.Renderer.SetRenderTarget(d.popupTexture)
	defer d.Renderer.SetRenderTarget(nil)
	d.Renderer.Clear()
	d.Renderer.SetDrawColor(0, 0, 0, 127)
	d.Renderer.FillRect(nil)
	rect := sdl.Rect{X: width / 4, Y: height / 4, W: width / 2, H: height / 2}
	d.Renderer.SetDrawColor(0, 0, 0, 255)
	d.Renderer.FillRect(&rect)
	d.shrinkByBorder(&rect)
	d.Renderer.SetDrawColor(255, 255, 255, 255)
	d.Renderer.FillRect(&rect)

	if err = d.renderKeyOptions(&rect, keyOption{key: "X", desc: "Quit"},
		keyOption{key: "S", desc: "Shutdown"}); err != nil {
		panic(err)
	}
	d.popupTexture.SetBlendMode(sdl.BLENDMODE_BLEND)
}
