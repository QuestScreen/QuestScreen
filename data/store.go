package data

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
)

// Store keeps all config & state data of the application
type Store struct {
	StaticData
	Config
	ActiveGroup  int
	ActiveSystem int
}

// Init initializes the SharedData, including loading the configuration files.
func (s *Store) Init(modules ConfigurableItemProvider, width int32, height int32) {
	s.StaticData.Init(width, height, modules)
	s.Config.Init(&s.StaticData)
	s.ActiveGroup = -1
	s.ActiveSystem = -1
}

// ListFiles queries the list of all files existing in the given subdirectory of
// the data belonging to the module. If subdir is empty, files directly in the
// module's data are returned. Never returns directories.
func (s *Store) ListFiles(item ConfigurableItem, subdir string) []Resource {
	resources := make([]Resource, 0, 64)
	resources = appendDir(resources, filepath.Join(s.DataDir, "base", item.InternalName(), subdir), -1, -1)
	for i := 0; i < s.Config.NumSystems(); i++ {
		if s.Config.SystemDirectory(i) != "" {
			resources = appendDir(resources, filepath.Join(s.DataDir, "systems",
				s.Config.SystemDirectory(i), item.InternalName(), subdir), -1, i)
		}
	}
	for i := 0; i < s.Config.NumGroups(); i++ {
		if s.Config.GroupDirectory(i) != "" {
			resources = appendDir(resources, filepath.Join(s.DataDir, "groups",
				s.Config.GroupDirectory(i), item.InternalName(), subdir), i, -1)
		}
	}
	return resources
}

