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

func TestListImagesByName(t *testing.T) {

	t.Run("Client setup", func(t *testing.T) {
		URL := os.Getenv("IMGAPI_URL")
		if URL == "" {
			URL = "http://10.99.99.21"
		}
		apiClient, err := api.NewImgapi(URL)
		if err != nil {
			t.Errorf("expected error to not be nil: received %v", err)
			return
		}

		t.Logf("API Client: %v", apiClient)

		reqInputs := &api.ListImagesInput{
			Name: "base-64-lts",
		}

		imgs, err := apiClient.ListImages(context.Background(), reqInputs)

		if err != nil {
			t.Errorf("expected error to not be nil: received %v", err)
			return
		}

		assert.GreaterOrEqual(t, len(imgs), 1)
		base64Img := imgs[0]
		assert.Equal(t, "smartos", base64Img.OS)
		assert.Equal(t, "zone-dataset", base64Img.Type)
		assert.True(t, base64Img.Public)

	})
}
