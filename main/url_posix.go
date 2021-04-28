// +build !windows

package main

import (
	"net/url"
)

func toFileUrl(path string) *url.URL {
	return &url.URL{Scheme: "file", Path: path}
}
