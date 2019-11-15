package api

// this file contains types that may be used as fields in a module's
// configuration struct.

// SelectableFont is used to allow the user to select a font family.
type SelectableFont struct {
	Family      string    `json:"-" yaml:"family"`
	FamilyIndex int       `yaml:"-" json:"familyIndex"`
	Size        FontSize  `json:"size" yaml:"size"`
	Style       FontStyle `json:"style" yaml:"style"`
}
