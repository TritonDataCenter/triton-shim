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

func TestAccAWSDescribeInstanceTypes(t *testing.T) {
	test.GetEC2Svc(t, func(ec2Svc *ec2.EC2) {
		result, err := ec2Svc.DescribeInstanceTypes(nil)
		if err != nil {
			t.Errorf("describe instances error %v", err)
		}

		if len(result.InstanceTypes) == 0 {
			t.Errorf("describe instance types did not return any results")
		}

		for _, inst := range result.InstanceTypes {
			if *inst.MemoryInfo.SizeInMiB == 0 {
				t.Errorf("instancetype memory is zero")
			}
		}
	})
}
