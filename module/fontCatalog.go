package module

import (
	"github.com/veandco/go-sdl2/ttf"
	"io/ioutil"
	"log"
	"path/filepath"
)

type LoadedFont struct {
	Font *ttf.Font
	Name string
}

func CreateFontCatalog(common *SharedData, defaultSize int) []LoadedFont {
	fontPath := filepath.Join(common.dataDir, "fonts")
	files, err := ioutil.ReadDir(fontPath)

	if err == nil {
		catalog := make([]LoadedFont, 0, len(files))
		for _, file := range files {
			if !file.IsDir() {
				if font, err := ttf.OpenFont(fontPath+"/"+file.Name(), defaultSize); err != nil {
					log.Println(err)
				} else {
					familyName := font.FaceFamilyName()
					if familyName == "" {
						familyName = file.Name()
					}
					catalog = append(catalog, LoadedFont{Font: font, Name: familyName})
				}
			}
		}
		return catalog
	} else {
		log.Println(err)
		return nil
	}
}
