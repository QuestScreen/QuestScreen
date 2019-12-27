package api

import (
	"fmt"
	"log"

	"gopkg.in/yaml.v3"
)

// this file contains types that may be used as fields in a module's
// configuration struct.

// ConfigItem describes an item in a module's configuration.
// A ConfigItem's public fields will be loaded from YAML structure automatically
// via reflection, and JSON serialization will also be done via reflection.
// you may use the tags `json:` and `yaml:` on those fields as documented in
// the json and yaml.v3 packages.
type ConfigItem interface {
	SerializableItem
	// LoadFrom loads the config item's state from the given input.
	//
	// input is given as YAML node since it might only be a subtree of the
	// complete data. This type is also used for JSON data since YAML is a
	// superset of JSON.
	//
	// LoadFrom should be robust when loading from Persisted layout, handling
	// errors by logging them and setting appropriate default values. An error
	// returned from loading Persisted data will lead to the app to exit.
	// For Web layout, LoadFrom should be strict, returning any error. Those
	// errors will cause the server to respond with a HTTP 400 status code.
	LoadFrom(input *yaml.Node, env Environment, layout DataLayout) error
}

// SelectableFont is used to allow the user to select a font family.
type SelectableFont struct {
	FamilyIndex int       `json:"familyIndex"`
	Size        FontSize  `json:"size"`
	Style       FontStyle `json:"style"`
}

type persistedSelectableFont struct {
	Family string    `yaml:"family"`
	Size   FontSize  `yaml:"size"`
	Style  FontStyle `yaml:"style"`
}

type webSelectableFont struct {
	FamilyIndex int `yaml:"familyIndex"`
	Size        int
	Style       int
}

// LoadFrom loads values from a JSON/YAML subtree
func (sf *SelectableFont) LoadFrom(input *yaml.Node, env Environment,
	layout DataLayout) error {
	fonts := env.FontCatalog()
	if layout == Persisted {
		var tmp persistedSelectableFont
		if err := input.Decode(&tmp); err != nil {
			return err
		}
		sf.Size = tmp.Size
		sf.Style = tmp.Style
		for i := range fonts {
			if tmp.Family == fonts[i].Name() {
				sf.FamilyIndex = i
				return nil
			}
		}
		log.Printf("unknown font \"%s\"\n", tmp.Family)
		sf.FamilyIndex = 0
		return nil
	}

	var tmp webSelectableFont
	if err := input.Decode(&tmp); err != nil {
		return err
	}
	if tmp.FamilyIndex < 0 || tmp.FamilyIndex >= len(fonts) {
		return fmt.Errorf("font index out of range: %d", tmp.FamilyIndex)
	}
	if tmp.Size < 0 || tmp.Size > int(HugeFont) {
		return fmt.Errorf("font size out of range: %d", tmp.Size)
	}
	if tmp.Style < 0 || tmp.Style > int(BoldItalic) {
		return fmt.Errorf("font style out of range: %d", tmp.Style)
	}
	*sf = SelectableFont{FamilyIndex: tmp.FamilyIndex, Size: FontSize(tmp.Size),
		Style: FontStyle(tmp.Style)}
	return nil
}

// SerializableView returns the object itself for Web, or an object with the
// family name instead of its index for Persisted
func (sf *SelectableFont) SerializableView(
	env Environment, layout DataLayout) interface{} {
	if layout == Persisted {
		return &persistedSelectableFont{
			Family: env.FontCatalog()[sf.FamilyIndex].Name(),
			Size:   sf.Size,
			Style:  sf.Style,
		}
	}
	return sf
}
