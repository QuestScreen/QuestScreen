package modules

import (
	"github.com/flyx/pnpscreen/api"
	"github.com/flyx/pnpscreen/base/background"
	"github.com/flyx/pnpscreen/base/herolist"
	"github.com/flyx/pnpscreen/base/persons"
	"github.com/flyx/pnpscreen/base/title"
	"github.com/flyx/pnpscreen/web"
)

// Base is a plugin providing the most common system-independent modules.
type Base struct {
}

// Name returns "Base"
func (*Base) Name() string { return "Base" }

// Modules returns a list of common modules
func (*Base) Modules() []api.Module {
	return []api.Module{
		&background.Background{},
		&herolist.HeroList{}, &persons.Persons{},
		&title.Title{}}
}

// AdditionalJS returns the JS handlers for the base modules.
func (*Base) AdditionalJS() []byte {
	return web.MustAsset("web/js/base.js")
}

// AdditionalHTML returns the HTML templates for the base modules.
func (*Base) AdditionalHTML() []byte {
	return web.MustAsset("web/html/base.html")
}

func (*Base) AdditionalCSS() []byte {
	return nil
}
