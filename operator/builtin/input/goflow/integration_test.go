// +build integration

package goflow

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

func netFlowV5GenContainer(targetIP string) (testcontainers.Container, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:      "bindplane/nflow-generator:1.0.0",
		Entrypoint: []string{"/go/bin/nflow-generator", "-t", targetIP, "-p", "2056"},
		BindMounts: map[string]string{
			path.Join(workingDir, "testdata"): "/testdata",
		},
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}

func stanzaContainer() (testcontainers.Container, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "stanza-integration:latest",
		ExposedPorts: []string{"2056:2056"},
		Entrypoint:   []string{"/bin/sleep", "9999"},
		BindMounts: map[string]string{
			path.Join(workingDir, "testdata"): "/testdata",
		},
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}

func TestNetflowV5(t *testing.T) {
	stanza, err := stanzaContainer()
	if err != nil {
		require.NoError(t, err)
		return
	}
	defer func() {
		err := stanza.Terminate(context.Background())
		require.NoError(t, err, "failed to cleanup test container")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := stanza.Exec(ctx, []string{"sh", "/testdata/netflowv5.sh"}); err != nil {
		if !strings.Contains(err.Error(), "context deadline exceeded") {
			require.NoError(t, err)
			return
		}
	}

	stanzaIP, err := stanza.ContainerIP(context.Background())
	if err != nil {
		require.NoError(t, err, "expected nil error when getting stanza container ip")
		return
	}

	loadgen, err := netFlowV5GenContainer(stanzaIP)
	if err != nil {
		require.NoError(t, err)
		return
	}
	defer func() {
		err := loadgen.Terminate(context.Background())
		require.NoError(t, err, "failed to cleanup test container")
	}()

	time.Sleep(time.Second * 2)

	// Read entire files into memory. Don't do this if you expect the output to be massive.
	// These files are reset to 0 lines on every run in testdata/netflowv5.sh
	//
	// Expect stanza stdout / stderr to contain zero output
	b, err := ioutil.ReadFile("./testdata/stdout.log")
	if err != nil {
		require.NoError(t, err, "expected to read testdata/stdout.log")
		return
	}
	require.Equal(t, 0, len(b), "expected stdout.log to be empty, this indicates the agent paniced or logged an error")

	// Expect stanza's log to contain exactly 3 lines
	/*
		{"level":"info","timestamp":"2021-06-15T21:03:10.763Z","message":"Starting stanza agent"}
		{"level":"info","timestamp":"2021-06-15T21:03:10.763Z","message":"Started Goflow on 0.0.0.0:2056 in netflow_v5 mode","operator_id":"$.goflow_input","operator_type":"goflow_input"}
		{"level":"info","timestamp":"2021-06-15T21:03:10.763Z","message":"Stanza agent started"}
		< new line >
	*/
	b, err = ioutil.ReadFile("./testdata/stanza.log") // just pass the file name
	if err != nil {
		require.NoError(t, err, "expected to read testdata/stanza.log")
		return
	}
	require.Equal(t, 4, len(strings.Split(string(b), "\n")), "expected stanza.log to contain exactly 3 lines")

	// Expect file_output to contain parsed logs, the test will usually dump 40+ logs
	/**
	{"timestamp":"2021-06-15T21:10:35Z","severity":0,"record":{"bytes":329,"dstaddr":"172.30.190.10","dstas":30207,"dstnet":24,"dstport":161,"etype":2048,"nexthop":"172.199.15.1","packets":663,"proto":17,"sampleraddress":"172.17.0.4","sequencenum":1,"srcaddr":"112.10.20.10","srcas":64858,"srcport":40,"timeflowend":1623791435,"timeflowstart":1623791435,"type":2}}
	{"timestamp":"2021-06-15T21:10:35Z","severity":0,"record":{"bytes":177,"dstaddr":"132.12.130.10","dstas":23810,"dstnet":11,"etype":2048,"nexthop":"132.12.130.1","packets":233,"proto":1,"sampleraddress":"172.17.0.4","sequencenum":1,"srcaddr":"172.16.50.10","srcas":44850,"srcnet":1,"timeflowend":1623791435,"timeflowstart":1623791435,"type":2}}
	{"timestamp":"2021-06-15T21:10:35Z","severity":0,"record":{"bytes":1006,"dstaddr":"242.164.127.44","dstas":45852,"dstport":19218,"etype":2048,"nexthop":"130.148.218.75","packets":1015,"proto":6,"sampleraddress":"172.17.0.4","sequencenum":1,"srcaddr":"165.148.192.105","srcas":2641,"srcnet":27,"srcport":39402,"timeflowend":1623791435,"timeflowstart":1623791435,"type":2}}
	**/
	b, err = ioutil.ReadFile("./testdata/out.log") // just pass the file name
	if err != nil {
		require.NoError(t, err, "expected to read testdata/out.log")
		return
	}
	lines := len(strings.Split(string(b), "\n"))
	require.Greater(t, lines, 5, "expected out.log to contain log entries")
}
