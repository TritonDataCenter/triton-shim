//
// Copyright 2020 Joyent, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//

package api_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/joyent/triton-shim/api"
)

func TestListPackagesByNames(t *testing.T) {

	t.Run("Client setup", func(t *testing.T) {
		URL := os.Getenv("PAPI_URL")
		if URL == "" {
			URL = "http://10.99.99.28"
		}
		apiClient, err := api.NewPapi(URL)
		if err != nil {
			t.Errorf("expected error to not be nil: received %v", err)
			return
		}

		t.Logf("API Client: %v", apiClient)

		reqInputs := &api.ListPackagesInput{
			Names: []string{"sdc_128", "sdc_256"},
		}

		pkgs, err := apiClient.ListPackages(context.Background(), reqInputs)

		if err != nil {
			t.Errorf("expected error to not be nil: received %v", err)
			return
		}

		assert.Equal(t, 2, len(pkgs))

		for _, pkg := range pkgs {
			if pkg.Name == "sdc_128" {
				assert.Equal(t, int64(128), pkg.Memory)
			} else {
				assert.Equal(t, int64(256), pkg.Memory)
			}
		}
	})
}
