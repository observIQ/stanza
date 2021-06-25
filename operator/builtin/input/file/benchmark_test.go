package file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/testutil"
)

type fileInputBenchmark struct {
	name   string
	config *InputConfig
}

func BenchmarkFileInput(b *testing.B) {
	cases := []fileInputBenchmark{
		{
			"Default",
			NewInputConfig("test_id"),
		},
		{
			"NoFileName",
			func() *InputConfig {
				cfg := NewInputConfig("test_id")
				cfg.IncludeFileName = false
				return cfg
			}(),
		},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			tempDir := testutil.NewTempDir(b)
			path := filepath.Join(tempDir, "in.log")

			cfg := tc.config
			cfg.OutputIDs = []string{"fake"}
			cfg.Include = []string{path}
			cfg.StartAt = "beginning"

			ops, err := cfg.Build(testutil.NewBuildContext(b))
			require.NoError(b, err)
			op := ops[0]

			fakeOutput := testutil.NewFakeOutput(b)
			err = op.SetOutputs([]operator.Operator{fakeOutput})
			require.NoError(b, err)

			err = op.Start()
			defer op.Stop()
			require.NoError(b, err)

			file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
			require.NoError(b, err)

			for i := 0; i < b.N; i++ {
				file.WriteString("testlog\n")
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				<-fakeOutput.Received
			}
		})
	}
}
