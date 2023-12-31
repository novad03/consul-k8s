// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build enterprise

package partition_init

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/mitchellh/cli"
	"github.com/stretchr/testify/require"
)

func TestRun_FlagValidation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		flags  []string
		expErr string
	}{
		{
			flags:  nil,
			expErr: "addresses must be set",
		},
		{
			flags:  []string{"-addresses", "foo"},
			expErr: "-partition must be set",
		},
		{
			flags: []string{
				"-addresses", "foo", "-partition", "bar", "-api-timeout", "0s"},
			expErr: "-api-timeout must be set to a value greater than 0",
		},
		{
			flags: []string{
				"-addresses", "foo",
				"-partition", "bar",
				"-log-level", "invalid",
			},
			expErr: "unknown log level: invalid",
		},
	}

	for _, c := range cases {
		t.Run(c.expErr, func(tt *testing.T) {
			ui := cli.NewMockUi()
			cmd := Command{UI: ui}
			exitCode := cmd.Run(c.flags)
			require.Equal(tt, 1, exitCode, ui.ErrorWriter.String())
			require.Contains(tt, ui.ErrorWriter.String(), c.expErr)
		})
	}
}

func TestRun_PartitionCreate(t *testing.T) {
	partitionName := "test-partition"

	server, err := testutil.NewTestServerConfigT(t, nil)
	require.NoError(t, err)
	server.WaitForLeader(t)
	defer server.Stop()

	consul, err := api.NewClient(&api.Config{
		Address: server.HTTPAddr,
	})
	require.NoError(t, err)

	ui := cli.NewMockUi()
	cmd := Command{
		UI: ui,
	}
	cmd.init()
	args := []string{
		"-addresses=" + "127.0.0.1",
		"-http-port=" + strings.Split(server.HTTPAddr, ":")[1],
		"-grpc-port=" + strings.Split(server.GRPCAddr, ":")[1],
		"-partition", partitionName,
	}

	responseCode := cmd.Run(args)

	require.Equal(t, 0, responseCode)

	partition, _, err := consul.Partitions().Read(context.Background(), partitionName, nil)
	require.NoError(t, err)
	require.NotNil(t, partition)
	require.Equal(t, partitionName, partition.Name)
}

func TestRun_PartitionExists(t *testing.T) {
	partitionName := "test-partition"

	server, err := testutil.NewTestServerConfigT(t, nil)
	require.NoError(t, err)
	server.WaitForLeader(t)
	defer server.Stop()

	consul, err := api.NewClient(&api.Config{
		Address: server.HTTPAddr,
	})
	require.NoError(t, err)

	// Create the Admin Partition before the test runs.
	_, _, err = consul.Partitions().Create(context.Background(), &api.Partition{Name: partitionName, Description: "Created before test"}, nil)
	require.NoError(t, err)

	ui := cli.NewMockUi()
	cmd := Command{
		UI: ui,
	}
	cmd.init()
	args := []string{
		"-addresses=" + "127.0.0.1",
		"-http-port=" + strings.Split(server.HTTPAddr, ":")[1],
		"-grpc-port=" + strings.Split(server.GRPCAddr, ":")[1],
		"-partition", partitionName,
	}

	responseCode := cmd.Run(args)

	require.Equal(t, 0, responseCode)

	partition, _, err := consul.Partitions().Read(context.Background(), partitionName, nil)
	require.NoError(t, err)
	require.NotNil(t, partition)
	require.Equal(t, partitionName, partition.Name)
	require.Equal(t, "Created before test", partition.Description)
}

func TestRun_ExitsAfterTimeout(t *testing.T) {
	partitionName := "test-partition"

	server, err := testutil.NewTestServerConfigT(t, nil)
	require.NoError(t, err)

	ui := cli.NewMockUi()
	cmd := Command{
		UI: ui,
	}
	cmd.init()
	args := []string{
		"-addresses=" + "127.0.0.1",
		"-http-port=" + strings.Split(server.HTTPAddr, ":")[1],
		"-grpc-port=" + strings.Split(server.GRPCAddr, ":")[1],
		"-timeout", "500ms",
		"-partition", partitionName,
	}
	server.Stop()
	startTime := time.Now()
	responseCode := cmd.Run(args)
	completeTime := time.Now()

	require.Equal(t, 1, responseCode)
	// While the timeout is 500ms, adding a buffer of 500ms ensures we account for
	// some buffer time required for the task to run and assignments to occur.
	require.WithinDuration(t, completeTime, startTime, 1*time.Second)
}
