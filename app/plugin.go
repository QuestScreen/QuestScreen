package app

import (
	"encoding/json"

	"github.com/QuestScreen/api/modules"
)

// SystemTemplate is a template to create a system from.
// The template is given as YAML file because the module API is not designed to
// create standalone state objects that would be required to define such a
// template.
type SystemTemplate struct {
	// YAML content of the configuration. the contained name should be equivalent
	// to the Name field.
	Config []byte
	ID     string
	Name   string
}

// SceneTmplRef is a reference to a scene template to be used in group templates
type SceneTmplRef struct {
	// The name this scene should have in the containing group
	Name string
	// index of the plugin defining the scene template
	PluginIndex int
	// index of the template referred to in its plugin's list of templates
	TmplIndex int
}

// GroupTemplate is a template to create a group from.
// Groups are always created from templates, the default and minimal templates
// are provided by the base plugin.
type GroupTemplate struct {
	Name, Description string
	// YAML content of the configuration. The group name contained herein will
	// always be overridden when applying the template.
	Config []byte
	// list of scenes in this group.
	//
	// At least one scene index must be defined for any GroupTemplate.
	// The reference defines the name of the scene in this group.
	Scenes []SceneTmplRef
}

// SceneTemplate is a template to create a scene from.
// It is typically used for defining scenes created as part of group templates,
// but can also be used on its own to create a new scene for an existing group.
type SceneTemplate struct {
	Name, Description string
	// YAML content of the configuration. The scene name contained herein will
	// always be overridden when applying the template.
	Config []byte
}

// Plugin describes a module provider.
type Plugin struct {
	// Name contains the name of the plugin.
	Name string
	// Modules contains the list of modules provided by this plugin.
	Modules []*modules.Module
	// SystemTemplates defines templates for creating systems.
	// It should provide a template for each system ID that is referenced from a
	// template in GroupTemplates.
	SystemTemplates []SystemTemplate
	// GroupTemplates defines templates for creating groups.
	// It should only reference system IDs for which templates exist in
	// SystemTemplates.
	GroupTemplates []GroupTemplate
	// SceneTemplates defines templates for creating scenes.
	// These scenes can be referenced from GroupTemplates.
	SceneTemplates []SceneTemplate
}

type jsonTemplate struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// MarshalJSON serializes a GroupTemplate for communication to the client
func (gt *GroupTemplate) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonTemplate{Name: gt.Name, Description: gt.Description})
}

// MarshalJSON serializes a SceneTemplate for communication to the client
func (st *SceneTemplate) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonTemplate{Name: st.Name, Description: st.Description})
}

// MarshalJSON serializes a plugin for communication to the client
func (p *Plugin) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Name           string          `json:"name"`
		GroupTemplates []GroupTemplate `json:"groupTemplates"`
		SceneTemplates []SceneTemplate `json:"sceneTemplates"`
	}{Name: p.Name, GroupTemplates: p.GroupTemplates, SceneTemplates: p.SceneTemplates})
}
