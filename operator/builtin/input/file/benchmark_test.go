package file

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/observiq/stanza/v2/operator"
	"github.com/observiq/stanza/v2/testutil"
)

type fileInputBenchmark struct {
	name   string
	paths  []string
	config func() *InputConfig
}

type benchFile struct {
	*os.File
	log func(int)
}

func simpleTextFile(file *os.File) *benchFile {
	line := stringWithLength(49) + "\n"
	return &benchFile{
		File: file,
		log:  func(_ int) { file.WriteString(line) },
	}
}

func BenchmarkFileInput(b *testing.B) {
	cases := []fileInputBenchmark{
		{
			name: "Single",
			paths: []string{
				"file0.log",
			},
			config: func() *InputConfig {
				cfg := NewInputConfig("test_id")
				cfg.Include = []string{
					"file0.log",
				}
				return cfg
			},
		},
		{
			name: "Glob",
			paths: []string{
				"file0.log",
				"file1.log",
				"file2.log",
				"file3.log",
			},
			config: func() *InputConfig {
				cfg := NewInputConfig("test_id")
				cfg.Include = []string{"file*.log"}
				return cfg
			},
		},
		{
			name: "MultiGlob",
			paths: []string{
				"file0.log",
				"file1.log",
				"log0.log",
				"log1.log",
			},
			config: func() *InputConfig {
				cfg := NewInputConfig("test_id")
				cfg.Include = []string{
					"file*.log",
					"log*.log",
				}
				return cfg
			},
		},
		{
			name: "MaxConcurrent",
			paths: []string{
				"file0.log",
				"file1.log",
				"file2.log",
				"file3.log",
			},
			config: func() *InputConfig {
				cfg := NewInputConfig("test_id")
				cfg.Include = []string{
					"file*.log",
				}
				cfg.MaxConcurrentFiles = 2
				return cfg
			},
		},
		{
			name: "FngrPrntLarge",
			paths: []string{
				"file0.log",
			},
			config: func() *InputConfig {
				cfg := NewInputConfig("test_id")
				cfg.Include = []string{
					"file*.log",
				}
				cfg.FingerprintSize = 10 * defaultFingerprintSize
				return cfg
			},
		},
		{
			name: "FngrPrntSmall",
			paths: []string{
				"file0.log",
			},
			config: func() *InputConfig {
				cfg := NewInputConfig("test_id")
				cfg.Include = []string{
					"file*.log",
				}
				cfg.FingerprintSize = defaultFingerprintSize / 10
				return cfg
			},
		},
	}

	for _, bench := range cases {
		b.Run(bench.name, func(b *testing.B) {
			rootDir, err := ioutil.TempDir("", "")
			require.NoError(b, err)

			files := []*benchFile{}
			for _, path := range bench.paths {
				file := openFile(b, filepath.Join(rootDir, path))
				files = append(files, simpleTextFile(file))
			}

			cfg := bench.config()
			cfg.OutputIDs = []string{"fake"}
			for i, inc := range cfg.Include {
				cfg.Include[i] = filepath.Join(rootDir, inc)
			}
			cfg.StartAt = "beginning"

			ops, err := cfg.Build(testutil.NewBuildContext(b))
			require.NoError(b, err)
			op := ops[0]

			fakeOutput := testutil.NewFakeOutput(b)
			err = op.SetOutputs([]operator.Operator{fakeOutput})
			require.NoError(b, err)

			// write half the lines before starting
			mid := b.N / 2
			for i := 0; i < mid; i++ {
				for _, file := range files {
					file.log(i)
				}
			}

			b.ResetTimer()
			err = op.Start()
			defer op.Stop()
			require.NoError(b, err)

			// write the remainder of lines while running
			go func() {
				for i := mid; i < b.N; i++ {
					for _, file := range files {
						file.log(i)
					}
				}
			}()

			for i := 0; i < b.N*len(files); i++ {
				<-fakeOutput.Received
			}
		})
	}
}
