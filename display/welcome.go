package display

import (
	"log"
	"net"
	"strconv"
	"unsafe"

	"github.com/QuestScreen/QuestScreen/api"
	"github.com/QuestScreen/QuestScreen/generated"

	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
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

func renderIPHint(r *sdl.Renderer, font *ttf.Font, ip string,
	portPart string, width int32, y int32) int32 {
	var ipHint *sdl.Surface
	var err error
	if ipHint, err = font.RenderUTF8Blended(
		"http://"+ip+portPart, fontColor); err != nil {
		log.Println("while rendering ip: " + err.Error())
		return 0
	}
	defer ipHint.Free()
	ipHintTexture, err := r.CreateTextureFromSurface(ipHint)
	if err != nil {
		log.Println("while rendering ip: " + err.Error())
		return 0
	}
	defer ipHintTexture.Destroy()
	ipHintRect := sdl.Rect{X: 0, Y: y, W: width, H: ipHint.H}
	shrinkTo(&ipHintRect, ipHint.W, ipHint.H)
	r.Copy(ipHintTexture, nil, &ipHintRect)
	return ipHint.H
}

func (d *Display) genWelcome(width int32, height int32, port uint16) error {
	var err error
	d.welcomeTexture, err = d.Backend.CreateTexture(sdl.PIXELFORMAT_RGB888,
		sdl.TEXTUREACCESS_TARGET, width, height)
	if err != nil {
		return err
	}
	d.Backend.SetRenderTarget(d.welcomeTexture)
	defer d.Backend.SetRenderTarget(nil)
	d.Backend.Clear()
	d.Backend.SetDrawColor(255, 255, 255, 255)
	d.Backend.FillRect(nil)

	logoSource := generated.MustAsset("web/favicon/android-chrome-512x512.png")
	logoStream := sdl.RWFromMem(unsafe.Pointer(&logoSource[0]), len(logoSource))
	logo, err := img.LoadTextureRW(d.Backend, logoStream, true)
	if err != nil {
		panic(err)
	}
	_, _, logoW, _, _ := logo.Query()
	var logoRect sdl.Rect
	if logoW > width/3 {
		logoW = width / 3
	}
	if logoW > height/3 {
		logoW = height / 3
	}
	logoRect = sdl.Rect{X: (width - logoW) / 2, Y: 0, W: logoW, H: logoW}
	d.Backend.Copy(logo, nil, &logoRect)

	if d.owner.NumFontFamilies() > 0 {
		fontFace := d.owner.Font(0, api.Standard, api.LargeFont)
		var title *sdl.Surface
		if title, err = fontFace.RenderUTF8Blended(
			"Quest Screen", fontColor); err != nil {
			return err
		}
		defer title.Free()
		titleTexture, err := d.Backend.CreateTextureFromSurface(title)
		if err != nil {
			return err
		}
		defer titleTexture.Destroy()
		titleRect := sdl.Rect{X: 0, Y: logoW, W: width, H: height/2 - logoW}
		shrinkTo(&titleRect, title.W, title.H)
		d.Backend.Copy(titleTexture, nil, &titleRect)

		ipFontFace := d.owner.Font(0, api.Standard, api.HeadingFont)
		y := height/2 + 15
		ips, err := getIPAddresses()
		portPart := ":" + strconv.Itoa(int(port)) + "/"
		if err == nil {
			for i := range ips {
				h := renderIPHint(d.Backend, ipFontFace, ips[i], portPart, width, y)
				if h > 0 {
					y = y + h + 15
				}
			}
		} else {
			log.Println("while getting IPs: " + err.Error())
		}
	}

	return nil
}
