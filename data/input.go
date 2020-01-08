package data

import (
	"io/ioutil"
)

type inputProvider func() ([]byte, error)

func fileInput(path string) inputProvider {
	return func() ([]byte, error) {
		return ioutil.ReadFile(path)
	}
}

func byteInput(input []byte) inputProvider {
	return func() ([]byte, error) {
		return input, nil
	}
}
