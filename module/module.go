package module

import (
	"html/template"
	"net/http"
	"net/url"
	"time"
)

type Module interface {
	// initialize the module.
	Init(common *SceneCommon) error
	// returns the name of the module.
	Name() string
	// Alphanumeric name used for:
	// * directories with module data
	// * HTTP setter endpoints
	// * IDs for menu setters
	// May not contain whitespace or special characters. Must be unique among loaded modules.
	InternalName() string
	// collect requests given to EndpointHandler() and initializes a transition.
	// returns the length of the transition animation.
	// TransitionStep() and Render() will be invoked continuously until the returned time has been elapsed
	// (counting starts after InitTransition() returns).
	// after that, FinishTransition() and Render() will be called.
	// if 0 is returned, TransitionStep will never be called; if a negative value is returned, neither FinishTransition()
	// nor Render() will be called.
	// this function is called in the OpenGL thread.
	InitTransition(common *SceneCommon) time.Duration
	// implements transitions. use this function to collect requests given to EndpointHandler and update
	// state data based on the elapsed time accordingly. don't render anything – Render() will always be
	// called immediately after Animate().
	//
	// The given elapsed time is guaranteed to always be smaller than what was returned by InitTransition().
	// this function is called in the OpenGL thread.
	TransitionStep(common *SceneCommon, elapsed time.Duration)
	// implements cleanup after transitions.
	// will be called once after each time InitTransition() returns a non-negative value
	// (but TransitionStep() and Render() may be called in between).
	FinishTransition(common *SceneCommon)
	// implements rendering of the module.
	// this function is called in the OpenGL thread.
	Render(common *SceneCommon)

	// returns partial HTML that creates the module's UI within the web interface.
	UI() template.HTML
	// implements handling for all endpoints of the module.
	// will be called for requests on all paths starting with EndpointPath.
	// returnPartial specifies whether a partial value may be returned (in case of AJAX requests).
	// on success, use WriteEndpointHeader, giving EndpointReturnRedirect if returnPartial is true.
	// on failure, write the appropriate HTTP return code.
	// this function will be called in the web server thread and may not call OpenGL functions.
	// if it returns true, InitTransition() will be called in the OpenGL thread and any OpenGL action
	// must be relayed there. return true only if calling InitTransition() will be necessary to update
	// the display.
	EndpointHandler(suffix string, values url.Values, w http.ResponseWriter, returnPartial bool) bool
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
