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

func TestAccAWSDescribeImages(t *testing.T) {
	test.GetEC2Svc(t, func(ec2Svc *ec2.EC2) {
		result, err := ec2Svc.DescribeImages(nil)
		if err != nil {
			t.Errorf("describe images error %v", err)
		}

		// fmt.Printf("Success: %+v\n", result)
		if len(result.Images) == 0 {
			t.Errorf("describe images did not return any images")
		}

		for _, img := range result.Images {
			if *img.State != ec2.ImageStateAvailable {
				t.Errorf("image state should be 'available', got '%s'",
					*img.State)
			}
		}
	})
}
