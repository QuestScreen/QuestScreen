// +build !darwin

package main

import (
	"log"

	"github.com/veandco/go-sdl2/sdl"
)

func setGLAttributes() {
	log.Println("using OpenGL ES 2.0 profile")
	sdl.GLSetAttribute(sdl.GL_CONTEXT_PROFILE_MASK,
		sdl.GL_CONTEXT_PROFILE_ES)
	sdl.GLSetAttribute(sdl.GL_CONTEXT_MAJOR_VERSION, 2)
	sdl.GLSetAttribute(sdl.GL_CONTEXT_MINOR_VERSION, 0)
}
