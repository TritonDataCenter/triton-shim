package tritonutils

import (
	triton "github.com/joyent/triton-go/v2"
	tritonauth "github.com/joyent/triton-go/v2/authentication"
	tritoncompute "github.com/joyent/triton-go/v2/compute"
)

// GetTritonComputeClient is a Helper to return a CloudAPI compute client.
func GetTritonComputeClient() (*tritoncompute.ComputeClient, error) {
	var err error
	var signer *tritonauth.Signer
	signer, err = GetTritonAuthSigner()

	if err != nil {
		return nil, err
	}

	config := &triton.ClientConfig{
		TritonURL:   triton.GetEnv("URL"),
		AccountName: triton.GetEnv("ACCOUNT"),
		Username:    triton.GetEnv("USER"),
		Signers:     []tritonauth.Signer{*signer},
	}

	return tritoncompute.NewClient(config)
}
