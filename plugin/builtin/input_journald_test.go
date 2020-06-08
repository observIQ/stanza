// +build linux

package builtin

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/bluemedora/bplogagent/plugin/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fakeJournaldCmd struct{}

func (f *fakeJournaldCmd) Start() error {
	return nil
}

func (f *fakeJournaldCmd) StdoutPipe() (io.ReadCloser, error) {
	response := `{ "_BOOT_ID": "c4fa36de06824d21835c05ff80c54468", "_CAP_EFFECTIVE": "0", "_TRANSPORT": "journal", "_UID": "1000", "_EXE": "/usr/lib/systemd/systemd", "_AUDIT_LOGINUID": "1000", "MESSAGE": "run-docker-netns-4f76d707d45f.mount: Succeeded.", "_PID": "13894", "_CMDLINE": "/lib/systemd/systemd --user", "_MACHINE_ID": "d777d00e7caf45fbadedceba3975520d", "_SELINUX_CONTEXT": "unconfined\n", "CODE_FUNC": "unit_log_success", "SYSLOG_IDENTIFIER": "systemd", "_HOSTNAME": "myhostname", "MESSAGE_ID": "7ad2d189f7e94e70a38c781354912448", "_SYSTEMD_CGROUP": "/user.slice/user-1000.slice/user@1000.service/init.scope", "_SOURCE_REALTIME_TIMESTAMP": "1587047866229317", "USER_UNIT": "run-docker-netns-4f76d707d45f.mount", "SYSLOG_FACILITY": "3", "_SYSTEMD_SLICE": "user-1000.slice", "_AUDIT_SESSION": "286", "CODE_FILE": "../src/core/unit.c", "_SYSTEMD_USER_UNIT": "init.scope", "_COMM": "systemd", "USER_INVOCATION_ID": "88f7ca6bbf244dc8828fa901f9fe9be1", "CODE_LINE": "5487", "_SYSTEMD_INVOCATION_ID": "83f7fc7799064520b26eb6de1630429c", "PRIORITY": "6", "_GID": "1000", "__REALTIME_TIMESTAMP": "1587047866229555", "_SYSTEMD_UNIT": "user@1000.service", "_SYSTEMD_USER_SLICE": "-.slice", "__CURSOR": "s=b1e713b587ae4001a9ca482c4b12c005;i=1eed30;b=c4fa36de06824d21835c05ff80c54468;m=9f9d630205;t=5a369604ee333;x=16c2d4fd4fdb7c36", "__MONOTONIC_TIMESTAMP": "685540311557", "_SYSTEMD_OWNER_UID": "1000" }
`
	reader := bytes.NewReader([]byte(response))
	return ioutil.NopCloser(reader), nil
}

func TestInputJournald(t *testing.T) {
	cfg := JournaldInputConfig{
		InputConfig: helper.InputConfig{
			BasicConfig: helper.BasicConfig{
				PluginID:   "my_journald_input",
				PluginType: "journald_input",
			},
			OutputID: "output",
		},
	}

	journaldInput, err := cfg.Build(testutil.NewTestBuildContext(t))
	require.NoError(t, err)

	mockOutput := testutil.NewMockOutput("output")
	received := make(chan *entry.Entry)
	mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		received <- args.Get(1).(*entry.Entry)
	}).Return(nil)

	err = journaldInput.SetOutputs([]plugin.Plugin{mockOutput})
	require.NoError(t, err)

	journaldInput.(*JournaldInput).newCmd = func(ctx context.Context, cursor []byte) cmd {
		return &fakeJournaldCmd{}
	}

	err = journaldInput.Start()
	require.NoError(t, err)

	expected := map[string]string{
		"_BOOT_ID":                   "c4fa36de06824d21835c05ff80c54468",
		"_CAP_EFFECTIVE":             "0",
		"_TRANSPORT":                 "journal",
		"_UID":                       "1000",
		"_EXE":                       "/usr/lib/systemd/systemd",
		"_AUDIT_LOGINUID":            "1000",
		"MESSAGE":                    "run-docker-netns-4f76d707d45f.mount: Succeeded.",
		"_PID":                       "13894",
		"_CMDLINE":                   "/lib/systemd/systemd --user",
		"_MACHINE_ID":                "d777d00e7caf45fbadedceba3975520d",
		"_SELINUX_CONTEXT":           "unconfined\n",
		"CODE_FUNC":                  "unit_log_success",
		"SYSLOG_IDENTIFIER":          "systemd",
		"_HOSTNAME":                  "myhostname",
		"MESSAGE_ID":                 "7ad2d189f7e94e70a38c781354912448",
		"_SYSTEMD_CGROUP":            "/user.slice/user-1000.slice/user@1000.service/init.scope",
		"_SOURCE_REALTIME_TIMESTAMP": "1587047866229317",
		"USER_UNIT":                  "run-docker-netns-4f76d707d45f.mount",
		"SYSLOG_FACILITY":            "3",
		"_SYSTEMD_SLICE":             "user-1000.slice",
		"_AUDIT_SESSION":             "286",
		"CODE_FILE":                  "../src/core/unit.c",
		"_SYSTEMD_USER_UNIT":         "init.scope",
		"_COMM":                      "systemd",
		"USER_INVOCATION_ID":         "88f7ca6bbf244dc8828fa901f9fe9be1",
		"CODE_LINE":                  "5487",
		"_SYSTEMD_INVOCATION_ID":     "83f7fc7799064520b26eb6de1630429c",
		"PRIORITY":                   "6",
		"_GID":                       "1000",
		"_SYSTEMD_UNIT":              "user@1000.service",
		"_SYSTEMD_USER_SLICE":        "-.slice",
		"__CURSOR":                   "s=b1e713b587ae4001a9ca482c4b12c005;i=1eed30;b=c4fa36de06824d21835c05ff80c54468;m=9f9d630205;t=5a369604ee333;x=16c2d4fd4fdb7c36",
		"__MONOTONIC_TIMESTAMP":      "685540311557",
		"_SYSTEMD_OWNER_UID":         "1000",
	}

	select {
	case e := <-received:
		require.Equal(t, expected, e.Record)
	case <-time.After(time.Second):
		require.FailNow(t, "Timed out waiting for entry to be read")
	}
}
