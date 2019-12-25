package modules

import (
	"github.com/flyx/pnpscreen/api"
	"github.com/flyx/pnpscreen/base/background"
	"github.com/flyx/pnpscreen/base/herolist"
	"github.com/flyx/pnpscreen/base/overlays"
	"github.com/flyx/pnpscreen/base/title"
	"github.com/flyx/pnpscreen/web"
)

// Base is a plugin providing the most common system-independent modules.
var Base = api.Plugin{
	Name: "Base",
	Modules: []*api.ModuleDescriptor{
		&background.Descriptor, &herolist.Descriptor, &overlays.Descriptor,
		&title.Descriptor},
	AdditionalJS:   web.MustAsset("web/js/base.js"),
	AdditionalHTML: web.MustAsset("web/html/base.html"),
	AdditionalCSS:  nil,
}
