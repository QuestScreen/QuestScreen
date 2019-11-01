package data

import (
	"io/ioutil"
	"log"
	"os/user"
	"path/filepath"

	"github.com/veandco/go-sdl2/ttf"
)

// StaticData contains data that doesn't change after it has been loaded
type StaticData struct {
	DataDir            string
	Fonts              []LoadedFontFamily
	DefaultBorderWidth int32
	DefaultTextSizes   [6]int32
	items              ConfigurableItemProvider
}

// Init initializes the static data
func (s *StaticData) Init(width int32, height int32, items ConfigurableItemProvider) {
	usr, _ := user.Current()
	s.DataDir = filepath.Join(usr.HomeDir, ".local", "share", "rpscreen")
	s.DefaultBorderWidth = height / 133
	s.DefaultTextSizes = [6]int32{height / 37, height / 27, height / 19,
		height / 13, height / 8, height / 4}
	s.Fonts = CreateFontCatalog(s.DataDir, s.DefaultTextSizes[ContentFont])
	s.items = items
}

// A Resource is a file in the file system.
type Resource struct {
	Name   string
	Path   string
	Group  int
	System int
}

func appendDir(resources []Resource, path string, group int, system int) []Resource {
	files, err := ioutil.ReadDir(path)
	if err == nil {
		for _, file := range files {
			if !file.IsDir() && file.Name()[0] != '.' {
				resources = append(resources, Resource{Name: file.Name(), Path: filepath.Join(path, file.Name()),
					Group: group, System: system})
			}
		}
	} else {
		log.Println(err)
	}
	return resources
}

// GetFontFace returns the font face of the selected font.
func (s *StaticData) GetFontFace(selected *SelectableFont) *ttf.Font {
	return s.Fonts[selected.FamilyIndex].GetStyle(selected.Style).GetSize(selected.Size, s.DefaultTextSizes)
}

// ResourceNames generates a list of resource names from a list of resources.
func ResourceNames(resources []Resource) []string {
	ret := make([]string, len(resources))
	for i := range resources {
		ret[i] = resources[i].Name
	}
	return ret
}
