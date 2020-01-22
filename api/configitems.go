package api

import (
	"encoding/json"
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
	// LoadWeb loads the config item's state from JSON data that as been
	// sent from the web client.
	//
	// Any structural and value error should result in returning a SendableError
	// (typically a *BadRequest) and should not alter the item's state.
	// Implementation should typically use the ReceiveData func, possibly together
	// with the strict ValidatedX types provided by the api package.
	LoadWeb(input json.RawMessage, env Environment) SendableError
	// LoadPersisted loads the config item's state from YAML data that has been
	// read from the file system.
	//
	// LoadPersisted should be robust when loading from Persisted layout, handling
	// errors by logging them and setting appropriate default values. An error
	// returned from loading Persisted data will lead to the app to exit.
	LoadPersisted(input *yaml.Node, env Environment) error
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
	FamilyIndex ValidatedInt `json:"familyIndex"`
	Size        ValidatedInt `json:"size"`
	Style       ValidatedInt `json:"style"`
}

// LoadWeb loads a selectable font from a json input
// `{"familyIndex": <number>, "size": <number>, "style": <number>}`
func (sf *SelectableFont) LoadWeb(
	input json.RawMessage, env Environment) SendableError {
	fonts := env.FontCatalog()
	tmp := webSelectableFont{
		FamilyIndex: ValidatedInt{Min: 0, Max: len(fonts) - 1},
		Size:        ValidatedInt{Min: 0, Max: int(HugeFont)},
		Style:       ValidatedInt{Min: 0, Max: int(BoldItalic)},
	}
	if err := ReceiveData(input, &tmp); err != nil {
		return err
	}
	*sf = SelectableFont{FamilyIndex: tmp.FamilyIndex.Value,
		Size:  FontSize(tmp.Size.Value),
		Style: FontStyle(tmp.Style.Value)}
	return nil
}

// LoadPersisted loads a selectable font from a YAML input
// `{family: <string>, size: <number>, style: <number>}`
func (sf *SelectableFont) LoadPersisted(input *yaml.Node, env Environment) error {
	fonts := env.FontCatalog()
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

// WebView returns the object itself.
func (sf *SelectableFont) WebView(env Environment) interface{} {
	return sf
}

// PersistingView returns a view that gives the family name as string.
func (sf *SelectableFont) PersistingView(env Environment) interface{} {
	return &persistedSelectableFont{
		Family: env.FontCatalog()[sf.FamilyIndex].Name(),
		Size:   sf.Size,
		Style:  sf.Style,
	}
}
