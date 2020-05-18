package app

import (
	"net/http"

	"github.com/QuestScreen/api"
	"github.com/QuestScreen/api/fonts"
	"github.com/QuestScreen/api/modules"
	"github.com/QuestScreen/api/resources"
	"github.com/QuestScreen/api/server"
	"github.com/veandco/go-sdl2/ttf"
)

// ModuleIndex identifies the module internally.
type ModuleIndex int

// FirstModule is the index of the first module
const FirstModule ModuleIndex = 0

// Message is a warning or an error that should be displayed on the starting
// screen of the client.
type Message struct {
	// true if this is an error, false if it's just a warning.
	IsError bool `json:"isError"`
	// Index of the module the message is issued from, -1 if none
	ModuleIndex ModuleIndex `json:"moduleIndex"`
	// text to display
	Text string `json:"text"`
}

// App is the interface to the application for the data and display modules.
type App interface {
	DataDir(subdirs ...string) string
	NumModules() ModuleIndex
	ModuleAt(index ModuleIndex) *modules.Module
	ModulePluginIndex(index ModuleIndex) int
	NumPlugins() int
	Plugin(index int) *api.Plugin
	// ServerContext builds an api.ServerContext with the given moduleIndex.
	ServerContext(moduleIndex ModuleIndex) server.Context
	GetResources(moduleIndex ModuleIndex,
		index resources.CollectionIndex) []resources.Resource
	GetTextures() []resources.Resource
	Font(fontFamily int, style fonts.Style, size fonts.Size) *ttf.Font
	NumFontFamilies() int
	FontNames() []string
	Messages() []Message
	MessageSenderFor(index ModuleIndex) server.MessageSender
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
