package actions

import (
	"bytes"
	"context"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/private/protocol/xml/xmlutil"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	triton "github.com/joyent/triton-go/v2"
	tritonauth "github.com/joyent/triton-go/v2/authentication"
	tritoncompute "github.com/joyent/triton-go/v2/compute"
)

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
