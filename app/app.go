package app

import "github.com/flyx/pnpscreen/api"

// App is the interface to the application for the data and display modules.
type App interface {
	api.Environment
	DataDir(subdirs ...string) string
	NumModules() api.ModuleIndex
	ModuleEnabled(index api.ModuleIndex) bool
	ModuleAt(index api.ModuleIndex) api.Module
}
