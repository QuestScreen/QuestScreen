package data

import (
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/veandco/go-sdl2/ttf"
)

// LoadedFontStyle describes a font style.
// faces is nil for sizes that have not yet been loaded into memory.
// path points to the file containing the font.
type LoadedFontStyle struct {
	faces [NumFontSizes]*ttf.Font
	path  string
}

// LoadedFontFamily is a font family that is available for usage.
type LoadedFontFamily struct {
	loadedFaces [NumFontStyles]LoadedFontStyle
	Name        string
}

// CreateFontCatalog loads all the fonts in the fonts directory
func CreateFontCatalog(dataDir string, defaultSize int32) []LoadedFontFamily {
	fontPath := filepath.Join(dataDir, "fonts")
	files, err := ioutil.ReadDir(fontPath)

	if err == nil {
		catalog := make([]LoadedFontFamily, 0, len(files))
		for _, file := range files {
			if !file.IsDir() {
				path := filepath.Join(fontPath, file.Name())
				if font, err := ttf.OpenFont(path, int(defaultSize)); err != nil {
					log.Println(err)
				} else {
					familyName := font.FaceFamilyName()
					if familyName == "" {
						familyName = file.Name()
					}
					isBold := (font.GetStyle() & ttf.STYLE_BOLD) != 0
					isItalic := (font.GetStyle() & ttf.STYLE_ITALIC) != 0

					var family *LoadedFontFamily
					for i := range catalog {
						if catalog[i].Name == familyName {
							family = &catalog[i]
							break
						}
					}
					if family == nil {
						catalog = append(catalog, LoadedFontFamily{
							Name: familyName})
						family = &catalog[len(catalog)-1]
					}
					if isBold {
						if isItalic {
							family.loadedFaces[BoldItalic] = LoadedFontStyle{
								faces: [NumFontSizes]*ttf.Font{ContentFont: font}, path: path}
						} else {
							family.loadedFaces[Bold] = LoadedFontStyle{
								faces: [NumFontSizes]*ttf.Font{ContentFont: font}, path: path}
						}
					} else if isItalic {
						family.loadedFaces[Italic] = LoadedFontStyle{
							faces: [NumFontSizes]*ttf.Font{ContentFont: font}, path: path}
					} else {
						family.loadedFaces[Standard] = LoadedFontStyle{
							faces: [NumFontSizes]*ttf.Font{ContentFont: font}, path: path}
					}
				}
			}
		}
		return catalog
	}
	log.Println(err)
	return nil
}

// GetSize returns the font at the given size;
// loads that size if it isn't already available.
func (style *LoadedFontStyle) GetSize(size FontSize, sizeMap [6]int32) *ttf.Font {
	ret := style.faces[size]
	if ret == nil {
		newSize, err := ttf.OpenFont(style.path, int(sizeMap[size]))
		if err != nil {
			panic(err)
		}
		style.faces[size] = newSize
		ret = newSize
	}
	return ret
}

// GetStyle returns the requested style if available.
// A fallback is returned if the requested style isn't available.
func (family *LoadedFontFamily) GetStyle(style FontStyle) *LoadedFontStyle {
	var ret *LoadedFontStyle

	for curStyle := style; curStyle >= 0; curStyle-- {
		ret = &family.loadedFaces[style]
		if ret.path != "" {
			return ret
		}
	}
	for curStyle := style + 1; curStyle < NumFontStyles; curStyle++ {
		ret = &family.loadedFaces[style]
		if ret.path != "" {
			return ret
		}
	}
	panic("loaded font " + family.Name + " has no available styles!")
}
