package app

import (
	"net/http"

	"github.com/flyx/pnpscreen/api"
)

// App is the interface to the application for the data and display modules.
type App interface {
	api.Environment
	DataDir(subdirs ...string) string
	NumModules() api.ModuleIndex
	ModuleAt(index api.ModuleIndex) api.Module
	ModulePluginIndex(index api.ModuleIndex) int
	NumPlugins() int
	Plugin(index int) *api.Plugin
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
