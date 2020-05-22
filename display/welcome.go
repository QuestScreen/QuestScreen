package display

import (
	"log"
	"net"
	"strconv"

	"github.com/QuestScreen/QuestScreen/generated"
	"github.com/QuestScreen/api/colors"
	"github.com/QuestScreen/api/fonts"
	"github.com/QuestScreen/api/render"

	"github.com/veandco/go-sdl2/sdl"
)

func getIPAddresses() ([]string, error) {
	ret := make([]string, 0, 16)
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	// handle err
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip.IsGlobalUnicast() {
				ret = append(ret, ip.String())
			}
		}
	}
	return ret, nil
}

var fontColor = sdl.Color{R: 0, G: 0, B: 0, A: 200}

func (d *Display) renderIPHint(font fonts.Config, ip string,
	portPart string, frame *render.Rectangle) {
	ipHint := d.RenderText("http://"+ip+portPart, font)
	var row render.Rectangle
	row, *frame = frame.Carve(render.North, ipHint.Height+4*d.r.unit)
	ipArea := row.Position(ipHint.Width, ipHint.Height,
		render.Center, render.Middle)
	ipHint.Draw(d, ipArea, 255)
	d.FreeImage(&ipHint)
}

func (d *Display) genWelcome(frame render.Rectangle, port uint16) error {
	c, _ := d.CreateCanvas(frame.Width, frame.Height,
		colors.RGBA{R: 255, G: 255, B: 255, A: 255}.AsBackground(),
		render.Nowhere)
	defer c.Close()

	logoRow, frame := frame.Carve(render.North, frame.Height/3)
	logoTex, err := d.LoadImageMem(
		generated.MustAsset("web/favicon/android-chrome-512x512.png"), true)
	if err != nil {
		panic("while generating welcome screen: " + err.Error())
	}
	logoArea := logoRow.Position(logoTex.Width, logoTex.Height,
		render.Center, render.Middle)
	heightWithMargin := logoArea.Height + 4*d.r.unit
	if heightWithMargin > logoRow.Height {
		logoArea = logoArea.Scale(float32(logoRow.Height) / float32(heightWithMargin))
	}
	logoTex.Draw(d, logoArea, 255)

	if d.owner.NumFontFamilies() > 0 {
		fontFace := fonts.Config{FamilyIndex: 0, Size: fonts.Large,
			Style: fonts.Regular,
			Color: colors.RGBA{R: 0, G: 0, B: 0, A: 255}}
		titleTex := d.RenderText("Quest Screen", fontFace)
		defer d.FreeImage(&titleTex)
		titleRow, frame := frame.Carve(render.North, titleTex.Height+4*d.r.unit)
		texFrame := titleRow.Position(titleTex.Width, titleTex.Height,
			render.Center, render.Middle)
		titleTex.Draw(d, texFrame, 255)

		fontFace.Size = fonts.Heading
		ips, err := getIPAddresses()
		portPart := ":" + strconv.Itoa(int(port)) + "/"
		if err == nil {
			for i := range ips {
				d.renderIPHint(fontFace, ips[i], portPart, &frame)
			}
		} else {
			log.Println("while getting IPs: " + err.Error())
		}
	}
	d.welcomeTexture = c.Finish()

	return nil
}
