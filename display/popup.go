package display

import (
	"github.com/QuestScreen/api"
	"github.com/veandco/go-sdl2/sdl"
)

type keyOption struct {
	key  string
	desc string
}

func (d *Display) shrinkByBorder(rect *sdl.Rect) {
	rect.X += d.unit
	rect.Y += d.unit
	rect.W -= 2 * d.unit
	rect.H -= 2 * d.unit
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
	for i := range options {
		curRect := sdl.Rect{X: frame.X + padding - 2*d.unit,
			Y: curY - 2*d.unit, W: maxHeight + 4*d.unit,
			H: maxHeight + 4*d.unit}
		d.Backend.SetDrawColor(0, 0, 0, 255)
		d.Backend.FillRect(&curRect)
		d.shrinkByBorder(&curRect)
		d.Backend.SetDrawColor(255, 255, 255, 255)
		d.Backend.FillRect(&curRect)
		var keySurface *sdl.Surface
		if keySurface, err = fontFace.RenderUTF8Blended(
			options[i].key, sdl.Color{R: 0, G: 0, B: 0, A: 230}); err != nil {
			return err
		}
		keyTex, err := d.Backend.CreateTextureFromSurface(keySurface)
		if err != nil {
			keySurface.Free()
			return err
		}
		shrinkTo(&curRect, keySurface.W, keySurface.H)
		d.Backend.Copy(keyTex, nil, &curRect)
		keySurface.Free()
		keyTex.Destroy()

		textTex, err := d.Backend.CreateTextureFromSurface(surfaces[i])
		if err != nil {
			return err
		}
		curRect = sdl.Rect{X: frame.X + padding + maxHeight + 4*d.unit,
			Y: curY, W: surfaces[i].W, H: maxHeight}
		shrinkTo(&curRect, surfaces[i].W, surfaces[i].H)
		d.Backend.Copy(textTex, nil, &curRect)
		textTex.Destroy()

		curY = curY + padding*2 + maxHeight
	}

	var bottomTextTex *sdl.Texture
	if bottomTextTex, err = d.Backend.CreateTextureFromSurface(bottomText); err != nil {
		return err
	}
	bottomRect := sdl.Rect{X: frame.X, Y: curY, W: frame.W, H: maxHeight}
	shrinkTo(&bottomRect, bottomText.W, bottomText.H)
	d.Backend.Copy(bottomTextTex, nil, &bottomRect)
	bottomTextTex.Destroy()
	return nil
}

func (d *Display) genPopup(width int32, height int32) {
	if d.owner.NumFontFamilies() == 0 {
		return
	}
	var err error
	d.popupTexture, err = d.Backend.CreateTexture(sdl.PIXELFORMAT_RGB888, sdl.TEXTUREACCESS_TARGET,
		width, height)
	if err != nil {
		panic(err)
	}
	d.Backend.SetRenderTarget(d.popupTexture)
	defer d.Backend.SetRenderTarget(nil)
	d.Backend.Clear()
	d.Backend.SetDrawColor(0, 0, 0, 127)
	d.Backend.FillRect(nil)
	rect := sdl.Rect{X: width / 4, Y: height / 4, W: width / 2, H: height / 2}
	d.Backend.SetDrawColor(0, 0, 0, 255)
	d.Backend.FillRect(&rect)
	d.shrinkByBorder(&rect)
	d.Backend.SetDrawColor(255, 255, 255, 255)
	d.Backend.FillRect(&rect)

	if err = d.renderKeyOptions(&rect, keyOption{key: "X", desc: "Quit"},
		keyOption{key: "S", desc: "Shutdown"}); err != nil {
		panic(err)
	}
	d.popupTexture.SetBlendMode(sdl.BLENDMODE_BLEND)
}
