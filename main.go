//
// Copyright 2020 Joyent, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//

package main

import (
	"bytes"
	"context"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/private/protocol/xml/xmlutil"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/gin-gonic/gin"

	triton "github.com/joyent/triton-go/v2"
	tritonauth "github.com/joyent/triton-go/v2/authentication"
	tritoncompute "github.com/joyent/triton-go/v2/compute"

	"github.com/pborman/uuid"

	"github.com/joyent/triton-shim/utils"
)

// TODO: Move all the errors to their own file

type xmlError struct {
	XMLName xml.Name `xml:"Error"`
	Code    string   `xml:"Code"`
	Message string   `xml:"Message"`
}
type xmlErrors struct {
	XMLName xml.Name `xml:"Errors"`
	Error   *xmlError
}

// XMLErrorResponse marshals error responses to XML
type XMLErrorResponse struct {
	XMLName   xml.Name `xml:"Response"`
	Errors    *xmlErrors
	RequestID string `xml:"RequestId"`
}

// ResponseError wraps the given Error Code & Message into XML
func ResponseError(Code string, Message string, RequestID string) *XMLErrorResponse {
	return &XMLErrorResponse{
		Errors: &xmlErrors{
			Error: &xmlError{
				Code:    Code,
				Message: Message,
			},
		},
		RequestID: RequestID,
	}
}

// MissingActionError wraps XML error when Action argument is not present
func MissingActionError(RequestID string) *XMLErrorResponse {
	return ResponseError("MissingAction", "Action parameter must be provided", RequestID)
}

// InvalidActionError will be used for unsupported actions
func InvalidActionError(Action string, RequestID string) *XMLErrorResponse {
	return ResponseError("InvalidAction", fmt.Sprintf("Action %s is not supported", Action), RequestID)
}

// Helper to return a CloudAPI compute client.
// TODO: This is currently hard coded to use a TRITON_ACCOUNT, we need to
// update this to pass through the CloudAPI user authentication.
func getComputeClient() (*tritoncompute.ComputeClient, error) {
	var err error
	var signer tritonauth.Signer

	account := triton.GetEnv("ACCOUNT")
	keyID := triton.GetEnv("KEY_ID")
	keyMaterial := triton.GetEnv("KEY_MATERIAL")
	// skipTLSVerify := triton.GetEnv("TRITON_SKIP_TLS_VERIFY")
	username := triton.GetEnv("USER")

	if keyMaterial == "" {
		signer, err = tritonauth.NewSSHAgentSigner(tritonauth.SSHAgentSignerInput{
			KeyID:       keyID,
			AccountName: account,
			Username:    username,
		})
		if err != nil {
			return nil, fmt.Errorf("Error creating SSH agent signer: %w", err)
		}
	} else {
		var keyBytes []byte
		if _, err = os.Stat(keyMaterial); err == nil {
			keyBytes, err = ioutil.ReadFile(keyMaterial)
			if err != nil {
				return nil, fmt.Errorf("Error reading key material from %s: %w",
					keyMaterial, err)
			}
			block, _ := pem.Decode(keyBytes)
			if block == nil {
				return nil, fmt.Errorf(
					"Failed to read key material '%s': no key found", keyMaterial)
			}

			if block.Headers["Proc-Type"] == "4,ENCRYPTED" {
				return nil, fmt.Errorf(
					"Failed to read key '%s': password protected keys are\n"+
						"not currently supported. Please decrypt the key prior to use.", keyMaterial)
			}

		} else {
			keyBytes = []byte(keyMaterial)
		}

		signer, err = tritonauth.NewPrivateKeySigner(tritonauth.PrivateKeySignerInput{
			KeyID:              keyID,
			PrivateKeyMaterial: keyBytes,
			AccountName:        account,
			Username:           username,
		})

		if err != nil {
			return nil, fmt.Errorf("Error creating SSH private key signer: %w", err)
		}
	}

	config := &triton.ClientConfig{
		TritonURL:   triton.GetEnv("URL"),
		AccountName: triton.GetEnv("ACCOUNT"),
		Username:    triton.GetEnv("USER"),
		Signers:     []tritonauth.Signer{signer},
	}

	return tritoncompute.NewClient(config)
}

func DescribeInstances(c *gin.Context) {
	client, err := getComputeClient()
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

func actionHandler(c *gin.Context, action string) {
	reqID := uuid.New()

	switch action {
	case "DescribeInstances":
		DescribeInstances(c)

	// Action not specified
	case "MissingAction":
		xml := MissingActionError(reqID)
		c.XML(http.StatusNotAcceptable, xml)

	// All the implemented actions should be before the default case,
	// which assumes that the action hasn't been implemented and will
	// return a MethodNotAllowed Error
	default:
		xml := InvalidActionError(action, reqID)
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

func main() {
	engine := gin.Default()

	setupMiddleware(engine)
	setupRouter(engine)

	// Start listening.
	engine.Run(":9090")
}
