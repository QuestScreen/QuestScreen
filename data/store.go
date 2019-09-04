package data

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"reflect"

	"gopkg.in/yaml.v3"
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
	s.StaticData.Init(width, height)
	s.Config.Init(&s.StaticData, modules)
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
	Name, DirName string
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

type jsonData struct {
	Systems      []jsonItem
	Groups       []jsonItem
	Fonts        []string
	ActiveGroup  int
	ActiveSystem int
}

// SendJSON sends a JSON describing all systems groups and fonts.
func (s *Store) SendJSON(w http.ResponseWriter) {
	SendAsJSON(w, jsonData{
		Systems: s.jsonSystems(), Groups: s.jsonGroups(),
		Fonts:       s.jsonFonts(),
		ActiveGroup: s.ActiveGroup, ActiveSystem: s.ActiveSystem})
}

func (s *Store) sendModuleConfigJSON(
	w http.ResponseWriter, config map[string]interface{}) {
	ret := make(map[string]jsonModuleConfig)
	for i := 0; i < s.items.NumItems(); i++ {
		item := s.items.ItemAt(i)
		itemConfig := config[item.Name()]
		itemValue := reflect.ValueOf(itemConfig).Elem()
		for ; itemValue.Kind() == reflect.Interface ||
			itemValue.Kind() == reflect.Ptr; itemValue = itemValue.Elem() {
		}
		if itemValue.NumField() > 0 {
			curConfig := item.GetConfig()
			jsonConfig := make(jsonModuleConfig)
			curValue := reflect.ValueOf(curConfig).Elem()
			for i := 0; i < itemValue.NumField(); i++ {
				jsonConfig[itemValue.Type().Field(i).Name] = jsonConfigItem{
					Type:    itemValue.Type().Field(i).Type.Elem().Name(),
					Value:   itemValue.Field(i).Interface(),
					Default: curValue.Field(i).Interface()}
			}
			ret[item.Name()] = jsonConfig
		}
	}
	SendAsJSON(w, ret)
}

func (s *StaticData) loadModuleConfigInto(target interface{},
	fromYaml bool,
	values map[string]interface{}, moduleName string, w http.ResponseWriter) bool {
	targetModule := reflect.ValueOf(target).Elem()
	targetModuleType := targetModule.Type()
	for i := 0; i < targetModuleType.NumField(); i++ {
		inValue, ok := values[targetModuleType.Field(i).Name]
		if !ok {
			http.Error(w, "value missing for "+moduleName+"."+
				targetModuleType.Field(i).Name, http.StatusBadRequest)
			return false
		}
		inModuleConfig, ok := inValue.(map[string]interface{})
		if !ok {
			raw, ok := inValue.(map[interface{}]interface{})
			if !ok {
				http.Error(w, "value of "+moduleName+"."+
					targetModuleType.Field(i).Name+" is not a JSON object",
					http.StatusBadRequest)
				return false
			}
			inModuleConfig = make(map[string]interface{})
			for key, value := range raw {
				stringKey, ok := key.(string)
				if !ok {
					http.Error(w, "value of"+moduleName+"."+
						targetModuleType.Field(i).Name+" contains non-string key",
						http.StatusBadRequest)
					return false
				}
				inModuleConfig[stringKey] = value
			}
		}
		wasNil := false
		if targetModule.Field(i).IsNil() {
			targetModule.Field(i).Set(reflect.New(targetModuleType.Field(i).Type.Elem()))
			wasNil = true
		}
		targetSetting := targetModule.Field(i).Interface()

		if !s.setModuleConfigFieldFrom(targetSetting, fromYaml, inModuleConfig, w) {
			if wasNil {
				targetModule.Field(i).Set(reflect.Zero(targetModuleType.Field(i).Type))
			}
			return false
		}
	}
	return true
}

func (s *Store) receiveModuleConfigJSON(
	w http.ResponseWriter, config map[string]interface{}, reader io.Reader) bool {
	raw, _ := ioutil.ReadAll(reader)
	var res map[string]interface{}
	if err := json.Unmarshal(raw, &res); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return false
	}
	for key, value := range res {
		valueMap, ok := value.(map[string]interface{})
		if !ok {
			http.Error(w, "data of module \""+key+"\" is not a JSON object!",
				http.StatusBadRequest)
			return false
		}
		conf, ok := config[key]
		if !ok {
			http.Error(w, "unknown module: \""+key+"\"", http.StatusBadRequest)
			return false
		}
		if !s.loadModuleConfigInto(conf, false, valueMap, key, w) {
			return false
		}
	}
	return true
}

// SendBaseJSON writes the base config as JSON to w
func (s *Store) SendBaseJSON(w http.ResponseWriter) {
	s.sendModuleConfigJSON(w, s.baseConfigs)
}

// ReceiveBaseJSON parses the given config as JSON, updates the internal
// config and writes it to the base/config.yaml file
func (s *Store) ReceiveBaseJSON(w http.ResponseWriter, reader io.Reader) {
	if s.receiveModuleConfigJSON(w, s.baseConfigs, reader) {
		path := filepath.Join(s.DataDir, "base", "config.yaml")
		raw, _ := yaml.Marshal(s.baseConfigs)
		ioutil.WriteFile(path, raw, 0644)
	}
}

// SendSystemJSON writes the config of the given system to w
func (s *Store) SendSystemJSON(w http.ResponseWriter, system string) {
	for i := range s.systems {
		if s.systems[i].DirName == system {
			s.sendModuleConfigJSON(w, s.systems[i].Modules)
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
			if s.receiveModuleConfigJSON(w, s.systems[i].Modules, reader) {
				path := filepath.Join(s.DataDir, "systems", s.systems[i].DirName, "config.yaml")
				raw, _ := yaml.Marshal(s.systems[i])
				ioutil.WriteFile(path, raw, 0644)
				return true
			}
			return false
		}
	}
	http.Error(w, "404: unknown system \""+system+"\"", http.StatusNotFound)
	return false
}

// SendGroupJSON writes the config of the given group to w
func (s *Store) SendGroupJSON(w http.ResponseWriter, group string) {
	for i := range s.groups {
		if s.groups[i].Config.DirName == group {
			s.sendModuleConfigJSON(w, s.groups[i].Config.Modules)
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
			if s.receiveModuleConfigJSON(w, s.groups[i].Config.Modules, reader) {
				path := filepath.Join(s.DataDir, "groups", s.groups[i].Config.DirName, "config.yaml")
				raw, _ := yaml.Marshal(s.systems[i])
				ioutil.WriteFile(path, raw, 0644)
				return true
			}
			return false
		}
	}
	http.Error(w, "404: unknown group \""+group+"\"", http.StatusNotFound)
	return false
}
