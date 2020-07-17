package actions

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go/private/protocol/xml/xmlutil"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	tritoncompute "github.com/joyent/triton-go/v2/compute"
)

func DescribeInstanceTypes(c *gin.Context) {
	client, err := GetTritonComputeClient()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError,
			fmt.Errorf("Unable to create triton compute client: %w", err))
		return
	}

	packageListInput := &tritoncompute.ListPackagesInput{}

	packages, err := client.Packages().List(context.Background(), packageListInput)

	if err != nil {
		log.Printf("[ERROR] list packages error: %v\n", err)
		c.AbortWithError(http.StatusInternalServerError,
			fmt.Errorf("Unable to list triton compute packages: %w", err))
		return
	}

	log.Printf("[DEBUG] loaded %d packages\n", len(packages))

	// Convert Triton package to AWS AMI.
	ec2Output := ec2.DescribeInstanceTypesOutput{}

	if len(packages) > 0 {
		for _, pkg := range packages {
			instType := &ec2.InstanceTypeInfo{
				InstanceType: &pkg.Name,
				MemoryInfo:   &ec2.MemoryInfo{SizeInMiB: &pkg.Memory},
			}
			ec2Output.InstanceTypes = append(ec2Output.InstanceTypes, instType)
		}
	}

	// Generate the XML response.
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	buf.WriteString(`<DescribeInstanceTypesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">` + "\n")

	// Build the XML from the AWS struct.
	err = xmlutil.BuildXML(ec2Output, xml.NewEncoder(&buf))
	if err != nil {
		log.Printf("[ERROR] xmlutil.BuildXML error: %v\n", err)
		c.AbortWithError(http.StatusInternalServerError,
			fmt.Errorf("Unable to translate triton compute instances: %w", err))
		return
	}

	buf.WriteString(`</DescribeInstanceTypesResponse>` + "\n")

	// Send the XML response.
	c.Header("content-type", "text/xml;charset=UTF-8")
	c.Data(http.StatusOK, "", buf.Bytes())
}
