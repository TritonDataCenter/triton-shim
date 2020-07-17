package actions

import (
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"

	triton "github.com/joyent/triton-go/v2"
	tritonauth "github.com/joyent/triton-go/v2/authentication"
	tritoncompute "github.com/joyent/triton-go/v2/compute"
)

// Helper to return a CloudAPI compute client.
// TODO: This is currently hard coded to use a TRITON_ACCOUNT, we need to
// update this to pass through the CloudAPI user authentication.
func GetTritonComputeClient() (*tritoncompute.ComputeClient, error) {
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
