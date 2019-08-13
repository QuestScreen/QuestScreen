package module

import (
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/veandco/go-sdl2/ttf"
)

// LoadedFontFace describes a font face (with a set style and size) that has
// been loaded into memory
type LoadedFontFace struct {
	font *ttf.Font
	path string
}

// FontStyle describes possible styles of a font
type FontStyle int

const (
	// Standard is the default font style
	Standard FontStyle = iota
	// Bold is the bold font style
	Bold
	// Italic is the italic font style
	Italic
	// BoldItalic is the bold and italic font style
	BoldItalic
	// NumFontStyles is not a valid FontStyle, but used for iterating.
	NumFontStyles
)

// LoadedFontSize is a font at a given size. At least one style has been loaded.
type LoadedFontSize struct {
	Size  int32
	faces [NumFontStyles]LoadedFontFace
}

// LoadedFontFamily is a family font that has been loaded. At least one size
// has been loaded.
type LoadedFontFamily struct {
	loadedSizes []LoadedFontSize
	baseIndex   int
	Name        string
}

// CreateFontCatalog loads all the fonts in the fonts directory
func CreateFontCatalog(common *SharedData, defaultSize int32) []LoadedFontFamily {
	fontPath := filepath.Join(common.DataDir, "fonts")
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

					var entry *LoadedFontSize
					for i := range catalog {
						if catalog[i].Name == familyName {
							entry = &catalog[i].loadedSizes[0]
							break
						}
					}
					if entry == nil {
						catalog = append(catalog, LoadedFontFamily{loadedSizes: make([]LoadedFontSize, 0, 16),
							Name: familyName, baseIndex: 0})
						catalog[len(catalog)-1].loadedSizes = append(catalog[len(catalog)-1].loadedSizes,
							LoadedFontSize{Size: defaultSize})
						entry = &catalog[len(catalog)-1].loadedSizes[0]
					}
					if isBold {
						if isItalic {
							entry.faces[BoldItalic] = LoadedFontFace{font: font, path: path}
						} else {
							entry.faces[Bold] = LoadedFontFace{font: font, path: path}
						}
					} else if isItalic {
						entry.faces[Italic] = LoadedFontFace{font: font, path: path}
					} else {
						entry.faces[Standard] = LoadedFontFace{font: font, path: path}
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
func (family *LoadedFontFamily) GetSize(size int32) *LoadedFontSize {
	var newLoaded *LoadedFontSize
	for i := range family.loadedSizes {
		if family.loadedSizes[i].Size == size {
			return &family.loadedSizes[i]
		} else if family.loadedSizes[i].Size > size {
			family.loadedSizes = append(family.loadedSizes, LoadedFontSize{})
			copy(family.loadedSizes[i+1:], family.loadedSizes[i:])
			if family.baseIndex >= i {
				family.baseIndex++
			}
			newLoaded = &family.loadedSizes[i]
		}
	}
	if newLoaded == nil {
		family.loadedSizes = append(family.loadedSizes, LoadedFontSize{})
		newLoaded = &family.loadedSizes[len(family.loadedSizes)-1]
	}

	*newLoaded = LoadedFontSize{Size: size}
	base := &family.loadedSizes[family.baseIndex]
	for i := Standard; i < NumFontStyles; i++ {
		if base.faces[i].font != nil {
			var err error
			if newLoaded.faces[i].font, err = ttf.OpenFont(base.faces[i].path, int(size)); err != nil {
				log.Println(err)
			}
		}
	}
	return newLoaded
}

// GetFace returns the font face with the requested style;
// loads that style if it hasn't already been loaded.
// Returns a fallback if the requested style isn't available.
func (fs *LoadedFontSize) GetFace(style FontStyle) *ttf.Font {
	var curStyle FontStyle
	for curStyle = style; curStyle >= 0; curStyle-- {
		if fs.faces[curStyle].font != nil {
			return fs.faces[curStyle].font
		}
	}
	for curStyle = style + 1; curStyle < NumFontStyles; curStyle++ {
		if fs.faces[curStyle].font != nil {
			return fs.faces[curStyle].font
		}
	}
	panic("loaded font size contains no faces!")
}
