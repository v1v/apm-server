// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func HasPrivileges(ctx context.Context, client *Client, privileges HasPrivilegesRequest, credentials string) (HasPrivilegesResponse, error) {
	b, err := json.Marshal(privileges)
	if err != nil {
		return HasPrivilegesResponse{}, fmt.Errorf("failed to marshal has_privileges request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/_security/user/_has_privileges", bytes.NewReader(b))
	if err != nil {
		return HasPrivilegesResponse{}, fmt.Errorf("failed to create has_privileges request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")
	if credentials != "" {
		req.Header.Set("Authorization", "ApiKey "+credentials)
	}
	var info HasPrivilegesResponse
	err = doRequest(client, req, &info)
	return info, err
}

type HasPrivilegesRequest struct {
	// can't reuse the `Applications` type because here the JSON attribute must be singular
	Applications []Application `json:"application"`
}
type HasPrivilegesResponse struct {
	Username    string                             `json:"username"`
	HasAll      bool                               `json:"has_all_requested"`
	Application map[AppName]PermissionsPerResource `json:"application"`
}

type Application struct {
	Name       AppName           `json:"application"`
	Privileges []PrivilegeAction `json:"privileges"`
	Resources  []Resource        `json:"resources"`
}

type Permissions map[PrivilegeAction]bool

type PermissionsPerResource map[Resource]Permissions

type AppName string

type Resource string

// NamedPrivilege is a tuple consisting of a name and an action.
// In Elasticsearch a "privilege" represents both an "action" that a user might/might not have authorization to
// perform, and such a tuple.
// In apm-server, each name is associated with one action, but that needs not to be the case (see PrivilegeGroup)
type NamedPrivilege struct {
	Name   PrivilegeName
	Action PrivilegeAction
}

type PrivilegeAction string

type PrivilegeName string

func NewPrivilege(name, action string) NamedPrivilege {
	return NamedPrivilege{
		Name:   PrivilegeName(name),
		Action: PrivilegeAction(action),
	}
}
