//
// Copyright 2020 Joyent, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//

package actions

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/private/protocol/xml/xmlutil"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	tritoncompute "github.com/joyent/triton-go/v2/compute"
)

func DescribeInstances(c *gin.Context) {
	client, err := GetTritonComputeClient()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError,
			fmt.Errorf("Unable to create triton compute client: %w", err))
		return
	}

	vmListInput := &tritoncompute.ListInstancesInput{}

	vms, err := client.Instances().List(context.Background(), vmListInput)

	if err != nil {
		log.Printf("[ERROR] list vms error: %v\n", err)
		c.AbortWithError(http.StatusInternalServerError,
			fmt.Errorf("Unable to list triton compute instances: %w", err))
		return
	}

	log.Printf("[DEBUG] loaded %d vms\n", len(vms))

	// Convert Triton vm to AWS instance.
	ec2Output := ec2.DescribeInstancesOutput{}

	if len(vms) > 0 {
		res := &ec2.Reservation{}

		for _, vm := range vms {
			inst := &ec2.Instance{
				InstanceId:         &vm.ID,
				VirtualizationType: aws.String("hvm"), // Is this correct?
				ImageId:            &vm.Image,
			}
			res.Instances = append(res.Instances, inst)
		}

		ec2Output.Reservations = append(ec2Output.Reservations, res)
	}

	// Generate the XML response.
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	buf.WriteString(`<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">` + "\n")

	// Build the XML from the AWS struct.
	err = xmlutil.BuildXML(ec2Output, xml.NewEncoder(&buf))
	if err != nil {
		log.Printf("[ERROR] xmlutil.BuildXML error: %v\n", err)
		c.AbortWithError(http.StatusInternalServerError,
			fmt.Errorf("Unable to translate triton compute instances: %w", err))
		return
	}

	buf.WriteString(`</DescribeInstancesResponse>` + "\n")

	// Send the XML response.
	c.Header("content-type", "text/xml;charset=UTF-8")
	c.Data(http.StatusOK, "", buf.Bytes())
}
