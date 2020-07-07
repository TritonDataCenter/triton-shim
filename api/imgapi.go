//
// Copyright 2020 Joyent, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// ImgapiClient represents a connection to Triton's ImagesAPI
type ImgapiClient struct {
	client *Client
}

// ImageFile represents one member of the Image.Files array
type ImageFile struct {
	Compression string `json:"compression"`
	SHA1        string `json:"sha1"`
	Size        int64  `json:"size"`
}

// Image type for Triton's ImagesAPI
type Image struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	OS           string                 `json:"os"`
	Description  string                 `json:"description"`
	Version      string                 `json:"version"`
	Type         string                 `json:"type"`
	Requirements map[string]interface{} `json:"requirements"`
	Homepage     string                 `json:"homepage"`
	Files        []*ImageFile           `json:"files"`
	PublishedAt  time.Time              `json:"published_at"`
	Owner        string                 `json:"owner"`
	Public       bool                   `json:"public"`
	State        string                 `json:"state"`
	Tags         map[string]string      `json:"tags"`
	EULA         string                 `json:"eula"`
	ACL          []string               `json:"acl"`
}

// ListImagesInput includes possible values for ListImages options.
// TODO: Add list images using a list of UUIDs like we do for Packages
type ListImagesInput struct {
	Name    string `json:"name,omitempty"`
	OS      string `json:"os,omitempty"`
	Version string `json:"version,omitempty"`
	Public  bool   `json:"public,omitempty"`
	State   string `json:"state,omitempty"`
	Owner   string `json:"owner,omitempty"`
	Type    string `json:"type,omitempty"`
}

// NewImgapi creates a new client object for the provided ServiceURL
func NewImgapi(ImgapiURL string) (*ImgapiClient, error) {
	client, err := New(ImgapiURL)
	if err != nil {
		return nil, err
	}

	return &ImgapiClient{
		client: client,
	}, nil
}

// ListImages retrieves a list of IMGAPI Images based into the provided ListImagesInput
func (c *ImgapiClient) ListImages(ctx context.Context, input *ListImagesInput) ([]*Image, error) {
	query := &url.Values{}
	if input.Name != "" {
		query.Set("name", input.Name)
	}
	if input.OS != "" {
		query.Set("os", input.OS)
	}
	if input.Version != "" {
		query.Set("version", input.Version)
	}
	if input.Public {
		query.Set("public", "true")
	}
	if input.State != "" {
		query.Set("state", input.State)
	}
	if input.Owner != "" {
		query.Set("owner", input.Owner)
	}
	if input.Type != "" {
		query.Set("type", input.Type)
	}

	fmt.Printf("Query: %v", query)

	reqInputs := RequestInput{
		Method: http.MethodGet,
		Path:   "/images",
		Query:  query,
	}

	respReader, err := c.client.ExecuteRequestURIParams(ctx, reqInputs)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, err
	}

	var result []*Image
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("unable to decode list images response: %w", err)
	}

	return result, nil
}
