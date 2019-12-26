package api

import (
	"reflect"
	"time"

	"github.com/veandco/go-sdl2/sdl"
	"gopkg.in/yaml.v3"
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
// All funcs are expected to be called in the server thread.
// Data should be passed to the Module via mutex or channel.
type ModuleState interface {
	SerializableItem
	// HandleAction updates the state. if no error is returned, the OpenGL
	// thread will call InitTransition on the linked Module.
	//
	// Returns the content to return on success (nil if nothing should be
	// returned). If not nil, the content will be encoded into JSON.
	HandleAction(index int, payload []byte) (interface{}, error)
	// SendToModule() sends the current complete state to the Module (which will
	// collect it by RebuildState).
	SendToModule()
}

// ResourceSelector defines a subdirectory and a filename suffix list.
// Those will be used to collect resource files for this module in the
// data directories.
type ResourceSelector struct {
	// may be empty, in which case resource files are searched directly
	// in the module directories.
	Subdirectory string
	// filters files by suffix. If empty or nil, no filter will be applied
	// (note however that files starting with a dot will always be filtered out).
	Suffixes []string
}

// ModuleDescriptor describes a module.
type ModuleDescriptor struct {
	// Name is the human-readable name of the module.
	Name string
	// ID is a unique string, used for identifying the module inside
	// HTTP URLs and in the filesystem. Therefore, the ID is restricted to ASCII
	// letters, digits, and the symbols `.,-_`
	// Note that internally, a module is identified by the ModuleIndex given
	// to CreateModule.
	ID string
	// ResourceCollections lists selectors for resource collections of this
	// module. The maximum ResourceCollectionIndex available to this module is
	// len(ResourceCollections()) - 1.
	ResourceCollections []ResourceSelector
	// ConfigType returns the type of the module's configuration.

	ConfigType reflect.Type
	// Actions() is the list of actions callable via HTTP POST that should be
	// registered for this module.
	Actions []string
	// DefaultConfig returns a configuration object with default values.
	// This must be a pointer to a struct in which each field is a pointer to an
	// item implementing ConfigItem.
	// reflect.TypeOf(DefaultConfig()) must equal ConfigType().
	// All fields of the struct object must hold a non-nil value.
	DefaultConfig interface{}
	// CreateModule creates the module object. This will only be called once
	// during app initialization, making the module a singleton object.
	//
	// The index argument should be retained by the module for identifying itself
	// to the environment.
	CreateModule func(renderer *sdl.Renderer, env StaticEnvironment,
		index ModuleIndex) (Module, error)
}

// RenderContext is the context given to all rendering funcs of a module
type RenderContext struct {
	Env      Environment
	Renderer *sdl.Renderer
}

// Module describes a module object. This object belongs with the OpenGL thread.
// All funcs are called in the OpenGL thread unless noted otherwise.
type Module interface {
	// Descriptor shall return the ModuleDescriptor that has created this module.
	Descriptor() *ModuleDescriptor
	// SetConfig sets the module's calculated configuration. The given config
	// object will be of the type specified by ModuleDescriptor.ConfigType().
	// The config object will have all fields set to non-nil values.
	// The rendering thread will always call RebuildState after SetConfig.
	SetConfig(config interface{})
	// InitTransition will be called after the current State has been modified via
	// HandleAction. It should retrieve the data sent by the State in a
	// thread-safe manner.
	//
	// The return value is the duration of the transition initiated by this call.
	// For that duration, the render thread will continuously call
	// TransitionStep() and Render(). After the time has passed,
	// FinishTransition() and Render() will be called to render the final state.
	//
	// if 0 is returned, TransitionStep() will never be called; if a negative
	// value is returned, neither FinishTransition() nor Render() will be
	// called.
	InitTransition(ctx RenderContext) time.Duration
	// TransitionStep should update the renderer's current state while
	// transitioning. A call to TransitionStep() will always immediately be
	// followed by a call to Render().
	//
	// The given elapsed time is guaranteed to always be smaller than what was
	// returned by InitTransition().
	TransitionStep(ctx RenderContext, elapsed time.Duration)
	// FinishTransition() is for cleanup after a transition and for preparing the
	// final state. It will be called exactly once for each call to
	// InitTransition() that returned a non-negative value.
	//
	// A call to FinishTransition() will always immediately be followed by a call
	// to Render().
	FinishTransition(ctx RenderContext)
	// Render renders the Module.s current state.
	Render(ctx RenderContext)
	// RebuildState will be called after any action that requires rebuilding the
	// Module's state, such as a scene, config or group change.
	//
	// A call to RebuildState() will always immediately be followed by a call to
	// Render() and may be preceded by a call to SendToModule on a ModuleState
	// linked to this module.
	RebuildState(ctx RenderContext)
	// CreateState will be called in the server thread. It shall create a
	// ModuleState that is linked to this module (i.e. actions on the state shall
	// send data to the module that can be retrieved via InitTransition).
	//
	// The exact way of communication between ModuleState and Module is up to the
	// implementation, but it must be thread-safe since the ModuleState's funcs
	// will be called in the server thread while the module's funcs will be called
	// in the OpenGL thread. It is adviced to use a mutex-protected object owned
	// by the Module and known by the state to transfer data.
	//
	// input will reflect the Persisted layout of the serialized state as
	// generated by the state's SerializableView method.
	// It may be nil in which case the state will be created with default values.
	CreateState(input *yaml.Node, env Environment) (ModuleState, error)
}
