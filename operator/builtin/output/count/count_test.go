package counter

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator/helper"
	"github.com/observiq/stanza/testutil"
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
	cfg := NewCounterOutputConfig("test")

	tmpFile, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	cfg.Path = tmpFile.Name()
	cfg.Duration = helper.NewDuration(2 * time.Second)

	ctx := testutil.NewBuildContext(t)
	ops, err := cfg.Build(ctx)
	require.NoError(t, err)

	counterOutput := ops[0].(*CountOutput)

	err = counterOutput.Start()
	require.NoError(t, err)
	defer func() {
		err := counterOutput.Stop()
		require.NoError(t, err)
	}()

	e := entry.New()
	err = counterOutput.Process(context.Background(), e)
	require.NoError(t, err)
	require.Equal(t, counterOutput.numEntries.Int64(), int64(1))

	counterOutput.logChan <- struct{}{}
	time.Sleep(500 * time.Millisecond)

	content, err := ioutil.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	var object countObject
	err = json.Unmarshal(content, &object)
	require.NoError(t, err)

	require.Equal(t, object.Entries.Int64(), int64(1))
	require.GreaterOrEqual(t, object.EntriesPerMinute, 0.0)
	require.GreaterOrEqual(t, object.ElapsedMinutes, 0.0)
}

func TestStartStdout(t *testing.T) {
	cfg := NewCounterOutputConfig("test")

	ctx := testutil.NewBuildContext(t)
	ops, err := cfg.Build(ctx)
	require.NoError(t, err)

	counterOutput := ops[0].(*CountOutput)

	err = counterOutput.Start()
	defer func() {
		err := counterOutput.Stop()
		require.NoError(t, err)
	}()
	require.NoError(t, err)
}

func TestStartFailure(t *testing.T) {
	cfg := NewCounterOutputConfig("test")
	cfg.Path = "/a/path/to/a/nonexistent/file/hopefully"

	ctx := testutil.NewBuildContext(t)
	ops, err := cfg.Build(ctx)
	require.NoError(t, err)

	counterOutput := ops[0].(*CountOutput)

	err = counterOutput.Start()
	defer func() {
		err := counterOutput.Stop()
		require.NoError(t, err)
	}()
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to write counter info to file")
}
