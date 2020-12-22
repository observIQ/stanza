package file

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewFingerprint(t *testing.T) {

	cases := []struct {
		name            string
		fingerprintSize int
		fileSize        int
		expectedLen     int
	}{
		{
			name:            "halfFingerprint",
			fingerprintSize: defaultFingerprintSize,
			fileSize:        defaultFingerprintSize / 2,
			expectedLen:     defaultFingerprintSize / 2,
		},
		// {
		// 	name:            "default",
		// 	fingerprintSize: defaultFingerprintSize,
		// 	fileSize:        defaultFingerprintSize,
		// 	expectedLen:     defaultFingerprintSize,
		// 	expectedCap:     defaultFingerprintSize,
		// },
		// {
		// 	name:            "negative",
		// 	fingerprintSize: -1,
		// 	expectedSize:    minFingerprintSize,
		// },
		// {
		// 	name:            "zero",
		// 	fingerprintSize: 0,
		// 	expectedSize:    defaultFingerprintSize,
		// },
		// {
		// 	name:            "one",
		// 	fingerprintSize: 1,
		// 	expectedSize:    0, // less than min
		// },
		// {
		// 	name:            "min",
		// 	fingerprintSize: minFingerprintSize,
		// 	expectedSize:    minFingerprintSize,
		// },
		// {
		// 	name:            "large",
		// 	fingerprintSize: 100000,
		// 	expectedSize:    100000,
		// },
	}

	for _, tc := range cases {

		t.Run(tc.name, func(t *testing.T) {
			f, _, tempDir := newTestFileOperator(t, nil, nil)

			// Create a new file
			temp := openTemp(t, tempDir)
			writeString(t, temp, stringWithLength(tc.fileSize))
			info, err := temp.Stat()
			require.NoError(t, err)
			require.Equal(t, tc.fileSize, int(info.Size()))

			fp, err := f.NewFingerprint(temp)
			require.NoError(t, err)

			require.Equal(t, tc.expectedLen, len(fp.FirstBytes))
		})
	}
}
