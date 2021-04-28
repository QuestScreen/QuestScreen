// +build !windows

package main

import (
	"net/url"
	"path/filepath"
)

func toFileUrl(path string) *url.URL {
	return &url.URL{Scheme: "file", Path: path}
}
