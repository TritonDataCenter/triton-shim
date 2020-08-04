package tritonutils

import (
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"

	triton "github.com/joyent/triton-go/v2"
	tritonauth "github.com/joyent/triton-go/v2/authentication"
)

// TODO: This is currently hard coded to use a TRITON_ACCOUNT, we need to
// update this to pass through the CloudAPI user authentication.
// For prototype purposes, the used account should be an admin account if
// we need to provide access to different accounts' resources.
// Ideally, we should use either token based auth for the application, or
// figure out a way to authenticate requests using AccessKeys against CloudAPI.

// GetTritonAuthSigner is a helper method used to retrieve a triton.Signer object
// suitable to be used with either triton compute or any other of the triton package
// clients
func GetTritonAuthSigner() (*tritonauth.Signer, error) {
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

	return &signer, nil
}
