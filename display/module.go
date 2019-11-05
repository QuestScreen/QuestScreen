package display

import (
	"time"

	"github.com/flyx/pnpscreen/data"
)

// Module describes a module usable with pnpscreen
type Module interface {
	data.ConfigurableItem
	// initialize the module.
	Init(display *Display, store *data.Store) error
	// collect requests given to EndpointHandler() and initializes a transition.
	// returns the length of the transition animation.
	// TransitionStep() and Render() will be invoked continuously until the returned time has been elapsed
	// (counting starts after InitTransition() returns).
	// after that, FinishTransition() and Render() will be called.
	// if 0 is returned, TransitionStep will never be called; if a negative value is returned, neither FinishTransition()
	// nor Render() will be called.
	// this function is called in the OpenGL thread.
	InitTransition() time.Duration
	// implements transitions. use this function to collect requests given to EndpointHandler and update
	// state data based on the elapsed time accordingly. don't render anything – Render() will always be
	// called immediately after Animate().
	//
	// The given elapsed time is guaranteed to always be smaller than what was returned by InitTransition().
	// this function is called in the OpenGL thread.
	TransitionStep(elapsed time.Duration)
	// implements cleanup after transitions.
	// will be called once after each time InitTransition() returns a non-negative value
	// (but TransitionStep() and Render() may be called in between).
	FinishTransition()
	// implements rendering of the module.
	// this function is called in the OpenGL thread.
	Render()
	// to be called in the OpenGL thread after group has been changed in the web
	// server thread. Updates the display to the loaded state.
	RebuildState()
}
