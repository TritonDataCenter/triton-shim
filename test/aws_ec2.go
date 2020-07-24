//
// Copyright 2020 Joyent, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//

package test

import (
	"fmt"
	"net"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/awstesting/unit"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/gin-gonic/gin"
	"github.com/joyent/triton-shim/server"
)

// GetEC2Svc will start a shim server and return an ec2 service that points to
// the shim server.
func GetEC2Svc(t *testing.T, runTest func(ec2Svc *ec2.EC2)) {
	// Switch to test mode so you don't get such noisy output
	gin.SetMode(gin.TestMode)
	engine := server.Setup()

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Errorf("listen error %v", err)
	}

	go func() {
		// service connections
		err := engine.RunListener(listener)
		if err != nil {
			t.Errorf("listen error %v", err)
		}
	}()

	// Load session from shared config
	awsConfig := aws.Config{
		Region:     unit.Session.Config.Region,
		DisableSSL: aws.Bool(true),
		Endpoint:   aws.String(fmt.Sprintf("http://%s", listener.Addr().String())),
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            awsConfig,
	}))

	// Create new EC2 client
	ec2Svc := ec2.New(sess)

	runTest(ec2Svc)
}
