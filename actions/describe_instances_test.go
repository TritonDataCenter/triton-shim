//
// Copyright 2020 Joyent, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//

package actions_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/gin-gonic/gin"
	"github.com/joyent/triton-shim/server"
	"github.com/joyent/triton-shim/test"
)

func TestAccDescribeInstances(t *testing.T) {
	// Switch to test mode so you don't get such noisy output
	gin.SetMode(gin.TestMode)

	engine := server.Setup()

	req, err := http.NewRequest(http.MethodPost, "/", bytes.NewBufferString("Action=DescribeInstances"))
	if err != nil {
		t.Fatalf("Couldn't create request: %v\n", err)
	}

	// Create a response recorder so we can inspect the response
	w := httptest.NewRecorder()

	// Perform the request
	engine.ServeHTTP(w, req)

	// log.Printf("[ERROR]: Response body: %s", w.Body.String())
	// Check the response
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

}

func TestAccAWSDescribeInstances(t *testing.T) {
	test.GetEC2Svc(t, func(ec2Svc *ec2.EC2) {
		result, err := ec2Svc.DescribeInstances(nil)
		if err != nil {
			t.Errorf("describe instances error %v", err)
		}

		// fmt.Printf("Success: %+v\n", result)
		if len(result.Reservations) == 0 {
			t.Errorf("describe instances does not have any reservations")
		}
		if len(result.Reservations[0].Instances) == 0 {
			t.Errorf("describe instances did not return any results")
		}

		for _, inst := range result.Reservations[0].Instances {
			if *inst.VirtualizationType != "hvm" {
				t.Errorf("instance VirtualizationType should be hvm, got: %s",
					*inst.VirtualizationType)
			}
		}
	})
}
