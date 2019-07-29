package module

import (
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
)

type Group struct {
	Name    string
	DirName string
	System  string
}

type System struct {
	Name    string
	DirName string
}

type SharedData struct {
	Systems      []System
	Groups       []Group
	dataDir      string
	ActiveGroup  int32
	ActiveSystem int32
}

func InitSharedData() SharedData {
	usr, _ := user.Current()
	ret := SharedData{Systems: make([]System, 0, 16), Groups: make([]Group, 0, 16),
		dataDir: filepath.Join(usr.HomeDir, ".local", "share", "rpscreen"), ActiveGroup: -1,
		ActiveSystem: -1}

	systemsDir := filepath.Join(ret.dataDir, "systems")
	files, err := ioutil.ReadDir(systemsDir)
	if err == nil {
		for _, file := range files {
			if file.IsDir() {
				name, err := ioutil.ReadFile(filepath.Join(systemsDir, file.Name(), ".name"))
				if err == nil {
					ret.Systems = append(ret.Systems, System{Name: string(name), DirName: file.Name()})
				} else {
					log.Println(err)
				}
			}
		}
	} else {
		log.Println(err)
	}

	groupsDir := filepath.Join(ret.dataDir, "groups")
	files, err = ioutil.ReadDir(groupsDir)
	if err == nil {
		for _, file := range files {
			if file.IsDir() {
				name, err := ioutil.ReadFile(filepath.Join(groupsDir, file.Name(), ".name"))
				if err == nil {
					system, err := ioutil.ReadFile(filepath.Join(systemsDir, file.Name(), ".system"))
					systemName := ""
					if err == nil {
						systemName = string(system)
					}
					ret.Groups = append(ret.Groups, Group{Name: string(name), DirName: file.Name(), System: systemName})
				}
			}
		}
	} else {
		log.Println(err)
	}

	return ret
}

type Resource struct {
	Name   string
	Path   string
	Group  int32
	System int32
}

func appendDir(resources []Resource, path string, group int32, system int32) []Resource {
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

/**
 * Query the list of all files existing in the given subdirectory of the data belonging to the module.
 * If subdir is empty, files directly in the module's data are returned.
 * Never returns directories.
 */
func (data *SharedData) ListFiles(module Module, subdir string) []Resource {
	resources := make([]Resource, 0, 64)
	resources = appendDir(resources, filepath.Join(data.dataDir, "common", module.InternalName(), subdir), -1, -1)
	for index := range data.Systems {
		if data.Systems[index].DirName != "" {
			resources = appendDir(resources, filepath.Join(data.dataDir, "systems",
				data.Systems[index].DirName, module.InternalName(), subdir), -1, int32(index))
		}
	}
	for index := range data.Groups {
		if data.Groups[index].DirName != "" {
			resources = appendDir(resources, filepath.Join(data.dataDir, "groups",
				data.Groups[index].DirName, module.InternalName(), subdir), int32(index), -1)
		}
	}
	return resources
}

func (res *Resource) Enabled(data *SharedData) bool {
	return (res.Group == -1 || res.Group == data.ActiveGroup) &&
		(res.System == -1 || res.System == data.ActiveSystem)
}

func isProperFile(path string) bool {
	stat, err := os.Stat(path)
	if err == nil {
		return !stat.IsDir()
	} else {
		return false
	}
}

/**
 * Get the path to the file with the given name in the given subdir within the module's data.
 * subdir may be empty.
 * This function searches the current group's data first, then the current system's data, then the common data.
 * The first file found will be returned. If no file has been found, the empty string is returned.
 */
func (data *SharedData) GetFilePath(module Module, subdir string, filename string) string {
	if data.ActiveGroup != -1 && data.Groups[data.ActiveGroup].DirName != "" {
		path := filepath.Join(data.dataDir, "groups", data.Groups[data.ActiveGroup].DirName,
			module.InternalName(), subdir, filename)
		if isProperFile(path) {
			return path
		}
	}
	if data.ActiveSystem != -1 && data.Systems[data.ActiveSystem].DirName != "" {
		path := filepath.Join(data.dataDir, "systems", data.Groups[data.ActiveGroup].DirName,
			module.InternalName(), subdir, filename)
		if isProperFile(path) {
			return path
		}
	}
	path := filepath.Join(data.dataDir, "common", module.InternalName(), subdir, filename)
	if isProperFile(path) {
		return path
	} else {
		return ""
	}
}