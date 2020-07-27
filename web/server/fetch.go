package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

// RequestMethod is an enum of known methods for Fetch.
type RequestMethod int

const (
	// Get is a GET request
	Get RequestMethod = iota
	// Post is a POST request
	Post
	// Put is a PUT request
	Put
	// Delete is a DELETE request
	Delete
)

func (r RequestMethod) String() string {
	switch r {
	case Get:
		return "GET"
	case Post:
		return "POST"
	case Put:
		return "PUT"
	case Delete:
		return "DELETE"
	default:
		panic("unknown request method!")
	}
}

// Fetch makes a request to the server and returns the response.
func Fetch(method RequestMethod, url string, target interface{}, payload interface{}) error {
	var body io.Reader
	if payload != nil {
		str, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(str)
	}

	req, err := http.NewRequest(method.String(), url, body)
	if err != nil {
		return err
	}
	req.Header.Add("X-Clacks-Overhead", "GNU Terry Pratchett")
	if body != nil {
		req.Header.Add("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode == 200 {
		dec := json.NewDecoder(resp.Body)
		err = dec.Decode(target)
		resp.Body.Close()
		return err
	}
	return nil
}
