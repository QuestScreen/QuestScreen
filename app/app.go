package app

import (
	"net/http"
	"reflect"

	"github.com/QuestScreen/QuestScreen/shared"
	"github.com/QuestScreen/api"
	"github.com/QuestScreen/api/fonts"
	"github.com/QuestScreen/api/modules"
	"github.com/QuestScreen/api/resources"
	"github.com/QuestScreen/api/server"
	"github.com/veandco/go-sdl2/ttf"
)

// App is the interface to the application for the data and display modules.
type App interface {
	DataDir(subdirs ...string) string
	NumPlugins() int
	Plugin(index int) *api.Plugin
	PluginID(index int) string
	NumModules() shared.ModuleIndex
	ModuleAt(index shared.ModuleIndex) *modules.Module
	ModulePluginIndex(index shared.ModuleIndex) int
	NumConfigItems() shared.ConfigItemIndex
	ConfigItemName(index shared.ConfigItemIndex) string
	ConfigItemFromType(t reflect.Type) shared.ConfigItemIndex
	ConfigItemPluginIndex(index shared.ConfigItemIndex) int
	// ServerContext builds an api.ServerContext with the given moduleIndex.
	ServerContext(moduleIndex shared.ModuleIndex) server.Context
	GetResources(moduleIndex shared.ModuleIndex,
		index resources.CollectionIndex) []resources.Resource
	GetTextures() []resources.Resource
	Font(fontFamily int, style fonts.Style, size fonts.Size) *ttf.Font
	NumFontFamilies() int
	FontNames() []string
	Messages() []shared.Message
	MessageSenderFor(index shared.ModuleIndex) server.MessageSender
}

// TooManyRequests is an error that is issued if the server receives more data
// than the rendering thread can process.
type TooManyRequests struct {
}

// Error returns "Too many requests"
func (TooManyRequests) Error() string {
	return "Too many requests"
}

// StatusCode returns 503
func (TooManyRequests) StatusCode() int {
	return http.StatusTooManyRequests
}
