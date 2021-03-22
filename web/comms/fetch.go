package comms

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"

	api "github.com/QuestScreen/api/web"
)

// Fetch makes a request to the server and returns the response.
func Fetch(method api.RequestMethod, url string, payload interface{}, target interface{}) error {
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
	switch resp.StatusCode {
	case 200:
		if target != nil {
			dec := json.NewDecoder(resp.Body)
			err = dec.Decode(target)
		} else {
			return errors.New("got content when none was expected")
		}
		resp.Body.Close()
	case 204:
		if target != nil {
			return errors.New("got no content what some was expected")
		}
		resp.Body.Close()
	default:
		content, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		return errors.New(url + ": " + strconv.Itoa(resp.StatusCode) +
			": " + string(content))
	}
	return nil
}
