package comms

import (
	"strings"

	"github.com/QuestScreen/QuestScreen/web"
	"github.com/QuestScreen/api/groups"
	"github.com/QuestScreen/api/resources"
	api "github.com/QuestScreen/api/web"
)

// ServerState implements web.Server.
type ServerState struct {
	Base string
}

// GetResources implements resources.Provider.
func (s *ServerState) GetResources(index resources.CollectionIndex) []resources.Resource {
	panic("GetResources not implemented")
}

// GetTextures implements resources.Provider.
func (s *ServerState) GetTextures() []resources.Resource {
	return web.StaticData.Textures
}

// NumFontFamilies implements web.Server.
func (s *ServerState) NumFontFamilies() int {
	return len(web.StaticData.Fonts)
}

// FontFamilyName implements web.Server.
func (s *ServerState) FontFamilyName(index int) string {
	return web.StaticData.Fonts[index]
}

// ActiveGroup implements web.Server.
func (s *ServerState) ActiveGroup() groups.Group {
	// TODO
	return nil
}

// Fetch implements web.ServerState (see there for description)
func (s *ServerState) Fetch(method api.RequestMethod, subpath string,
	payload interface{}, target interface{}) {
	var urlBuilder strings.Builder
	urlBuilder.WriteString(s.Base)
	if subpath != "" {
		if subpath[0] == '/' {
			panic("Fetch @ " + s.Base + ": subpath '" + subpath + "' begins with a slash")
		}
		urlBuilder.WriteByte('/')
		urlBuilder.WriteString(subpath)
	}
	if err := Fetch(method, urlBuilder.String(), payload, target); err != nil {
		panic(err)
	}
}
