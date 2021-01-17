package shared

// GroupCreationRequest is sent from the client to the server to request the
// creation of a group.
type GroupCreationRequest struct {
	Name               string `json:"name"`
	PluginIndex        int    `json:"pluginIndex"`
	GroupTemplateIndex int    `json:"groupTemplateIndex"`
}