// Enabled checks whether a resource is currently enabled based on the group
// and system selection in sd.
func (res *Resource) Enabled(store *Store) bool {
	return (res.Group == -1 || res.Group == store.ActiveGroup) &&
		(res.System == -1 || res.System == store.ActiveSystem)
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
func (s *Store) GetFilePath(item ConfigurableItem, subdir string, filename string) string {
	if s.ActiveGroup != -1 && s.Config.GroupDirectory(s.ActiveGroup) != "" {
		path := filepath.Join(s.DataDir, "groups", s.Config.GroupDirectory(s.ActiveGroup),
			item.InternalName(), subdir, filename)
		if isProperFile(path) {
			return path
		}
	}
	if s.ActiveSystem != -1 && s.Config.SystemDirectory(s.ActiveSystem) != "" {
		path := filepath.Join(s.DataDir, "systems", s.Config.SystemDirectory(s.ActiveSystem),
			item.InternalName(), subdir, filename)
		if isProperFile(path) {
			return path
		}
	}
	path := filepath.Join(s.DataDir, "base", item.InternalName(), subdir, filename)
	if isProperFile(path) {
		return path
	}
	return ""
}

type jsonItem struct {
	Name    string `json:"name"`
	DirName string `json:"dirName"`
}

type jsonModuleSetting struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type jsonModuleDesc struct {
	Name   string              `json:"name"`
	Config []jsonModuleSetting `json:"config"`
}

func (s *Store) jsonSystems() []jsonItem {
	ret := make([]jsonItem, 0, s.NumSystems())
	for i := 0; i < s.NumSystems(); i++ {
		ret = append(ret, jsonItem{Name: s.SystemName(i),
			DirName: s.SystemDirectory(i)})
	}
	return ret
}

func (s *Store) jsonGroups() []jsonItem {
	ret := make([]jsonItem, 0, s.NumGroups())
	for i := 0; i < s.NumGroups(); i++ {
		ret = append(ret, jsonItem{Name: s.GroupName(i),
			DirName: s.GroupDirectory(i)})
	}
	return ret
}

func (s *Store) jsonFonts() []string {
	ret := make([]string, 0, len(s.Fonts))
	for i := 0; i < len(s.Fonts); i++ {
		ret = append(ret, s.Fonts[i].Name)
	}
	return ret
}

func (s *Store) jsonModules() []jsonModuleDesc {
	ret := make([]jsonModuleDesc, 0, s.items.NumItems())
	for i := 0; i < s.items.NumItems(); i++ {
		item := s.items.ItemAt(i)
		itemConfig := s.baseConfigs[i]
		itemValue := reflect.ValueOf(itemConfig).Elem()
		for ; itemValue.Kind() == reflect.Interface ||
			itemValue.Kind() == reflect.Ptr; itemValue = itemValue.Elem() {
		}
		cur := jsonModuleDesc{
			Name:   item.Name(),
			Config: make([]jsonModuleSetting, 0, itemValue.NumField())}
		for j := 0; j < itemValue.NumField(); j++ {
			cur.Config = append(cur.Config, jsonModuleSetting{
				Name: itemValue.Type().Field(j).Name,
				Type: itemValue.Type().Field(j).Type.Elem().Name()})
		}
		ret = append(ret, cur)
	}
	return ret
}

type jsonGlobal struct {
	Systems      []jsonItem       `json:"systems"`
	Groups       []jsonItem       `json:"groups"`
	Fonts        []string         `json:"fonts"`
	Modules      []jsonModuleDesc `json:"modules"`
	ActiveGroup  int              `json:"activeGroup"`
	ActiveSystem int              `json:"activeSystem"`
}

// SendGlobalJSON sends a JSON describing all systems, groups, fonts and modules.
func (s *Store) SendGlobalJSON(w http.ResponseWriter) {
	SendAsJSON(w, jsonGlobal{
		Systems: s.jsonSystems(), Groups: s.jsonGroups(),
		Fonts:       s.jsonFonts(),
		Modules:     s.jsonModules(),
		ActiveGroup: s.ActiveGroup, ActiveSystem: s.ActiveSystem})
}

// SendBaseJSON writes the base config as JSON to w
func (s *Store) SendBaseJSON(w http.ResponseWriter) {
	SendAsJSON(w, s.buildModuleConfigJSON(s.baseConfigs))
}

// ReceiveBaseJSON parses the given config as JSON, updates the internal
// config and writes it to the base/config.yaml file
func (s *Store) ReceiveBaseJSON(w http.ResponseWriter, reader io.Reader) {
	raw, _ := ioutil.ReadAll(reader)
	if err := s.loadJSONModuleConfigs(raw, s.baseConfigs); err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else {
		s.writeYamlBaseConfig(s.baseConfigs)
	}
}

// SendSystemJSON writes the config of the given system to w
func (s *Store) SendSystemJSON(w http.ResponseWriter, system string) {
	for i := range s.systems {
		if s.systems[i].DirName == system {
			SendAsJSON(w, s.buildModuleConfigJSON(s.systems[i].Modules))
			return
		}
	}
	http.Error(w, "404: unknown system \""+system+"\"", http.StatusNotFound)
}

// ReceiveSystemJSON parses the given config as JSON, updates the internal
// config and writes it to the system's config.yaml file
// returns true iff the config has been successfully parsed.
func (s *Store) ReceiveSystemJSON(w http.ResponseWriter, system string,
	reader io.Reader) bool {
	for i := range s.systems {
		if s.systems[i].DirName == system {
			raw, _ := ioutil.ReadAll(reader)
			if err := s.loadJSONModuleConfigs(raw, s.systems[i].Modules); err != nil {
				log.Println(err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return false
			}
			s.writeYamlSystemConfig(s.systems[i])
			return true
		}
	}
	http.Error(w, "404: unknown system \""+system+"\"", http.StatusNotFound)
	return false
}

// SendGroupJSON writes the config of the given group to w
func (s *Store) SendGroupJSON(w http.ResponseWriter, group string) {
	for i := range s.groups {
		if s.groups[i].Config.DirName == group {
			SendAsJSON(w, s.buildModuleConfigJSON(s.groups[i].Config.Modules))
			return
		}
	}
	http.Error(w, "404: unknown group \""+group+"\"", http.StatusNotFound)
}

// ReceiveGroupJSON parses the given config as JSON, updates the internal
// config and writes it to the group's config.yaml file.
// returns true iff the config has been successfully parsed.
func (s *Store) ReceiveGroupJSON(w http.ResponseWriter, group string,
	reader io.Reader) bool {
	for i := range s.groups {
		if s.groups[i].Config.DirName == group {
			raw, _ := ioutil.ReadAll(reader)
			if err := s.loadJSONModuleConfigs(raw, s.groups[i].Config.Modules); err != nil {
				log.Println(err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return false
			}
			s.writeYamlGroupConfig(s.groups[i].Config, s.systems)
			return true
		}
	}
	http.Error(w, "404: unknown group \""+group+"\"", http.StatusNotFound)
	return false
}
