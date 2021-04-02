// +build darwin windows

package main

import (
	"log"

	"github.com/veandco/go-sdl2/sdl"
)

func setGLAttributes(debug bool) {
	sdl.GLSetAttribute(sdl.GL_CONTEXT_PROFILE_MASK,
		sdl.GL_CONTEXT_PROFILE_CORE)
	if debug {
		log.Println("using OpenGL 4.3 core profile for debugging")
		sdl.GLSetAttribute(sdl.GL_CONTEXT_MAJOR_VERSION, 4)
		sdl.GLSetAttribute(sdl.GL_CONTEXT_MINOR_VERSION, 3)
		sdl.GLSetAttribute(sdl.GL_CONTEXT_FLAGS, sdl.GL_CONTEXT_DEBUG_FLAG)
	} else {
		log.Println("using OpenGL 3.2 core profile")
		sdl.GLSetAttribute(sdl.GL_CONTEXT_MAJOR_VERSION, 3)
		sdl.GLSetAttribute(sdl.GL_CONTEXT_MINOR_VERSION, 2)
	}
}
