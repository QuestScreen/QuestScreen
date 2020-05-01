package main

import (
	"github.com/veandco/go-sdl2/sdl"
	"gopkg.in/yaml.v3"
)

type keyAction struct {
	key         sdl.Keycode
	returnValue int
}

type config struct {
	fullscreen    bool
	width, height int32
	port          int
	keyActions    []keyAction
}

func (c *config) UnmarshalYAML(value yaml.Node) error {
	var tmp struct {
		Fullscreen    bool
		Width, Height int32
		Port          int
		KeyActions    []struct {
			Key         string
			ReturnValue int
		}
	}
	if err := value.Decode(&tmp); err != nil {
		return err
	}

	*c = config{fullscreen: tmp.Fullscreen, width: tmp.Width, height: tmp.Height,
		port: tmp.Port, keyActions: make([]keyAction, len(tmp.KeyActions))}

}
