package module

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/flyx/rpscreen/data"
)

// SharedData contains configuration and state of the whole application.
type SharedData struct {
	data.Config
	ActiveGroup  int
	ActiveSystem int
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

// Init initializes the SharedData, including loading the configuration files.
func (data *SharedData) Init(modules data.ConfigurableItemProvider) {
	data.Config.Init(modules)
	data.ActiveGroup = -1
	data.ActiveSystem = -1
}

// ListFiles queries the list of all files existing in the given subdirectory of
// the data belonging to the module. If subdir is empty, files directly in the
// module's data are returned. Never returns directories.
func (data *SharedData) ListFiles(module Module, subdir string) []Resource {
	resources := make([]Resource, 0, 64)
	resources = appendDir(resources, filepath.Join(data.Config.DataDir, "base", module.InternalName(), subdir), -1, -1)
	for i := 0; i < data.Config.NumSystems(); i++ {
		if data.Config.SystemDirectory(i) != "" {
			resources = appendDir(resources, filepath.Join(data.Config.DataDir, "systems",
				data.Config.SystemDirectory(i), module.InternalName(), subdir), -1, i)
		}
	}
	for i := 0; i < data.Config.NumGroups(); i++ {
		if data.Config.GroupDirectory(i) != "" {
			resources = appendDir(resources, filepath.Join(data.DataDir, "groups",
				data.Config.GroupDirectory(i), module.InternalName(), subdir), i, -1)
		}
	}
	return resources
}

// Enabled checks whether a resource is currently enabled based on the group
// and system selection in data.
func (res *Resource) Enabled(data *SharedData) bool {
	return (res.Group == -1 || res.Group == data.ActiveGroup) &&
		(res.System == -1 || res.System == data.ActiveSystem)
}

func isProperFile(path string) bool {
	stat, err := os.Stat(path)
	if err == nil {
		return !stat.IsDir()
	}
	return false
}

// GetFilePath tries to find a file that may exist multiple times.
// It searches in the current group's data first, then in the current system's
// data, then in the common data. The first file found will be returned.
// If no file has been found, the empty string is returned.
func (data *SharedData) GetFilePath(module Module, subdir string, filename string) string {
	if data.ActiveGroup != -1 && data.Config.GroupDirectory(data.ActiveGroup) != "" {
		path := filepath.Join(data.DataDir, "groups", data.Config.GroupDirectory(data.ActiveGroup),
			module.InternalName(), subdir, filename)
		if isProperFile(path) {
			return path
		}
	}
	if data.ActiveSystem != -1 && data.Config.SystemDirectory(data.ActiveSystem) != "" {
		path := filepath.Join(data.DataDir, "systems", data.Config.SystemDirectory(data.ActiveSystem),
			module.InternalName(), subdir, filename)
		if isProperFile(path) {
			return path
		}
	}
	path := filepath.Join(data.DataDir, "base", module.InternalName(), subdir, filename)
	if isProperFile(path) {
		return path
	}
	return ""
}
