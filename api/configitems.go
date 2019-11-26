package api

import (
	"errors"
	"fmt"
)

// this file contains types that may be used as fields in a module's
// configuration struct.

// ConfigItem describes an item in a module's configuration.
// A ConfigItem's public fields will be loaded from YAML structure automatically
// via reflection, and JSON serialization will also be done via reflection.
// you may use the tags `json:` and `yaml:` on those fields as documented in
// the json and yaml.v3 packages.
type ConfigItem interface {
	// PostLoad consolidates data after loading from JSON or YAML.
	PostLoad(env Environment, fromYAML bool) error
}

// SelectableFont is used to allow the user to select a font family.
type SelectableFont struct {
	Family      string    `json:"-" yaml:"family"`
	FamilyIndex int       `yaml:"-" json:"familyIndex"`
	Size        FontSize  `json:"size" yaml:"size"`
	Style       FontStyle `json:"style" yaml:"style"`
}

// PostLoad sets FamilyIndex from Family (YAML) or the other way round (JSON)
func (sf *SelectableFont) PostLoad(env Environment, fromYAML bool) error {
	fonts := env.FontCatalog()
	if fromYAML {
		for i := range fonts {
			if sf.Family == fonts[i].Name() {
				sf.FamilyIndex = i
				return nil
			}
		}
		return fmt.Errorf("unknown font \"%s\"", sf.Family)
	}
	if sf.FamilyIndex < 0 || sf.FamilyIndex >= len(fonts) {
		return errors.New("font index out of range")
	}
	sf.Family = fonts[sf.FamilyIndex].Name()
	return nil
}
