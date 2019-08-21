package module

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/flyx/rpscreen/data"
)

// SharedData contains configuration and state of the whole application.
type SharedData struct {
	data.Config
	Fonts                  []LoadedFontFamily
	DefaultBorderWidth     int32
	DefaultHeadingTextSize int32
	DefaultBodyTextSize    int32
	ActiveGroup            int
	ActiveSystem           int
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

// Init initializes the Sharedsd, including loading the configuration files.
func (sd *SharedData) Init(modules data.ConfigurableItemProvider, width int32, height int32) {
	sd.Config.Init(modules)
	sd.Fonts = CreateFontCatalog(sd, sd.DefaultBodyTextSize)
	sd.DefaultBorderWidth = height / 133
	sd.DefaultHeadingTextSize = height / 13
	sd.DefaultBodyTextSize = height / 27
	sd.ActiveGroup = -1
	sd.ActiveSystem = -1
}

// ListFiles queries the list of all files existing in the given subdirectory of
// the data belonging to the module. If subdir is empty, files directly in the
// module's data are returned. Never returns directories.
func (sd *SharedData) ListFiles(module Module, subdir string) []Resource {
	resources := make([]Resource, 0, 64)
	resources = appendDir(resources, filepath.Join(sd.Config.DataDir, "base", module.InternalName(), subdir), -1, -1)
	for i := 0; i < sd.Config.NumSystems(); i++ {
		if sd.Config.SystemDirectory(i) != "" {
			resources = appendDir(resources, filepath.Join(sd.Config.DataDir, "systems",
				sd.Config.SystemDirectory(i), module.InternalName(), subdir), -1, i)
		}
	}
	for i := 0; i < sd.Config.NumGroups(); i++ {
		if sd.Config.GroupDirectory(i) != "" {
			resources = appendDir(resources, filepath.Join(sd.DataDir, "groups",
				sd.Config.GroupDirectory(i), module.InternalName(), subdir), i, -1)
		}
	}
	return resources
}

// Enabled checks whether a resource is currently enabled based on the group
// and system selection in sd.
func (res *Resource) Enabled(sd *SharedData) bool {
	return (res.Group == -1 || res.Group == sd.ActiveGroup) &&
		(res.System == -1 || res.System == sd.ActiveSystem)
}

func isProperFile(path string) bool {
	stat, err := os.Stat(path)
	if err == nil {
		return !stat.IsDir()
	}
	return false
}

// GetFilePath tries to find a file that may exist multiple times.
// It searches in the current group's sd first, then in the current system's
// sd, then in the common sd. The first file found will be returned.
// If no file has been found, the empty string is returned.
func (sd *SharedData) GetFilePath(module Module, subdir string, filename string) string {
	if sd.ActiveGroup != -1 && sd.Config.GroupDirectory(sd.ActiveGroup) != "" {
		path := filepath.Join(sd.DataDir, "groups", sd.Config.GroupDirectory(sd.ActiveGroup),
			module.InternalName(), subdir, filename)
		if isProperFile(path) {
			return path
		}
	}
	if sd.ActiveSystem != -1 && sd.Config.SystemDirectory(sd.ActiveSystem) != "" {
		path := filepath.Join(sd.DataDir, "systems", sd.Config.SystemDirectory(sd.ActiveSystem),
			module.InternalName(), subdir, filename)
		if isProperFile(path) {
			return path
		}
	}
	path := filepath.Join(sd.DataDir, "base", module.InternalName(), subdir, filename)
	if isProperFile(path) {
		return path
	}
	return ""
}

type jsonItem struct {
	Name, DirName string
}

func (sd *SharedData) jsonSystems() []jsonItem {
	ret := make([]jsonItem, 0, sd.NumSystems())
	for i := 0; i < sd.NumSystems(); i++ {
		ret = append(ret, jsonItem{Name: sd.SystemName(i),
			DirName: sd.SystemDirectory(i)})
	}
	return ret
}

func (sd *SharedData) jsonGroups() []jsonItem {
	ret := make([]jsonItem, 0, sd.NumGroups())
	for i := 0; i < sd.NumGroups(); i++ {
		ret = append(ret, jsonItem{Name: sd.GroupName(i),
			DirName: sd.GroupDirectory(i)})
	}
	return ret
}

func (sd *SharedData) jsonFonts() []string {
	ret := make([]string, 0, len(sd.Fonts))
	for i := 0; i < len(sd.Fonts); i++ {
		ret = append(ret, sd.Fonts[i].Name)
	}
	return ret
}

type jsonData struct {
	Systems      []jsonItem
	Groups       []jsonItem
	Fonts        []string
	ActiveGroup  int
	ActiveSystem int
}

// SendJSON sends a JSON describing all systems groups and fonts.
func (sd *SharedData) SendJSON(w http.ResponseWriter, curGroup int, curSystem int) {
	data.SendAsJSON(w, jsonData{
		Systems: sd.jsonSystems(), Groups: sd.jsonGroups(),
		Fonts:       sd.jsonFonts(),
		ActiveGroup: curGroup, ActiveSystem: curSystem})
}
