package display

import (
	"github.com/QuestScreen/api"
	"github.com/QuestScreen/api/render"
	"github.com/veandco/go-sdl2/sdl"
)

func keyName(k sdl.Keycode) string {
	switch k {
	case sdl.K_ESCAPE:
		return "Esc"
	case sdl.K_INSERT:
		return "Ins"
	case sdl.K_DELETE:
		return "Del"
	case sdl.K_BACKSPACE:
		return "<-"
	default:
		return sdl.GetKeyName(k)
	}
}

func (d *Display) renderKeyOptions(frame render.Rectangle, actions []KeyAction) error {
	actionImages := make([]render.Image, len(actions))

	fontDesc := api.Font{FamilyIndex: 0, Size: api.ContentFont,
		Style: api.RegularFont, Color: api.RGBA{R: 0, G: 0, B: 0, A: 200}}

	bottomText := d.RenderText("any other key to close", fontDesc)
	defer d.FreeImage(&bottomText)

	fontDesc.Color.A = 230

	maxHeight := bottomText.Height
	for i := range actions {
		actionImages[i] = d.RenderText(actions[i].Description, fontDesc)

		if actionImages[i].Height > maxHeight {
			maxHeight = actionImages[i].Height
		}
	}
	padding := (frame.Height - maxHeight*int32(len(actions)+1)) /
		(2 * int32(len(actions)+1))

	for i := range actions {
		_, frame = frame.Carve(render.North, padding)
		var row render.Rectangle
		row, frame = frame.Carve(render.North, maxHeight)
		_, row = row.Carve(render.West, padding)
		keyFrame, row := row.Carve(render.West, maxHeight)
		keyFrame = keyFrame.Shrink(-2*d.r.unit, -2*d.r.unit)
		keyFrame.Fill(d, api.RGBA{R: 0, G: 0, B: 0, A: 255})
		keyFrame = keyFrame.Shrink(2*d.r.unit, 2*d.r.unit)
		keyFrame.Fill(d, api.RGBA{R: 255, G: 255, B: 255, A: 255})
		keyTex := d.RenderText(keyName(actions[i].Key), fontDesc)
		keyArea := keyFrame.Position(keyTex.Width, keyTex.Height, render.Center,
			render.Middle)
		keyTex.Draw(d, keyArea, 255)
		d.FreeImage(&keyTex)

		action := &actionImages[i]
		actionArea := row.Position(action.Width, action.Height,
			render.Center, render.Middle)
		actionImages[i].Draw(d, actionArea, 255)
		d.FreeImage(action)
	}
	frame.Carve(render.North, padding)
	bottomFrame, _ := frame.Carve(render.North, maxHeight)
	bottomArea := bottomFrame.Position(bottomText.Width, bottomText.Height,
		render.Center, render.Middle)
	bottomText.Draw(d, bottomArea, 255)

	return nil
}

func (d *Display) genPopup(frame render.Rectangle, actions []KeyAction) {
	if d.owner.NumFontFamilies() == 0 {
		return
	}
	canvas, _ := d.CreateCanvas(frame.Width, frame.Height,
		api.RGBA{R: 0, G: 0, B: 0, A: 127}.AsBackground(), render.Nowhere)
	defer canvas.Close()

	frame = frame.Shrink(frame.Width/2, frame.Height/2)
	frame.Fill(d, api.RGBA{R: 0, G: 0, B: 0, A: 255})
	frame = frame.Shrink(2*d.r.unit, 2*d.r.unit)
	frame.Fill(d, api.RGBA{R: 255, G: 255, B: 255, A: 255})
	if err := d.renderKeyOptions(frame, actions); err != nil {
		panic(err)
	}
}
