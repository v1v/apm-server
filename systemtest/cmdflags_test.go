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

package systemtest_test

import (
	"testing"

	"github.com/elastic/apm-server/systemtest/apmservertest"
)

func TestAPMServerEnvironment(t *testing.T) {
	// Check that apm-server starts up cleanly with the "--environment" flag.
	for _, env := range []string{
		"container",
		"systemd",
		"macos_service",
		"windows_service",
	} {
		env := env
		t.Run(env, func(t *testing.T) {
			t.Parallel()
			// NewServer adds a cleanup to close the server.
			apmservertest.NewServerTB(t, "--environment", env)
		})
	}
}
