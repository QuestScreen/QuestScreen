package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/flyx/pnpscreen/api"
)

type httpMethods int

const (
	httpGet httpMethods = 1 << iota
	httpPost
	httpPut
	httpDelete
	httpUnknown
)

func parseMethod(raw string) httpMethods {
	switch raw {
	case "GET":
		return httpGet
	case "POST":
		return httpPost
	case "PUT":
		return httpPut
	case "DELETE":
		return httpDelete
	default:
		return httpUnknown
	}
}

// String returns the method name for a single method and "[ <name1>... ]" for
// multiple methods.
func (m httpMethods) String() string {
	switch m {
	case httpGet:
		return "GET"
	case httpPost:
		return "POST"
	case httpPut:
		return "PUT"
	case httpDelete:
		return "DELETE"
	case httpUnknown:
		return "UNKNOWN"
	default:
		var sb strings.Builder
		sb.WriteByte('[')
		for c := httpMethods(1); c < httpUnknown; c = c << 1 {
			if c&m == c {
				sb.WriteByte(' ')
				sb.WriteString(c.String())
			}
		}
		sb.WriteByte(']')
		return sb.String()
	}
}

type pathItem interface {
	walk(url *string, ids *[]string) bool
}

type pathFragment string

func (pf pathFragment) walk(url *string, ids *[]string) bool {
	if len(pf) > len(*url) || (*url)[:len(pf)] != string(pf) {
		return false
	}
	if len(*url) == len(pf) {
		*url = ""
	} else {
		if (*url)[len(pf)] != '/' {
			return false
		}
		*url = (*url)[len(pf)+1:]
	}
	return true
}

type idCapture struct{}

func (idc idCapture) walk(url *string, ids *[]string) bool {
	pos := strings.Index(*url, "/")
	if pos == -1 {
		*ids = append(*ids, *url)
		*url = ""
	} else {
		*ids = append(*ids, (*url)[:pos])
		*url = (*url)[pos+1:]
	}
	return true
}

type endpointHandler interface {
	Handle(method httpMethods, ids []string, payload []byte) (interface{},
		api.SendableError)
}

type endpoint struct {
	allowedMethods httpMethods
	handler        endpointHandler
}

func (endpoint) walk(url *string, id *[]string) bool {
	return true
}

// handler is a HTTP handler that is able to capture URL path fragments as IDs
// (i.e. /config/groups/<group-id>/scenes/<scene-id>) and then call the
// registered endpointHandler with the captured IDs as strings. It also filters
// requests by HTTP method and only calls an endpoint handler when the actual
// HTTP method is specified for that endpoint.
type handler struct {
	name     string
	basePath string
	path     []pathItem
}

func sendJSON(w http.ResponseWriter, data interface{}) {
	content, err := json.Marshal(data)
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	} else {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Clacks-Overhead", "GNU Terry Pratchett")
	method := parseMethod(r.Method)

	var ids []string
	var e endpoint
	if h.basePath[len(h.basePath)-1] == '/' {
		url := r.URL.Path[len(h.basePath):]
		for i := range h.path {
			ok := true
			if url == "" {
				e, ok = h.path[i].(endpoint)
				if ok {
					break
				}
			} else {
				ok = h.path[i].walk(&url, &ids)
			}
			if !ok {
				http.Error(w, fmt.Sprintf("[404] %s: not found", h.name),
					http.StatusNotFound)
				return
			}
		}
	} else {
		e = h.path[0].(endpoint)
	}

	if method&e.allowedMethods == 0 {
		http.Error(w, fmt.Sprintf(
			"[405] %s: Method not allowed (supports %s, got %s)",
			h.name, e.allowedMethods, method), http.StatusMethodNotAllowed)
		return
	}

	var raw []byte
	if method == httpPost || method == httpPut {
		var err error
		raw, err = ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("[500] %s: unable to read body:\n  %s", h.name,
				err.Error()), http.StatusInternalServerError)
			return
		}
	}
	ret, err := e.handler.Handle(method, ids, raw)
	if err != nil {
		msg := fmt.Sprintf("[%d] %s: %s", err.StatusCode(), h.name, err.Error())
		http.Error(w, msg, err.StatusCode())
		return
	}
	if ret != nil {
		sendJSON(w, ret)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

func reg(name string, basePath string, pathItems ...pathItem) {
	http.Handle(basePath, &handler{name: name, basePath: basePath,
		path: pathItems})
}
