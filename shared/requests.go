package shared

import "encoding/json"

// SystemModificationRequest is sent from the client to the server to request
// the modification of a system.
type SystemModificationRequest struct {
	Name string `json:"name"`
}

// GroupCreationRequest is sent from the client to the server to request the
// creation of a group.
type GroupCreationRequest struct {
	Name               string `json:"name"`
	PluginIndex        int    `json:"pluginIndex"`
	GroupTemplateIndex int    `json:"groupTemplateIndex"`
}

// GroupModificationRequest is sent from the client to the server to request
// the modification of a group.
type GroupModificationRequest struct {
	Name        string `json:"name"`
	SystemIndex int    `json:"systemIndex"`
}

// SceneCreationRequest is sent from the client to the server to request the
// creation of a scene.
type SceneCreationRequest struct {
	Name               string `json:"name"`
	PluginIndex        int    `json:"pluginIndex"`
	SceneTemplateIndex int    `json:"sceneTemplateIndex"`
}

// SceneModificationRequest is sent from the client to the server to request the
// modification of a scene's name and modules.
type SceneModificationRequest struct {
	Name    string `json:"name"`
	Modules []bool `json:"modules"`
}

// HeroModificationRequest is sent from the client to the server to request the
// modification of a hero.
type HeroModificationRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type StateRequest struct {
	Action string `json:"action"`
	Index  int    `json:"index"`
}

type StateResponse struct {
	ActiveGroup int               `json:"activeGroup"`
	ActiveScene int               `json:"activeScene"`
	Modules     []json.RawMessage `json:"modules"`
}
