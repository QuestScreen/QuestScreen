package shared

// GroupCreationRequest is sent from the client to the server to request the
// creation of a group.
type GroupCreationRequest struct {
	Name               string `json:"name"`
	PluginIndex        int    `json:"pluginIndex"`
	GroupTemplateIndex int    `json:"groupTemplateIndex"`
}

// SystemModificationRequest is sent from the client to the server to request
// the modification of a system.
type SystemModificationRequest struct {
	Name string `json:"name"`
}

// GroupModificationRequest is sent from the client to the server to request
// the modification of a group.
type GroupModificationRequest struct {
	Name string `json:"name"`
}
