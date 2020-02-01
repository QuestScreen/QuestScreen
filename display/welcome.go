package display

import (
	"strconv"

	"github.com/QuestScreen/QuestScreen/api"

	"github.com/veandco/go-sdl2/sdl"
)

func (d *Display) genWelcome(width int32, height int32, port uint16) error {
	// get outbound IP address
	/*conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return err
	}
	defer conn.Close()*/

	outboundAddr := /*conn.LocalAddr().(*net.UDPAddr).IP.String()*/ "localhost"

	var err error
	d.welcomeTexture, err = d.Renderer.CreateTexture(sdl.PIXELFORMAT_RGB888, sdl.TEXTUREACCESS_TARGET, width, height)
	if err != nil {
		return err
	}
	d.Renderer.SetRenderTarget(d.welcomeTexture)
	defer d.Renderer.SetRenderTarget(nil)
	d.Renderer.Clear()
	d.Renderer.SetDrawColor(255, 255, 255, 255)
	d.Renderer.FillRect(nil)

	fontFace := d.owner.Font(0, api.Standard, api.HugeFont)
	var title *sdl.Surface
	if title, err = fontFace.RenderUTF8Blended(
		"Quest Screen", sdl.Color{R: 0, G: 0, B: 0, A: 200}); err != nil {
		return err
	}
	defer title.Free()
	titleTexture, err := d.Renderer.CreateTextureFromSurface(title)
	if err != nil {
		return err
	}
	defer titleTexture.Destroy()
	titleRect := sdl.Rect{X: 0, Y: 0, W: width, H: height / 2}
	shrinkTo(&titleRect, title.W, title.H)
	d.Renderer.Copy(titleTexture, nil, &titleRect)

	ipFontFace := d.owner.Font(0, api.Standard, api.HeadingFont)
	var ipHint *sdl.Surface
	if ipHint, err = ipFontFace.RenderUTF8Blended(
		"Connect to http://"+outboundAddr+":"+strconv.Itoa(int(port))+"/",
		sdl.Color{R: 0, G: 0, B: 0, A: 200}); err != nil {
		return err
	}
	defer ipHint.Free()
	ipHintTexture, err := d.Renderer.CreateTextureFromSurface(ipHint)
	if err != nil {
		return err
	}
	defer ipHintTexture.Destroy()
	ipHintRect := sdl.Rect{X: 0, Y: height / 2, W: width, H: height / 2}
	shrinkTo(&ipHintRect, ipHint.W, ipHint.H)
	d.Renderer.Copy(ipHintTexture, nil, &ipHintRect)

	return nil
}
