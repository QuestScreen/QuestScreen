package data

import (
	"io/ioutil"
	"log"
	"os/user"
	"path/filepath"
)

// StaticData contains data that doesn't change after it has been loaded
type StaticData struct {
	DataDir                string
	Fonts                  []LoadedFontFamily
	DefaultBorderWidth     int32
	DefaultHeadingTextSize int32
	DefaultBodyTextSize    int32
}

// Init initializes the static data
func (s *StaticData) Init(width int32, height int32) {
	usr, _ := user.Current()
	s.DataDir = filepath.Join(usr.HomeDir, ".local", "share", "rpscreen")
	s.DefaultBorderWidth = height / 133
	s.DefaultHeadingTextSize = height / 13
	s.DefaultBodyTextSize = height / 27
	s.Fonts = CreateFontCatalog(s.DataDir, s.DefaultBodyTextSize)
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
			if !file.IsDir() {
				resources = append(resources, Resource{Name: file.Name(), Path: filepath.Join(path, file.Name()),
					Group: group, System: system})
			}
		}
	} else {
		log.Println(err)
	}
	return resources
}
