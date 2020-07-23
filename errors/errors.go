//
// Copyright 2020 Joyent, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//

package errors

import (
	"encoding/xml"
	"fmt"
)

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
