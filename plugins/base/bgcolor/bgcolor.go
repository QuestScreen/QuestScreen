package bgcolor

import (
	"encoding/json"
	"log"

	"github.com/QuestScreen/api"
	"github.com/QuestScreen/api/comms"
	"github.com/QuestScreen/api/server"
	"gopkg.in/yaml.v3"
)

// Config is a config.Item that allows the user to
// define a background color by setting a primary color and optionally,
// a secondary color together with a texture.
type Config struct {
	api.Background
}

// NewConfig creates a new Background having the given primary color
// and no texture. This can be used for the default config of a module, since
// it requires *Background (and RGBA.AsBackground does provide Background).
func NewConfig(value api.Background) *Config {
	return &Config{Background: value}
}

// LoadWeb loads a background from a json input
// `{"primary": <rgb>, "secondary": <rgb>, "textureIndex": <number>}`
func (b *Config) LoadWeb(
	input json.RawMessage, ctx server.Context) error {
	textures := ctx.GetTextures()
	value := struct {
		Primary      api.RGBA           `json:"primary"`
		Secondary    api.RGBA           `json:"secondary"`
		TextureIndex comms.ValidatedInt `json:"textureIndex"`
	}{TextureIndex: comms.ValidatedInt{Min: -1, Max: len(textures) - 1}}
	if err := comms.ReceiveData(input, &value); err != nil {
		return err
	}
	*b = Config{Background: api.Background{Primary: value.Primary,
		Secondary: value.Secondary, TextureIndex: value.TextureIndex.Value}}
	return nil
}

type persistedBackground struct {
	Primary, Secondary api.RGBA
	Texture            string
}

// LoadPersisted loads a background from a YAML input
// `{primary: <rgb>, secondary: <rgb>, texture: <name>}`
func (b *Config) LoadPersisted(
	input *yaml.Node, ctx server.Context) error {
	var value persistedBackground
	if err := input.Decode(&value); err != nil {
		return err
	}
	b.Primary = value.Primary
	b.Secondary = value.Secondary
	b.TextureIndex = -1
	if value.Texture != "" {
		textures := ctx.GetTextures()
		for i := range textures {
			if textures[i].Name() == value.Texture {
				b.TextureIndex = i
				break
			}
		}
		if b.TextureIndex == -1 {
			log.Println("Unknown texture: " + value.Texture)
		}
	}
	return nil
}

// WebView returns the object itself.
func (b *Config) WebView(ctx server.Context) interface{} {
	return b.Background
}

// PersistingView returns a view that gives the texture name as string.
func (b *Config) PersistingView(ctx server.Context) interface{} {
	ret := &persistedBackground{
		Primary: b.Primary, Secondary: b.Secondary,
	}
	if b.TextureIndex != -1 {
		ret.Texture = ctx.GetTextures()[b.TextureIndex].Name()
	}
	return ret
}
