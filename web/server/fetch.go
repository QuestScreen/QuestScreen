package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/QuestScreen/api/web"
)

// Fetch makes a request to the server and returns the response.
func Fetch(method web.RequestMethod, url string, payload interface{}, target interface{}) error {
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
