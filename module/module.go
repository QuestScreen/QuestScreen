package module

import (
	"html/template"
	"net/http"
)

type Module interface {
	// initialize the module.
	Init(common *SceneCommon) error
	// update module data based on user input occurred via EndpointHandler.
	// this function is called in the OpenGL thread and may use OpenGL functions.
	ProcessUpdate(common *SceneCommon)
	// implements rendering of the module.
	// this function is called in the OpenGL thread.
	Render(common *SceneCommon)
	// returns the name of the module.
	Name() string

	// returns partial HTML that creates the module's UI within the web interface.
	UI() template.HTML
	// returns the base path for the module's endpoints.
	// must start with a '/'.
	EndpointPath() string
	// implements handling for all endpoints of the module.
	// will be called for requests on all paths starting with EndpointPath.
	// returnPartial specifies whether a partial value may be returned (in case of AJAX requests).
	// on success, use WriteEndpointHeader, giving EndpointReturnRedirect if returnPartial is true.
	// on failure, write the appropriate HTTP return code.
	// this function will be called in the web server thread and may not call OpenGL functions.
	// if it returns true, ProcessUpdate will be called in the OpenGL thread and any OpenGL action
	// must be relayed there. return true only if calling ProcessUpdate will be necessary to update
	// the display.
	EndpointHandler(suffix string, value string, w http.ResponseWriter, returnPartial bool) bool
}

type EndpointReturn int

const (
	EndpointReturnEmpty EndpointReturn = iota
	EndpointReturnPartial
	EndpointReturnRedirect
)

func WriteEndpointHeader(w http.ResponseWriter, returns EndpointReturn) {
	switch returns {
	case EndpointReturnPartial:
		w.Header().Set("Content-Type", "text/xml")
		w.WriteHeader(http.StatusOK)
	case EndpointReturnEmpty:
		w.WriteHeader(http.StatusNoContent)
	case EndpointReturnRedirect:
		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusSeeOther)
	}
}
