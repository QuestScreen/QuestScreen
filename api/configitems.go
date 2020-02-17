package api

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"

	"github.com/veandco/go-sdl2/sdl"
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
	LoadWeb(input json.RawMessage, ctx ServerContext) SendableError
	// LoadPersisted loads the config item's state from YAML data that has been
	// read from the file system.
	//
	// LoadPersisted should be robust when loading from Persisted layout, handling
	// errors by logging them and setting appropriate default values. An error
	// returned from loading Persisted data will lead to the app to exit.
	LoadPersisted(input *yaml.Node, ctx ServerContext) error
}

// SelectableFont is a ConfigItem that allow the user to select a font family.
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
	input json.RawMessage, ctx ServerContext) SendableError {
	tmp := webSelectableFont{
		FamilyIndex: ValidatedInt{Min: 0, Max: ctx.NumFontFamilies() - 1},
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
func (sf *SelectableFont) LoadPersisted(
	input *yaml.Node, ctx ServerContext) error {
	var tmp persistedSelectableFont
	if err := input.Decode(&tmp); err != nil {
		return err
	}
	sf.Size = tmp.Size
	sf.Style = tmp.Style
	for i := 0; i < ctx.NumFontFamilies(); i++ {
		if tmp.Family == ctx.FontFamilyName(i) {
			sf.FamilyIndex = i
			return nil
		}
	}
	log.Printf("unknown font \"%s\"\n", tmp.Family)
	sf.FamilyIndex = 0
	return nil
}

// WebView returns the object itself.
func (sf *SelectableFont) WebView(ctx ServerContext) interface{} {
	return sf
}

// PersistingView returns a view that gives the family name as string.
func (sf *SelectableFont) PersistingView(ctx ServerContext) interface{} {
	return &persistedSelectableFont{
		Family: ctx.FontFamilyName(sf.FamilyIndex),
		Size:   sf.Size,
		Style:  sf.Style,
	}
}

// RGBColor represents a 24bit color in RGB color space.
type RGBColor struct {
	Red   uint8 `yaml:"r"`
	Green uint8 `yaml:"g"`
	Blue  uint8 `yaml:"b"`
}

// Use sets the color as draw color for the given renderer
func (c *RGBColor) Use(renderer *sdl.Renderer) error {
	return renderer.SetDrawColor(c.Red, c.Green, c.Blue, 255)
}

// UnmarshalJSON loads a JSON string as HTML hexcode into RGBColor
func (c *RGBColor) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if len(s) != 7 || s[0] != '#' {
		return fmt.Errorf("\"%s\" is not a valid color hexcode", s)
	}
	bytes, err := hex.DecodeString(s[1:])
	if err != nil {
		return err
	}
	c.Red = bytes[0]
	c.Green = bytes[1]
	c.Blue = bytes[2]
	return nil
}

// MarshalJSON represents the color as JSON string containing a HTML hexcode
func (c *RGBColor) MarshalJSON() ([]byte, error) {
	bytes := [3]byte{c.Red, c.Green, c.Blue}
	s := "#" + hex.EncodeToString(bytes[:])
	return json.Marshal(&s)
}

// SelectableTexturedBackground is a ConfigItem that allows the user to
// define a background color by setting a primary color and optionally,
// a secondary color together with a texture.
type SelectableTexturedBackground struct {
	Primary      RGBColor `json:"primary"`
	Secondary    RGBColor `json:"secondary"`
	TextureIndex int      `json:"textureIndex"`
}

// LoadWeb loads a background from a json input
// `{"primary": <rgb>, "secondary": <rgb>, "textureIndex": <number>}`
func (stb *SelectableTexturedBackground) LoadWeb(
	input json.RawMessage, ctx ServerContext) SendableError {
	textures := ctx.GetTextures()
	value := struct {
		Primary      RGBColor     `json:"primary"`
		Secondary    RGBColor     `json:"secondary"`
		TextureIndex ValidatedInt `json:"textureIndex"`
	}{TextureIndex: ValidatedInt{Min: -1, Max: len(textures) - 1}}
	if err := ReceiveData(input, &value); err != nil {
		return err
	}
	*stb = SelectableTexturedBackground{Primary: value.Primary,
		Secondary: value.Secondary, TextureIndex: value.TextureIndex.Value}
	return nil
}

type persistedSelectableTextureBackground struct {
	Primary, Secondary RGBColor
	Texture            string
}

// LoadPersisted loads a background from a YAML input
// `{primary: <rgb>, secondary: <rgb>, texture: <name>}`
func (stb *SelectableTexturedBackground) LoadPersisted(
	input *yaml.Node, ctx ServerContext) error {
	var value persistedSelectableTextureBackground
	if err := input.Decode(&value); err != nil {
		return err
	}
	stb.Primary = value.Primary
	stb.Secondary = value.Secondary
	stb.TextureIndex = -1
	if value.Texture != "" {
		textures := ctx.GetTextures()
		for i := range textures {
			if textures[i].Name() == value.Texture {
				stb.TextureIndex = i
				break
			}
		}
		if stb.TextureIndex == -1 {
			log.Println("Unknown texture: " + value.Texture)
		}
	}
	return nil
}

// WebView returns the object itself.
func (stb *SelectableTexturedBackground) WebView(ctx ServerContext) interface{} {
	return stb
}

// PersistingView returns a view that gives the texture name as string.
func (stb *SelectableTexturedBackground) PersistingView(ctx ServerContext) interface{} {
	ret := &persistedSelectableTextureBackground{
		Primary: stb.Primary, Secondary: stb.Secondary,
	}
	if stb.TextureIndex != -1 {
		ret.Texture = ctx.GetTextures()[stb.TextureIndex].Name()
	}
	return ret
}
