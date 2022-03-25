package otlp

import (
	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator/helper"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestOtlpOperator(t *testing.T) {
	cfg := OtlpConfig{
		OutputConfig: helper.OutputConfig{
			BasicConfig: helper.BasicConfig{
				OperatorID:   "test_operator_id",
				OperatorType: "otlp",
			}},
		Endpoint:      "test:80",
		Insecure:      "",
		Headers:       Headers{Authorization: "test"},
		RetrySettings: RetrySettings{Enabled: true},
		Timeout:       5,
	}

	ops, err := cfg.Build(testutil.NewBuildContext(t))
	require.NotNil(t, ops)
	require.NoError(t, err)

	//op := ops[0]

	entry := entry.New()
	entry.Timestamp = time.Now()
	entry.Resource = map[string]string{"test": "test"}
	entry.Record = "test message"

	//TODO mock logsClient.Export call
	//err = op.(*OtlpOutput).Process(context.Background(), entry)
	require.NoError(t, err)
}
