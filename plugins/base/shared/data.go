package shared

// BackgroundState is the background module's state as shared with the web app.
type BackgroundState struct {
	CurIndex int      `json:"curIndex"`
	Items    []string `json:"items"`
}

// HerolistState is the herolist module's state as shared with the web app.
type HerolistState struct {
	Global bool   `json:"global"`
	Heroes []bool `json:"heroes"`
}

// OverlayItem is an item of the overlay.
type OverlayItem struct {
	Name     string `json:"name"`
	Selected bool   `json:"selected"`
}

// OverlayState is the state of the overlay module.
type OverlayState []OverlayItem
