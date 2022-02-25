package counter

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/observiq/stanza/v2/testutil"
	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	"github.com/open-telemetry/opentelemetry-log-collection/operator/helper"
	"github.com/stretchr/testify/require"
)

func TestBuildValid(t *testing.T) {
	cfg := NewCounterOutputConfig("test")
	ctx := testutil.NewBuildContext(t)
	ops, err := cfg.Build(ctx)
	require.NoError(t, err)
	op := ops[0]
	require.IsType(t, &CountOutput{}, op)
}

func TestBuildInvalid(t *testing.T) {
	cfg := NewCounterOutputConfig("test")
	ctx := testutil.NewBuildContext(t)
	ctx.Logger = nil
	_, err := cfg.Build(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "build context is missing a logger")
}

func TestFileCounterOutput(t *testing.T) {
	persister := &testutil.MockPersister{}
	cfg := NewCounterOutputConfig("test")

	tmpFile, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	cfg.Path = tmpFile.Name()
	cfg.Duration = helper.NewDuration(1 * time.Second)

	ctx := testutil.NewBuildContext(t)
	ops, err := cfg.Build(ctx)
	require.NoError(t, err)

	counterOutput := ops[0].(*CountOutput)

	err = counterOutput.Start(persister)
	require.NoError(t, err)
	defer func() {
		err := counterOutput.Stop()
		require.NoError(t, err)
	}()

	e := entry.New()
	err = counterOutput.Process(context.Background(), e)
	require.NoError(t, err)
	require.Equal(t, counterOutput.numEntries, uint64(1))

	stat, err := os.Stat(tmpFile.Name())
	require.NoError(t, err)

	intialSize := stat.Size()

	to, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ticker := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case <-to.Done():
			require.FailNow(t, "timed out waiting for file to be written to")
		case <-ticker.C:
		}
		size, err := os.Stat(tmpFile.Name())
		require.NoError(t, err)
		if size.Size() != intialSize {
			break
		}
	}

	content, err := ioutil.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	var object countObject
	err = json.Unmarshal(content, &object)
	require.NoError(t, err)

	require.Equal(t, object.Entries, uint64(1))
	require.GreaterOrEqual(t, object.EntriesPerMinute, 0.0)
	require.GreaterOrEqual(t, object.ElapsedMinutes, 0.0)
}

func TestStartStdout(t *testing.T) {
	persister := &testutil.MockPersister{}
	cfg := NewCounterOutputConfig("test")

	ctx := testutil.NewBuildContext(t)
	ops, err := cfg.Build(ctx)
	require.NoError(t, err)

	counterOutput := ops[0].(*CountOutput)

	err = counterOutput.Start(persister)
	defer func() {
		err := counterOutput.Stop()
		require.NoError(t, err)
	}()
	require.NoError(t, err)
}

func TestStartFailure(t *testing.T) {
	persister := &testutil.MockPersister{}
	cfg := NewCounterOutputConfig("test")
	cfg.Path = "/a/path/to/a/nonexistent/file/hopefully"

	ctx := testutil.NewBuildContext(t)
	ops, err := cfg.Build(ctx)
	require.NoError(t, err)

	counterOutput := ops[0].(*CountOutput)

	err = counterOutput.Start(persister)
	defer func() {
		err := counterOutput.Stop()
		require.NoError(t, err)
	}()
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to write counter info to file")
}
