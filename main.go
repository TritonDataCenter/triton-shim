//
// Copyright 2020 Joyent, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//

package main

import (
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pborman/uuid"
)

// TODO: Move all the errors to their own file

type xmlError struct {
	XMLName xml.Name `xml:"Error"`
	Code    string   `xml:"Code"`
	Message string   `xml:"Message"`
}
type xmlErrors struct {
	XMLName xml.Name `xml:"Errors"`
	Error   *xmlError
}

// XMLErrorResponse marshals error responses to XML
type XMLErrorResponse struct {
	XMLName   xml.Name `xml:"Response"`
	Errors    *xmlErrors
	RequestID string `xml:"RequestId"`
}

// ResponseError wraps the given Error Code & Message into XML
func ResponseError(Code string, Message string, RequestID string) *XMLErrorResponse {
	return &XMLErrorResponse{
		Errors: &xmlErrors{
			Error: &xmlError{
				Code:    Code,
				Message: Message,
			},
		},
		RequestID: RequestID,
	}
}

// MissingActionError wraps XML error when Action argument is not present
func MissingActionError(RequestID string) *XMLErrorResponse {
	return ResponseError("MissingAction", "Action parameter must be provided", RequestID)
}

// InvalidActionError will be used for unsupported actions
func InvalidActionError(Action string, RequestID string) *XMLErrorResponse {
	return ResponseError("InvalidAction", fmt.Sprintf("Action %s is not supported", Action), RequestID)
}

func setupRouter() *gin.Engine {
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	r.GET("/", func(c *gin.Context) {
		action := c.DefaultQuery("Action", "MissingAction")
		reqID := uuid.New()
		switch action {
		// Action not specified
		case "MissingAction":
			xml := MissingActionError(reqID)
			c.XML(http.StatusNotAcceptable, xml)
		// All the implemented actions should be before the default case,
		// which assumes that the action hasn't been implemented and will
		// return a MethodNotAllowed Error
		default:
			xml := InvalidActionError(action, reqID)
			c.XML(http.StatusMethodNotAllowed, xml)
		}
	})

	return r
}

func main() {
	r := setupRouter()
	// Listen and Serve in 0.0.0.0:9090
	r.Run(":9090")
}
