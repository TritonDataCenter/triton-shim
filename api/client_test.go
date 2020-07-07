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
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/joyent/triton-shim/api"
)

func TestNew(t *testing.T) {

	t.Run("Client setup", func(t *testing.T) {
		URL := os.Getenv("PAPI_URL")
		if URL == "" {
			URL = "http://10.99.99.28"
		}
		apiClient, err := api.New(URL)
		if err != nil {
			t.Errorf("expected error to not be nil: received %v", err)
			return
		}

		t.Logf("API Client: %v", apiClient)

		query := &url.Values{}
		query.Set("name", "sdc_128")

		reqInputs := api.RequestInput{
			Method: http.MethodGet,
			Path:   "/packages",
			Query:  query,
		}

		res, err := apiClient.ExecuteRequestURIParams(context.Background(), reqInputs)
		if res != nil {
			defer res.Close()
		}
		if err != nil {
			t.Errorf("expected error to not be nil: received %v", err)
			return
		}

		// No need to mess with package details here
		var result []map[string]interface{}
		decoder := json.NewDecoder(res)

		if err = decoder.Decode(&result); err != nil {
			t.Errorf("unable to decode response: %v", err)
			return
		}

		t.Logf("API Client response: %v", result)
	})
}
