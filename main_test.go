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
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter() *gin.Engine {
	engine := gin.Default()
	setupMiddleware(engine)
	setupRouter(engine)
	return engine
}

func TestPingRoute(t *testing.T) {
	router := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ping", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "pong", w.Body.String())

}

func TestDefaultAction(t *testing.T) {
	router := setupTestRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotAcceptable, w.Code)
	var xmlBytesOut XMLErrorResponse
	err := xml.Unmarshal(w.Body.Bytes(), &xmlBytesOut)
	assert.Empty(t, err)
	assert.Equal(t, xmlBytesOut.XMLName, xml.Name(xml.Name{Space: "", Local: "Response"}))
	assert.Equal(t, xmlBytesOut.Errors.XMLName, xml.Name(xml.Name{Space: "", Local: "Errors"}))
	assert.NotEmpty(t, xmlBytesOut.RequestID)
	assert.NotEmpty(t, xmlBytesOut.Errors.Error.Message)
	assert.Equal(t, "MissingAction", xmlBytesOut.Errors.Error.Code)
}

func TestNamedAction(t *testing.T) {
	router := setupTestRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?Action=DescribeRegions", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	var xmlBytesOut XMLErrorResponse
	err := xml.Unmarshal(w.Body.Bytes(), &xmlBytesOut)
	assert.Empty(t, err)
	assert.Equal(t, xmlBytesOut.XMLName, xml.Name(xml.Name{Space: "", Local: "Response"}))
	assert.Equal(t, xmlBytesOut.Errors.XMLName, xml.Name(xml.Name{Space: "", Local: "Errors"}))
	assert.NotEmpty(t, xmlBytesOut.RequestID)
	assert.NotEmpty(t, xmlBytesOut.Errors.Error.Message)
	assert.Equal(t, "InvalidAction", xmlBytesOut.Errors.Error.Code)
	assert.Regexp(t, regexp.MustCompile("DescribeRegions"), xmlBytesOut.Errors.Error.Message)
}
