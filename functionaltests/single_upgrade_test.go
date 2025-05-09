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

package functionaltests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/go-elasticsearch/v8/typedapi/types"

	"github.com/elastic/apm-server/functionaltests/internal/asserts"
	"github.com/elastic/apm-server/functionaltests/internal/ecclient"
	"github.com/elastic/apm-server/functionaltests/internal/esclient"
	"github.com/elastic/apm-server/functionaltests/internal/kbclient"
)

type additionalFunc func(t *testing.T, ctx context.Context, esc *esclient.Client, kbc *kbclient.Client) error

// singleUpgradeTestCase is a basic functional test case that performs a
// cluster upgrade between 2 specified versions.
//
// The cluster is created, some data is ingested and the first
// check is run to ensure it's in a known state.
// Then an upgrade is triggered and once completed a second check
// is run, to confirm the state did not drift after upgrade.
// A new ingestion is performed and a final check is run, to
// verify that ingestion works after upgrade and brings the cluster
// to a know state.
//
// Deprecated: To be removed soon, use testStepsRunner instead.
type singleUpgradeTestCase struct {
	fromVersion ecclient.StackVersion
	toVersion   ecclient.StackVersion
	// apmDeployMode determines whether to deploy APM in
	// managed mode (default) as opposed to standalone
	apmDeployMode apmDeploymentMode

	dataStreamNamespace          string
	setupFn                      additionalFunc
	checkPreUpgradeAfterIngest   asserts.CheckDataStreamsWant
	postUpgradeFn                additionalFunc
	checkPostUpgradeBeforeIngest asserts.CheckDataStreamsWant
	checkPostUpgradeAfterIngest  asserts.CheckDataStreamsWant

	// apmErrorLogsIgnored are the error logs to be ignored when
	// checking for existence of errors in the upgrade test.
	apmErrorLogsIgnored []types.Query
}

func (tt singleUpgradeTestCase) Run(t *testing.T) {
	integrations := tt.apmDeployMode.enableIntegrations()
	if tt.dataStreamNamespace == "" {
		tt.dataStreamNamespace = "default"
	}

	start := time.Now()
	ctx := context.Background()
	tf := initTerraformRunner(t)

	t.Log("------ cluster setup ------")
	deployInfo := createCluster(t, ctx, tf, *target, tt.fromVersion, integrations)
	t.Logf("time elapsed: %s", time.Since(start))

	esc := createESClient(t, deployInfo)
	kbc := createKibanaClient(t, deployInfo)
	g := createAPMGenerator(t, ctx, esc, kbc, deployInfo)

	atStartCount := getDocCountPerDS(t, ctx, esc)
	if tt.setupFn != nil {
		t.Log("------ custom setup ------")
		err := tt.setupFn(t, ctx, esc, kbc)
		require.NoError(t, err, "custom setup failed")
	}

	t.Log("------ pre-upgrade ingestion ------")
	require.NoError(t, g.RunBlockingWait(ctx, tt.fromVersion, integrations))
	t.Logf("time elapsed: %s", time.Since(start))

	t.Log("------ pre-upgrade ingestion assertions ------")
	t.Log("check number of documents after initial ingestion")
	firstIngestCount := getDocCountPerDS(t, ctx, esc)
	asserts.CheckDocCount(t, firstIngestCount, atStartCount,
		expectedDataStreamsIngest(tt.dataStreamNamespace))

	t.Log("check data streams after initial ingestion")
	dss, err := esc.GetDataStream(ctx, "*apm*")
	require.NoError(t, err)
	asserts.CheckDataStreams(t, tt.checkPreUpgradeAfterIngest, dss)
	t.Logf("time elapsed: %s", time.Since(start))

	t.Log("------ perform upgrade ------")
	beforeUpgradeCount := getDocCountPerDS(t, ctx, esc)
	upgradeCluster(t, ctx, tf, *target, tt.toVersion, integrations)
	t.Logf("time elapsed: %s", time.Since(start))

	if tt.postUpgradeFn != nil {
		t.Log("------ custom post-upgrade ------")
		err = tt.postUpgradeFn(t, ctx, esc, kbc)
		require.NoError(t, err, "custom post-upgrade failed")
	}

	t.Log("------ post-upgrade assertions ------")
	// We assert that no changes happened in the number of documents after upgrade
	// to ensure the state didn't change before running the next ingestion round
	// and further assertions.
	// We don't expect any change here unless something broke during the upgrade.
	t.Log("check number of documents across upgrade")
	afterUpgradeCount := getDocCountPerDS(t, ctx, esc)
	asserts.CheckDocCount(t, afterUpgradeCount, beforeUpgradeCount,
		emptyDataStreamsIngest(tt.dataStreamNamespace))

	t.Log("check data streams after upgrade")
	dss, err = esc.GetDataStream(ctx, "*apm*")
	require.NoError(t, err)
	asserts.CheckDataStreams(t, tt.checkPostUpgradeBeforeIngest, dss)

	t.Log("------ post-upgrade ingestion ------")
	require.NoError(t, g.RunBlockingWait(ctx, tt.toVersion, integrations))
	t.Logf("time elapsed: %s", time.Since(start))

	t.Log("------ post-upgrade ingestion assertions ------")
	t.Log("check number of documents after final ingestion")
	secondIngestCount := getDocCountPerDS(t, ctx, esc)
	asserts.CheckDocCount(t, secondIngestCount, afterUpgradeCount,
		expectedDataStreamsIngest(tt.dataStreamNamespace))

	t.Log("check data streams after final ingestion")
	dss2, err := esc.GetDataStream(ctx, "*apm*")
	require.NoError(t, err)
	asserts.CheckDataStreams(t, tt.checkPostUpgradeAfterIngest, dss2)
	t.Logf("time elapsed: %s", time.Since(start))

	t.Log("------ check ES and APM error logs ------")
	t.Log("checking ES error logs")
	resp, err := esc.GetESErrorLogs(ctx)
	require.NoError(t, err)
	asserts.ZeroESLogs(t, *resp)

	t.Log("checking APM error logs")
	resp, err = esc.GetAPMErrorLogs(ctx, tt.apmErrorLogsIgnored...)
	require.NoError(t, err)
	asserts.ZeroAPMLogs(t, *resp)
}
