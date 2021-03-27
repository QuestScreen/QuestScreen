// +build darwin windows

package main

import (
	"log"

	"github.com/veandco/go-sdl2/sdl"
)

func setGLAttributes() {
	log.Println("using OpenGL 3.2 core profile")

	sdl.GLSetAttribute(sdl.GL_CONTEXT_PROFILE_MASK,
		sdl.GL_CONTEXT_PROFILE_CORE)
	sdl.GLSetAttribute(sdl.GL_CONTEXT_MAJOR_VERSION, 3)
	sdl.GLSetAttribute(sdl.GL_CONTEXT_MINOR_VERSION, 2)
}
