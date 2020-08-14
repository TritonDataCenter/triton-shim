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
	tritonutils "github.com/joyent/triton-shim/utils/triton"
)

func imageConvertState(state string) string {
	switch state {
	case "active":
		return "available"
	case "unactivated":
		return "pending"
	case "disabled":
		return "deregistered"
	case "creating":
		return "pending"
	case "failed":
		return "failed"
	default:
		return "invalid"
	}
}

func convertTagMapToTagset(tags map[string]string) []*ec2.Tag {
	var ec2Tags []*ec2.Tag
	for k, v := range tags {
		ec2Tags = append(ec2Tags, &ec2.Tag{Key: aws.String(k), Value: aws.String(v)})
	}
	return ec2Tags
}

func DescribeImages(c *gin.Context) {
	client, err := tritonutils.GetTritonComputeClient()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError,
			fmt.Errorf("Unable to create triton compute client: %w", err))
		return
	}

	vmListInput := &tritoncompute.ListImagesInput{}

	images, err := client.Images().List(context.Background(), vmListInput)

	if err != nil {
		log.Printf("[ERROR] list images error: %v\n", err)
		c.AbortWithError(http.StatusInternalServerError,
			fmt.Errorf("Unable to list triton compute images: %w", err))
		return
	}

	log.Debug().Msgf("loaded %d images\n", len(images))

	// Convert Triton vm to AWS image.
	ec2Output := ec2.DescribeImagesOutput{}

	for _, img := range images {
		awsImage := &ec2.Image{
			ImageId:      &img.ID,
			CreationDate: aws.String(img.PublishedAt.String()),
			Description:  aws.String(img.Description),
			ImageType:    aws.String(img.Type),
			Name:         aws.String(img.Name),
			Public:       aws.Bool(img.Public),
			State:        aws.String(imageConvertState(img.State)),
			Tags:         convertTagMapToTagset(img.Tags),
			// Hypervizor: aws.String("ovm"),
			// VirtualizationType: aws.String(img.OS),
		}
		ec2Output.Images = append(ec2Output.Images, awsImage)
	}

	// Generate the XML response.
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	buf.WriteString(`<DescribeImagesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">` + "\n")

	// Build the XML from the AWS struct.
	err = xmlutil.BuildXML(ec2Output, xml.NewEncoder(&buf))
	if err != nil {
		log.Printf("[ERROR] xmlutil.BuildXML error: %v\n", err)
		c.AbortWithError(http.StatusInternalServerError,
			fmt.Errorf("Unable to translate triton compute images: %w", err))
		return
	}

	buf.WriteString(`</DescribeImagesResponse>` + "\n")

	// Send the XML response.
	c.Header("content-type", "text/xml;charset=UTF-8")
	c.Data(http.StatusOK, "", buf.Bytes())
}
