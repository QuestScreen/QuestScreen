package main

import (
	"fmt"

	"github.com/QuestScreen/QuestScreen/display"
	"github.com/veandco/go-sdl2/sdl"
	"gopkg.in/yaml.v3"
)

type appConfig struct {
	fullscreen bool
	width      int32
	height     int32
	port       uint16
	keyActions []display.KeyAction `yaml:"keyActions"`
}

type tmpKeyAction struct {
	Key         string
	ReturnValue int `yaml:"returnValue"`
	Description string
}

type tmpConfig struct {
	Fullscreen    bool
	Width, Height int32
	Port          uint16
	KeyActions    []tmpKeyAction
}

func (c *appConfig) MarshalYAML() (interface{}, error) {
	ret := tmpConfig{
		Fullscreen: c.fullscreen,
		Width:      c.width, Height: c.height, Port: c.port,
		KeyActions: make([]tmpKeyAction, len(c.keyActions))}
	for i := range c.keyActions {
		a := c.keyActions[i]
		ret.KeyActions[i] = tmpKeyAction{
			Key:         sdl.GetKeyName(a.Key),
			ReturnValue: a.ReturnValue,
			Description: a.Description}
	}
	return ret, nil
}

func (c *appConfig) UnmarshalYAML(value *yaml.Node) error {
	var tmp tmpConfig
	if err := value.Decode(&tmp); err != nil {
		return err
	}
	if tmp.Width == 0 || tmp.Height == 0 {
		return fmt.Errorf("invalid size (w=%d, h=%d)", tmp.Width, tmp.Height)
	}

	*c = appConfig{fullscreen: tmp.Fullscreen, width: tmp.Width, height: tmp.Height,
		port: tmp.Port, keyActions: make([]display.KeyAction, len(tmp.KeyActions))}

	for i := range tmp.KeyActions {
		ta := tmp.KeyActions[i]
		a := display.KeyAction{Key: sdl.GetKeyFromName(ta.Key),
			ReturnValue: ta.ReturnValue, Description: ta.Description}
		if a.Key == sdl.K_UNKNOWN {
			return fmt.Errorf("unknown key: %s", tmp.KeyActions[i].Key)
		}
		c.keyActions[i] = a
	}
	if len(c.keyActions) == 0 {
		c.keyActions = append(c.keyActions,
			display.KeyAction{Key: sdl.K_ESCAPE, ReturnValue: 0, Description: "Exit"})
	}
	return nil
}

func defaultConfig() appConfig {
	return appConfig{
		fullscreen: false, width: 800, height: 600, port: 8080,
		keyActions: []display.KeyAction{{Key: sdl.K_ESCAPE, ReturnValue: 0,
			Description: "Exit"}},
	}
}
