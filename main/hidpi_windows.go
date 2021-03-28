// +build windows

package main

// #cgo LDFLAGS: -lSDL2
// #include <hidpi_windows.h>
import "C"

func init() {
	C.enable_hidpi()
}
