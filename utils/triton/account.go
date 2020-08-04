package tritonutils

import (
	triton "github.com/joyent/triton-go/v2"
	tritonaccount "github.com/joyent/triton-go/v2/account"
	tritonauth "github.com/joyent/triton-go/v2/authentication"
)

// GetTritonAccountClient is a Helper to return a CloudAPI account client used to manipulate
// AccessKeys
func GetTritonAccountClient() (*tritonaccount.AccountClient, error) {
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

	return tritonaccount.NewClient(config)
}
