package api

import (
	"net/http"
)

// SendableError is an error that can be sent as HTTP response.
//
// This is used as error type on funcs that are invoked as part of handling an
// HTTP request.
type SendableError interface {
	error
	StatusCode() int
}

// BadRequest is an error resulting from getting a request whose parameters are
// generally or in the current state invalid.
type BadRequest struct {
	Message string
	// usually nil, but may be used to transport an error that has occurred while
	// trying to interpret parameter data.
	Inner error
}

func (br *BadRequest) Error() string {
	if br.Inner == nil {
		return br.Message
	}
	return br.Message + ": " + br.Inner.Error()
}

// StatusCode returns 400
func (BadRequest) StatusCode() int {
	return http.StatusBadRequest
}

// NotFound is an error resulting from referencing an unknown ID either in the
// URL or in parameters.
type NotFound struct {
	Name string
}

func (nf *NotFound) Error() string {
	return "Not found: \"" + nf.Name + "\""
}

// StatusCode returns 404
func (NotFound) StatusCode() int {
	return http.StatusNotFound
}

// InternalError describes an unexpected internal error. This should be used
// for problems with external systems (e.g. file system).
type InternalError struct {
	// may be empty; if not, describes the circumstances in which the error was
	// encountered.
	Description string
	// may be nil; if not, describes the error that has occurred
	Inner error
}

func (ie *InternalError) Error() string {
	msg := ""
	if ie.Description != "" {
		msg += ie.Description + ": "
	}
	if ie.Inner != nil {
		msg += ie.Inner.Error()
	}
	return msg
}

// StatusCode returns 500
func (InternalError) StatusCode() int {
	return http.StatusInternalServerError
}
