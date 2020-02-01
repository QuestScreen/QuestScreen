package api

import (
	"time"

	"github.com/veandco/go-sdl2/sdl"
	"gopkg.in/yaml.v3"
)

// the interface declared in this file are to be implemented by
// a pnpscreen module.

// ResourceCollectionIndex indexes all resource collections of a module.
type ResourceCollectionIndex int

// ModulePureEndpoint is an endpoint of a module for the HTTP server.
// It takes PUT requests on the path specified by the ModuleDescriptor.
type ModulePureEndpoint interface {
	// Put handles a PUT request. Two return values are expected:
	//
	// The first return value will be serialized to JSON and sent back to the
	// client. If it's nil, nothing will be sent and the client will get a
	// 204 No Content.
	//
	// The second return value will be handed over to the module's InitTransition
	// which will be called in the OpenGL thread after Put has returned.
	// For thread safety, that value should be constructed from scratch and not be
	// a pointer into the ModuleState object.
	//
	// If an error is returned, InitTransition will not be called and both return
	// values will be ignored. The server will the respond according to the cause
	// of the returned error.
	Put(payload []byte) (interface{}, interface{}, SendableError)
}

// ModuleIDEndpoint is an endpoint of a module for the HTTP server.
// It takes PUT requests on the path specified by the ModuleDescriptor, with an
// additional URL path item interpreted as ID.
type ModuleIDEndpoint interface {
	// Put works analoguous to ModulePureEndpoint.Put, but gets the id from the
	// request URL path as additional parameter.
	Put(id string, payload []byte) (interface{}, interface{}, SendableError)
}

// ModuleState describes the state of a module. It is written to and loaded
// from a group's state.yaml.
//
// All funcs are expected to be called in the server thread.
type ModuleState interface {
	SerializableItem
	// CreateModuleData generates a data object that contains all required data
	// for the module to rebuild its state. The returned data object will be
	// handed over to the module's RebuildState. For thread safety, it should not
	// be a pointer into the ModuleState object.
	CreateModuleData() interface{}
}

// HeroChangeAction is an enum describing a change in the list of heroes
type HeroChangeAction int

const (
	// HeroAdded describes the action of adding a hero to the list of heroes
	HeroAdded HeroChangeAction = iota
	// HeroModified describes the action of modifying a hero's data
	HeroModified
	// HeroDeleted describes the action of deleting a hero from the list of heroes
	HeroDeleted
)

// HeroAwareModuleState is an interface that must be implemented by module
// states if they work with heroes. It lets the application send messages to the
// state when the list of heroes changes.
type HeroAwareModuleState interface {
	HeroListChanged(heroes HeroList, action HeroChangeAction, heroIndex int)
}

// PureEndpointProvider is a ModuleState extension for modules whose
// ModuleDescriptor defines one or more pure endpoints in its EndpointPaths.
type PureEndpointProvider interface {
	// PureEndpoint returns the pure endpoint defined at the given index of the
	// ModuleDecriptor's EndpointPaths slice. This should be a cheap getter as it
	// will be called for every request on one of the module's pure endpoints.
	PureEndpoint(index int) ModulePureEndpoint
}

// IDEndpointProvider is a ModuleState extension for modules whose
// ModuleDescriptor defines one or more id endpoints in its EndpointPaths.
type IDEndpointProvider interface {
	// IDEndpoint returns the id endpoint defined at the given index of the
	// ModuleDescriptor's EndpointPaths slice. This should be a cheap getter as it
	// will be called for every request on one of the module's id endpoints.
	IDEndpoint(index int) ModuleIDEndpoint
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
	ID string
	// ResourceCollections lists selectors for resource collections of this
	// module. The maximum ResourceCollectionIndex available to this module is
	// len(ResourceCollections()) - 1.
	ResourceCollections []ResourceSelector
	// EndpointPaths defines a list of API endpoints for the client to change this
	// module's state and trigger animations.
	//
	// The endpoints will be queryable at
	//
	//     /state/<module-id>/<endpoint-path>[/<entity-id>]
	//
	// If a path ends with a `/`, it will take the additional <entity-id>
	// parameter. At most one path may be empty, in which cause it will be
	// queryable at
	//
	//     /state/<module-id>
	//
	// At most one path may be `"/"`, in which case it will be queryable at
	//
	//     /state/<module-id>/<entity-id>
	//
	// If the `"/"` path exists, the only other path that may exist is the empty
	// path.
	//
	// If at least one path not ending with `/` exists, the module's state must
	// implement PureEndpointProvider, and if at least one path ending with `/`
	// exists, the module's state must implement IDEndpointProvider.
	EndpointPaths []string
	// DefaultConfig is a configuration object with default values.
	//
	// This value defines the type of this module's configuration. Its type must
	// be a pointer to a struct in which each field is a pointer to an item
	// implementing ConfigItem.
	//
	// Generally, a value of this type may have any of its fields set to nil,
	// meaning that it should inherit the value from a previous level. This is
	// for scene, group, system and base config (in that order). However, the
	// default config must only have non-nil values since it defines the fallback
	// if the whole path up from scene config to base config does not define any
	// value for a certain item.
	DefaultConfig interface{}
	// CreateModule creates the module object. This will only be called once
	// during app initialization, making the module a singleton object.
	//
	// CreateModule should only initialize the bare minimum of the module's data;
	// RebuildState will be issued to the module before the first Render() call.
	CreateModule func(renderer *sdl.Renderer) (Module, error)
	// CreateState will be called in the server thread. It shall create a
	// ModuleState for the module created by CreateModule.
	//
	// input will reflect the Persisted layout of the serialized state as
	// generated by the state's SerializableView method.
	// It may be nil in which case the state will be created with default values.
	//
	// Communication between ModuleState and Module will be done via the state's
	// HandleAction and CreateModuleData methods which create data, and the
	// module's InitTransition and RebuildState methods which consume that data.
	//
	// If the module accesses a group's heroes, its state must additionally
	// implement HeroAwareModuleState.
	CreateState func(input *yaml.Node, ctx ServerContext) (ModuleState, error)
}

// Module describes a module object. This object belongs with the OpenGL thread.
// All funcs are called in the OpenGL thread unless noted otherwise.
type Module interface {
	// Descriptor shall return the ModuleDescriptor that has created this module.
	Descriptor() *ModuleDescriptor
	// SetConfig sets the module's calculated configuration. The given config
	// object will be of the type of the value ModuleDescriptor.DefaultConfig.
	// The config object will have all fields set to non-nil values.
	// The rendering thread will always call RebuildState after SetConfig.
	SetConfig(config interface{})
	// InitTransition will be called after the current ModuleState has been
	// modified via HandleAction.
	// data contains the data generated by HandleAction.
	//
	// The return value is the duration of the transition initiated by this call.
	// For that duration, the render thread will continuously call
	// TransitionStep and Render. After the time has passed,
	// FinishTransition and Render will be called to render the final state.
	//
	// if 0 is returned, TransitionStep will never be called; if a negative
	// value is returned, neither FinishTransition nor Render will be
	// called.
	InitTransition(ctx RenderContext, data interface{}) time.Duration
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
	// Render renders the Module's current state.
	Render(ctx RenderContext)
	// RebuildState will be called after any action that requires rebuilding the
	// Module's state, such as a scene, config or group change. For scene and
	// group change, data contains data generated by the ModuleState's
	// CreateModuleData; it will be nil for config changes.
	//
	// A call to RebuildState will always immediately be followed by a call to
	// Render.
	RebuildState(ctx ExtendedRenderContext, data interface{})
}
