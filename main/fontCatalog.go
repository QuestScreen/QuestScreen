package main

import (
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/QuestScreen/api"
	"github.com/veandco/go-sdl2/ttf"
)

// LoadedFontStyle describes a font style. it implements api.StyledFont.
// faces is nil for sizes that have not yet been loaded into memory.
// path points to the file containing the font.
type LoadedFontStyle struct {
	faces       [api.NumFontSizes]*ttf.Font
	path        string
	fontSizeMap [api.NumFontSizes]int32
}

// LoadedFontFamily is a font family that is available for usage.
// in implements api.FontFamily.
type LoadedFontFamily struct {
	loadedFaces [api.NumFontStyles]LoadedFontStyle
	name        string
}

func (qs *QuestScreen) loadFonts(dir string, fontSizeMap [api.NumFontSizes]int32) {
	files, err := ioutil.ReadDir(dir)

	if err != nil {
		log.Println(err)
		return
	}

	for _, file := range files {
		if !file.IsDir() {
			path := filepath.Join(dir, file.Name())
			if font, err := ttf.OpenFont(path, int(fontSizeMap[api.ContentFont])); err != nil {
				log.Println(err)
			} else {
				familyName := font.FaceFamilyName()
				if familyName == "" {
					familyName = file.Name()
				}
				isBold := (font.GetStyle() & ttf.STYLE_BOLD) != 0
				isItalic := (font.GetStyle() & ttf.STYLE_ITALIC) != 0

				var family *LoadedFontFamily
				for i := range qs.fonts {
					if qs.fonts[i].Name() == familyName {
						family = &qs.fonts[i]
						break
					}
				}
				if family == nil {
					qs.fonts = append(qs.fonts, LoadedFontFamily{name: familyName})
					family = &qs.fonts[len(qs.fonts)-1]
				}
				fontList := [api.NumFontSizes]*ttf.Font{}
				fontList[api.ContentFont] = font
				if isBold {
					if isItalic {
						family.loadedFaces[api.BoldItalicFont] = LoadedFontStyle{
							faces: fontList, path: path, fontSizeMap: fontSizeMap}
					} else {
						family.loadedFaces[api.BoldFont] = LoadedFontStyle{
							faces: fontList, path: path, fontSizeMap: fontSizeMap}
					}
				} else if isItalic {
					family.loadedFaces[api.ItalicFont] = LoadedFontStyle{
						faces: fontList, path: path, fontSizeMap: fontSizeMap}
				} else {
					family.loadedFaces[api.RegularFont] = LoadedFontStyle{
						faces: fontList, path: path, fontSizeMap: fontSizeMap}
				}
			}
		}
	}
}

// Font returns the font at the given size;
// loads that size if it isn't already available.
func (style *LoadedFontStyle) Font(size api.FontSize) *ttf.Font {
	ret := style.faces[size]
	if ret == nil {
		newSize, err := ttf.OpenFont(style.path, int(style.fontSizeMap[size]))
		if err != nil {
			panic(err)
		}
		style.faces[size] = newSize
		ret = newSize
	}
	return ret
}

// Styled returns the requested style if available.
// A fallback is returned if the requested style isn't available.
func (family *LoadedFontFamily) Styled(style api.FontStyle) *LoadedFontStyle {
	var ret *LoadedFontStyle

	for curStyle := style; curStyle >= 0; curStyle-- {
		ret = &family.loadedFaces[curStyle]
		if ret.path != "" {
			return ret
		}
	}
	for curStyle := style + 1; curStyle < api.NumFontStyles; curStyle++ {
		ret = &family.loadedFaces[curStyle]
		if ret.path != "" {
			return ret
		}
	}
	panic("loaded font " + family.name + " has no available styles!")
}

// Name returns the font family's name.
func (family *LoadedFontFamily) Name() string {
	return family.name
}
