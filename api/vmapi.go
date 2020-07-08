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
	"strings"
)

// VmapiClient represents a connection to Triton's VMAPI
type VmapiClient struct {
	client *Client
}

// NIC is one of the VM's NICs
type NIC struct {
	IP        string `json:"ip"`
	MAC       string `json:"mac"`
	VlanID    int    `json:"vlan_id"`
	NicTag    string `json:"nic_tag"`
	Primary   bool   `json:"primary"`
	Netmask   string `json:"netmask"`
	Gateway   string `json:"gateway"`
	Network   string `json:"network_uuid"`
	Interface string `json:"interface"`
}

// VM is a VMAPI virtual machine object. We don't need to add all the fields
// provided by VMAPI, only those we plan to use
type VM struct {
	UUID             string                 `json:"uuid"`
	Alias            string                 `json:"alias"`
	Brand            string                 `json:"brand"`
	State            string                 `json:"state"`
	Image            string                 `json:"image_uuid"`
	Package          string                 `json:"billing_id"`
	Datasets         []string               `json:"datasets,omitempty"`
	RAM              int                    `json:"ram"`
	Quota            int                    `json:"quota"`
	CustomerMetadata map[string]interface{} `json:"customer_metadata"`
	InternalMetadata map[string]interface{} `json:"internal_metadata"`
	FirewallEnabled  bool                   `json:"firewall_enabled"`
	ComputeNode      string                 `json:"server_uuid"`
	Tags             map[string]interface{} `json:"tags"`
	Owner            string                 `json:"owner_uuid"`
	Nics             []NIC                  `json:"nics"`
}

// NewVmapi creates a new client object for the provided ServiceURL
func NewVmapi(VmapiURL string) (*VmapiClient, error) {
	client, err := New(VmapiURL)
	if err != nil {
		return nil, err
	}

	return &VmapiClient{
		client: client,
	}, nil
}

// ListVmsInput includes fields allowed for VM searches
type ListVmsInput struct {
	UUIDs  []string `json:"uuids,omitempty"`
	RAM    int64    `json:"ram,omitempty"`
	Alias  string   `json:"alias,omitempty"`
	Brand  string   `json:"brand,omitempty"`
	Owner  string   `json:"owner_uuid,omitempty"`
	Server string   `json:"server_uuid,omitempty"`
	State  string   `json:"state,omitempty"`
	Limit  int64    `json:"limit,omitempty"`
	Order  string   `json:"order,omitempty"`
	Sort   string   `json:"sort,omitempty"`
}

// ListVms retrieves a list of VM objects from VMAPI using the provided
// ListVmsInput as filters
func (c *VmapiClient) ListVms(ctx context.Context, input *ListVmsInput) ([]*VM, error) {
	query := &url.Values{}
	if input.UUIDs != nil {
		query.Set("uuids", strings.Join(input.UUIDs, ","))
	}
	if input.RAM != 0 {
		query.Set("ram", fmt.Sprintf("%d", input.RAM))
	}
	if input.Alias != "" {
		query.Set("alias", input.Alias)
	}
	if input.Brand != "" {
		query.Set("brand", input.Brand)
	}
	if input.Owner != "" {
		query.Set("owner_uuid", input.Owner)
	}
	if input.Server != "" {
		query.Set("server_uuid", input.Server)
	}
	if input.State != "" {
		query.Set("state", input.State)
	}
	if input.Limit != 0 {
		query.Set("limit", fmt.Sprintf("%d", input.Limit))
	}
	if input.Sort != "" {
		query.Set("sort", input.Sort)
	}
	if input.Order != "" {
		query.Set("order", input.Order)
	}

	fmt.Printf("Query: %v", query)

	reqInputs := RequestInput{
		Method: http.MethodGet,
		Path:   "/vms",
		Query:  query,
	}

	respReader, err := c.client.ExecuteRequestURIParams(ctx, reqInputs)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, err
	}

	var result []*VM
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("unable to decode list VMs response: %w", err)
	}

	return result, nil
}
