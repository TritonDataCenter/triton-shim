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

func TestListVMs(t *testing.T) {

	t.Run("Client setup", func(t *testing.T) {
		URL := os.Getenv("VMAPI_URL")
		if URL == "" {
			URL = "http://10.99.99.26"
		}
		apiClient, err := api.NewVmapi(URL)
		if err != nil {
			t.Errorf("expected error to not be nil: received %v", err)
			return
		}

		t.Logf("API Client: %v", apiClient)

		reqInputs := &api.ListVmsInput{}

		vms, err := apiClient.ListVms(context.Background(), reqInputs)

		if err != nil {
			t.Errorf("expected error to not be nil: received %v", err)
			return
		}

		assert.NotEmpty(t, vms)
		// Let's grab some existing VMs to test searching by uuid
		someVms := vms[:3]
		uuids := []string{}
		for _, vm := range someVms {
			uuids = append(uuids, vm.UUID)
		}
		reqInputsByUUID := &api.ListVmsInput{
			UUIDs: uuids,
		}

		vmsByUUID, err := apiClient.ListVms(context.Background(), reqInputsByUUID)

		if err != nil {
			t.Errorf("expected error to not be nil: received %v", err)
			return
		}

		assert.NotEmpty(t, vmsByUUID)
		assert.Equal(t, len(uuids), len(vmsByUUID))
	})
}
