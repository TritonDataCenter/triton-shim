//
// Copyright 2020 Joyent, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//

package server

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"github.com/google/uuid"

	"github.com/joyent/triton-shim/actions"
	"github.com/joyent/triton-shim/errors"
	"github.com/joyent/triton-shim/utils"
)

func actionHandler(c *gin.Context, action string) {
	reqID := uuid.New().String()

	switch action {
	case "DescribeInstances":
		actions.DescribeInstances(c)
	case "DescribeInstanceTypes":
		actions.DescribeInstanceTypes(c)

	// Action not specified
	case "MissingAction":
		xml := errors.MissingActionError(reqID)
		c.XML(http.StatusNotAcceptable, xml)

	// All the implemented actions should be before the default case,
	// which assumes that the action hasn't been implemented and will
	// return a MethodNotAllowed Error
	default:
		xml := errors.InvalidActionError(action, reqID)
		c.XML(http.StatusMethodNotAllowed, xml)
	}
}

func getPostAction(c *gin.Context) (string, error) {
	// TODO: Better way to read the POST body - but not also having to read in
	// megabytes of data?
	buf := make([]byte, 10000)
	n, err := c.Request.Body.Read(buf)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("Unable to parse request body: %w", err)
	}

	input, err := url.ParseQuery(string(buf[:n]))
	if err != nil {
		return "", fmt.Errorf("Unable to parse request body: %w", err)
	}

	return input.Get("Action"), nil
}

func setupRouter(router *gin.Engine) {
	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	router.GET("/", func(c *gin.Context) {
		action := c.DefaultQuery("Action", "MissingAction")

		log.Printf("[DEBUG] GET action: '%s'\n", action)

		actionHandler(c, action)
	})

	router.POST("/", func(c *gin.Context) {
		action, err := getPostAction(c)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		log.Printf("[DEBUG] POST action: '%s'\n", action)

		actionHandler(c, action)

		c.Next()
	})
}

func setupMiddleware(engine *gin.Engine) {
	engine.Use(utils.ShimLogger())
}

func Setup() *gin.Engine {
	engine := gin.Default()

	setupMiddleware(engine)
	setupRouter(engine)

	return engine
}
