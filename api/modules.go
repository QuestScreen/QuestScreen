package api

import (
	"time"

	"github.com/veandco/go-sdl2/sdl"
)

// the interface declared in this file are to be implemented by
// a pnpscreen module.

// ResourceCollectionIndex indexes all resource collections of a module.
type ResourceCollectionIndex int

// ModuleIndex identifies the module internally.
type ModuleIndex int

// ModuleState describes the state of a module. It is written to and loaded
// from a group's state.yaml.
//
// A type implementing this typically holds a reference to its module.
// All funcs are expected to be called in the server thread.
// Data should be passed to the module's Renderer via mutex or channel.
type ModuleState interface {
	LoadFrom(yamlSubtree interface{}, env Environment) error

	ToYAML(env Environment) interface{}
	ToJSON() interface{}

	// defines the list of actions that should be registered for this module.
	Actions() []string
	// HandleAction updates the state. if no error is returned, the OpenGL
	// thread will call InitTransition on the module's Renderer.
	//
	// returns the content to return on success (nil if nothing should be returned).
	// if not nil, the content must be valid JSON.
	HandleAction(index int, payload []byte) ([]byte, error)
}

// Persistence describes a module's persisted data (i.e. data that will be
// written to the filesystem).
// It holds the module's current state and supplies generators for generating
// module configuration.
//
// A module's configuration is not stored in Persistence since it is split
// among three layers: base config, system config, group config.
// Those layers are held by the global Persistence, which also generates a
// calculated config from them which is then sent to the Renderer.
//
// config and state objects are expected to be a pointer.
type Persistence interface {
	// returns an empty configuration
	// The item defines the type of its configuration.
	EmptyConfig() interface{}
	// returns a configuration object with default values.
	DefaultConfig() interface{}
	// GetState retrieves the current state of the item.
	State() ModuleState
}

// Renderer describes an item that renders to an SDL surface.
// All funcs of the renderer are expected to be called in the
// OpenGL thread.
type Renderer interface {
	// SetConfig sets the module's calculated configuration.
	// This must be done in the OpenGL thread; it belongs to the Renderer.
	// The display will always call RebuildState after SetConfig.
	SetConfig(config interface{})
	// InitTransition queries update requests created by the State's
	// HandleAction. Returns the length of the transition's animation.
	//
	// TransitionStep() and Render() will be invoked continuously until the
	// returned time has been elapsed. after that, FinishTransition() and
	// Render() will be called.
	// if 0 is returned, TransitionStep will never be called; if a negative
	// value is returned, neither FinishTransition() nor Render() will be
	// called.
	InitTransition(renderer *sdl.Renderer) time.Duration
	// updates the renderer's current state while transitioning.
	// A call to TransitionStep() will always immediately be followed by a call
	// to Render().
	//
	// The given elapsed time is guaranteed to always be smaller than what was
	// returned by InitTransition().
	TransitionStep(renderer *sdl.Renderer, elapsed time.Duration)
	// cleanup after transition.
	//
	// will be called exactly once after each time InitTransition() returns a
	// non-negative value (but TransitionStep() and Render() may be called in
	// between).
	FinishTransition(renderer *sdl.Renderer)
	// Renders the module's output.
	Render(renderer *sdl.Renderer)
	// will be called after group has been changed in the web server thread.
	// Will always be preceded by a call to SetConfig.
	// Updates the renderer's state according to the current config.
	RebuildState(renderer *sdl.Renderer)
}

// ResourceSelector defines a subdirectory and a filename suffix list.
// Those will be used to collect resource files for this modules in the
// data directory.
type ResourceSelector struct {
	// may be empty, in which case resource files are searched directly
	// in the module directories.
	Subdirectory string
	// filters files by suffix. If empty or nil, no filter will be applied
	// (not however that files starting with a dot will always be filtered out).
	Suffixes []string
}

// Module describes a module usable with PnP Screen.
// A module consists of a Renderer and a Persistence.
//
// The Renderer part belongs with the OpenGL thread, while the Persistence
// part belongs with the Server thread.
// It is the implementor's responsibility to make any communication between the
// two components thread-safe.
type Module interface {
	Persistence
	Renderer

	Name() string
	// Unique ID string, used for identifying the module via HTTP and
	// in the filesystem and URL, therefore restricted to ASCII letters digits,
	// and the symbols `.,-_`
	// Note that internally, the module is identified by the ModuleIndex given
	// to its Init() func.
	ID() string
	// describes selectors for resource collections of this module.
	// the maximum ResourceCollectionIndex available to this module is
	// len(ResourceCollections()) - 1.
	ResourceCollections() []ResourceSelector
	// initializes the module. This must be called before any funcs of
	// Persistence or Renderer are called; however Name(), Id() and
	// ResourceCollections() must work on an uninitialized module.
	//
	// index should be retained by the module for identifying itself to the
	// environment.
	// This func is called during startup before multithreading begins and
	// therefore does not need to be threadsafe.
	//
	// you may not yet call GetResources on env since those will be loaded
	// after module initialization.
	Init(renderer *sdl.Renderer, env Environment, index ModuleIndex) error
}
