//
// Copyright 2020 Joyent, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//

package actions_test

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/joyent/triton-shim/test"
)

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
