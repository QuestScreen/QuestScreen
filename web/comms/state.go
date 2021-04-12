package comms

import (
	"strconv"
	"strings"

	"github.com/QuestScreen/QuestScreen/shared"
	"github.com/QuestScreen/QuestScreen/web"
	"github.com/QuestScreen/api/groups"
	"github.com/QuestScreen/api/resources"
	api "github.com/QuestScreen/api/web"
)

// ServerState implements web.Server.
type ServerState struct {
	*shared.State
	// <plugin-id>/<module-id>
	Base string
}

type groupWrapper struct {
	*shared.Group
}

func (gw groupWrapper) Heroes() groups.HeroList {
	return herolist{gw.Group.Heroes}
}

type herolist struct {
	data []shared.Hero
}

func (hl herolist) NumHeroes() int {
	return len(hl.data)
}

func (hl herolist) Hero(index int) groups.Hero {
	return heroWrapper{&hl.data[index]}
}

type heroWrapper struct {
	*shared.Hero
}

func (hw heroWrapper) ID() string {
	return hw.Hero.ID
}

func (hw heroWrapper) Name() string {
	return hw.Hero.Name
}

func (hw heroWrapper) Description() string {
	return hw.Hero.Description
}

// GetResources implements resources.Provider.
func (s *ServerState) GetResources(index resources.CollectionIndex) []resources.Resource {
	var urlBuilder strings.Builder
	urlBuilder.WriteString("/resources/")
	urlBuilder.WriteString(s.Base)
	urlBuilder.WriteByte('/')
	urlBuilder.WriteString(strconv.Itoa(int(index)))
	var ret []resources.Resource
	if err := Fetch(api.Get, urlBuilder.String(), nil, &ret); err != nil {
		panic(err)
	}
	return ret
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
	return groupWrapper{&web.Data.Groups[s.State.ActiveGroup]}
}

// Fetch implements web.ServerState (see there for description)
func (s *ServerState) Fetch(method api.RequestMethod, subpath string,
	payload interface{}, target interface{}) {
	var urlBuilder strings.Builder
	urlBuilder.WriteString("/state/")
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
