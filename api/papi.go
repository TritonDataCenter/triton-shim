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
	"log"
	"net/http"
	"net/url"
	"strings"
)

// PapiClient represents a connection to Triton's PackagesAPI
type PapiClient struct {
	client *Client
}

// PackageDisk can have arbitrary properties, not always present
// or mutually exclusive
type PackageDisk struct {
	Size       interface{}
	SizeInMiB  int64
	Remaining  bool
	OSDiskSize bool
}

// UnmarshalJSON for PackageDisk custom type
func (d *PackageDisk) UnmarshalJSON(data []byte) error {
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		log.Fatal(err)
		return err
	}
	if decoded["size"] != nil {
		if err := json.Unmarshal(decoded["size"], &d.Size); err != nil {
			log.Fatal(err)
			return err
		}
	}
	switch d.Size.(type) {
	case string:
		d.Remaining = true
		d.OSDiskSize = false
	case nil:
		d.Remaining = false
		d.OSDiskSize = true
	default:
		d.Remaining = false
		d.OSDiskSize = false
		d.SizeInMiB = int64(d.Size.(float64))
	}
	return nil
}

// Package is equivalent to a Triton's PackagesAPI package
type Package struct {
	UUID          string        `json:"uuid"`
	Name          string        `json:"name"`
	Memory        int64         `json:"max_physical_memory"`
	Disk          int64         `json:"disk"`
	Swap          int64         `json:"max_swap"`
	LWPs          int64         `json:"max_lwps"`
	VCPUs         int64         `json:"vcpus"`
	V             int64         `json:"v"`
	CPUCap        int64         `json:"cpu_cap"`
	FSS           int64         `json:"fss"`
	ZFSIOPriority int64         `json:"zfs_io_priority"`
	Quota         int64         `json:"quota"`
	Version       string        `json:"version"`
	Group         string        `json:"group"`
	Description   string        `json:"description"`
	Default       bool          `json:"default"`
	Active        bool          `json:"active"`
	Brand         string        `json:"brand"`
	FlexibleDisk  bool          `json:"flexible_disk,omitempty"`
	Disks         []PackageDisk `json:"disks,omitempty"`
	BillingTag    string        `json:"billing_tag"`
	CreatedAt     string        `json:"created_at"`
	UpdatedAt     string        `json:"updated_at"`
	Owners        []string      `json:"owners"`
}

// ListPackagesInput includes possible values for ListPackages options. While
// there are more possibilities here, we're only interested into the following
// options:
// - Filter (string) LDAP filter string. When provided, everything else should
//   be ignored.
// - Names ([]string) One or more package names.
// - Memory (int64) Max Physical Memory in Megabytes
// - Version (string) Package version to difference between different versions
//   of the same package type
// - Group (string) Retrieve just a group of packages
// - Brand (string) Equivalent to Hypervisor for other cloud providers.
// Usually, the most common filter will be a collection of the package names to
// be retrieved.
// Pagination options are also supported: Limit(int64), Offset(int64),
// Order(string) and Sort(string). By default, packages listing will be limited
// to 1k records, sorted ASCending by the internal _id attribute (i.e, same
// order they were created into Triton's Package API)
type ListPackagesInput struct {
	Filter  string   `json:"filter,omitempty"`
	Names   []string `json:"names,omitempty"`
	Memory  int64    `json:"memory,omitempty"`
	Version string   `json:"version,omitempty"`
	Group   string   `json:"group,omitempty"`
	Brand   string   `json:"brand,omitempty"`
	Limit   int64    `json:"limit,omitempty"`
	Offset  int64    `json:"offset,omitempty"`
	Order   string   `json:"order,omitempty"`
	Sort    string   `json:"sort,omitempty"`
}

// NewPapi creates a new client object for the provided ServiceURL
func NewPapi(PapiURL string) (*PapiClient, error) {
	client, err := New(PapiURL)
	if err != nil {
		return nil, err
	}

	return &PapiClient{
		client: client,
	}, nil
}

// ListPackages provides a list of packages retrieved from Triton's PackagesAPI
// using the provided ListPackagesInput
func (c *PapiClient) ListPackages(ctx context.Context, input *ListPackagesInput) ([]*Package, error) {
	query := &url.Values{}
	if input.Filter != "" {
		query.Set("filter", input.Filter)
	} else {
		if input.Names != nil {
			query.Set("name", fmt.Sprintf("[\"%s\"]", strings.Join(input.Names, "\",\"")))
		}
		if input.Memory != 0 {
			query.Set("memory", fmt.Sprintf("%d", input.Memory))
		}
		if input.Limit != 0 {
			query.Set("disk", fmt.Sprintf("%d", input.Limit))
		}
		if input.Offset != 0 {
			query.Set("swap", fmt.Sprintf("%d", input.Offset))
		}
		if input.Sort != "" {
			query.Set("sort", input.Sort)
		}
		if input.Order != "" {
			query.Set("order", input.Order)
		}
		if input.Version != "" {
			query.Set("version", input.Version)
		}
		if input.Group != "" {
			query.Set("group", input.Group)
		}
		if input.Brand != "" {
			query.Set("brand", input.Brand)
		}
	}

	fmt.Printf("Query: %v", query)

	reqInputs := RequestInput{
		Method: http.MethodGet,
		Path:   "/packages",
		Query:  query,
	}

	respReader, err := c.client.ExecuteRequestURIParams(ctx, reqInputs)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, err
	}

	var result []*Package
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("unable to decode list packages response: %w", err)
	}

	return result, nil
}
