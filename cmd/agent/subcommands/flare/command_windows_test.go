// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024-present Datadog, Inc.

//go:build windows

// Package flare implements 'agent flare'.
package flare

import (
	"net/http"
	"net/http/httptest"

	"github.com/stretchr/testify/require"

	"github.com/DataDog/datadog-agent/pkg/config/model"
	processNet "github.com/DataDog/datadog-agent/pkg/process/net"
)

const (
	// SystemProbeTestPipeName is the test named pipe for system-probe
	systemProbeTestPipeName = `\\.\pipe\dd_system_probe_flare_test`

	// systemProbeTestPipeSecurityDescriptor has a DACL that allows Everyone access for these tests.
	systemProbeTestPipeSecurityDescriptor = "D:PAI(A;;FA;;;WD)"
)

// NewSystemProbeTestServer starts a new mock server to handle System Probe requests.
func NewSystemProbeTestServer(handler http.Handler) (*httptest.Server, error) {
	server := httptest.NewUnstartedServer(handler)

	// Override the named pipe path for tests to avoid conflicts with the locally installed Datadog agent.
	processNet.OverrideSystemProbeNamedPipeConfig(
		systemProbeTestPipeName,
		systemProbeTestPipeSecurityDescriptor)

	conn, err := processNet.NewSystemProbeListener("")
	if err != nil {
		return nil, err
	}

	server.Listener = conn.GetListener()
	return server, nil
}

// InjectConnectionFailures injects a failure in TestReadProfileDataErrors.
func InjectConnectionFailures(mockSysProbeConfig model.Config, mockConfig model.Config) {
	// Explicitly enabled system probe to exercise connections to it.
	mockSysProbeConfig.SetWithoutSource("system_probe_config.enabled", true)

	// Exercise a connection failure for a Windows system probe named pipe client by
	// making them use a bad path.
	// The system probe http server must be setup before this override.
	processNet.OverrideSystemProbeNamedPipeConfig(
		`\\.\pipe\dd_system_probe_test_bad`,
		systemProbeTestPipeSecurityDescriptor)

	// The security-agent connection is expected to fail too in this test, but
	// by enabling system probe, a port will be provided to it (security agent).
	// Here we make sure the security agent port is a bad one.
	mockConfig.SetWithoutSource("security_agent.expvar_port", 0)
}

// CheckExpectedConnectionFailures checks the expected errors after simulated
// connection failures.
func CheckExpectedConnectionFailures(c *commandTestSuite, err error) {
	// In Windows, this test explicitly simulates a system probe connection failure.
	// We expect the standard socket errors (4) and a named pipe failure for system probe.
	require.Regexp(c.T(), "^5 errors occurred:\n", err.Error())
}
