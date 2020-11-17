package forward

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

func TestForwardInput(t *testing.T) {
	cfg := NewForwardInputConfig("test")
	cfg.ListenAddress = "0.0.0.0:0"
	cfg.OutputIDs = []string{"fake"}

	ops, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)
	forwardInput := ops[0].(*ForwardInput)

	fake := testutil.NewFakeOutput(t)
	err = forwardInput.SetOutputs([]operator.Operator{fake})
	require.NoError(t, err)

	require.NoError(t, forwardInput.Start())
	defer forwardInput.Stop()

	newEntry := entry.New()
	newEntry.Record = "test"
	newEntry.Timestamp = newEntry.Timestamp.Round(time.Second)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	require.NoError(t, enc.Encode([]*entry.Entry{newEntry}))

	_, port, err := net.SplitHostPort(forwardInput.ln.Addr().String())
	require.NoError(t, err)

	_, err = http.Post(fmt.Sprintf("http://localhost:%s", port), "application/json", &buf)
	require.NoError(t, err)

	fake.ExpectEntry(t, newEntry)
}

func TestForwardInputTLS(t *testing.T) {
	certFile, keyFile := createCertFiles(t)

	cfg := NewForwardInputConfig("test")
	cfg.ListenAddress = "0.0.0.0:0"
	cfg.TLS = &TLSConfig{
		CertFile: certFile,
		KeyFile:  keyFile,
	}
	cfg.OutputIDs = []string{"fake"}

	ops, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)
	forwardInput := ops[0].(*ForwardInput)

	fake := testutil.NewFakeOutput(t)
	err = forwardInput.SetOutputs([]operator.Operator{fake})
	require.NoError(t, err)

	require.NoError(t, forwardInput.Start())
	defer forwardInput.Stop()

	newEntry := entry.New()
	newEntry.Record = "test"
	newEntry.Timestamp = newEntry.Timestamp.Round(time.Second)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	require.NoError(t, enc.Encode([]*entry.Entry{newEntry}))

	_, port, err := net.SplitHostPort(forwardInput.ln.Addr().String())
	require.NoError(t, err)

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(publicCrt)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: pool,
			},
		},
	}

	_, err = client.Post(fmt.Sprintf("https://localhost:%s", port), "application/json", &buf)
	require.NoError(t, err)

	fake.ExpectEntry(t, newEntry)
}

func createCertFiles(t *testing.T) (cert, key string) {
	tempDir := testutil.NewTempDir(t)

	certFile, err := os.Create(filepath.Join(tempDir, "cert"))
	require.NoError(t, err)
	_, err = certFile.Write(publicCrt)
	require.NoError(t, err)
	certFile.Close()

	keyFile, err := os.Create(filepath.Join(tempDir, "key"))
	require.NoError(t, err)
	_, err = keyFile.Write(privateKey)
	require.NoError(t, err)
	keyFile.Close()

	return certFile.Name(), keyFile.Name()
}

// openssl genrsa -out private.key 4096
var privateKey = []byte(`-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----`)

// openssl req -new -x509 -sha256 -days 1825 -key private.key -out public.crt
var publicCrt = []byte(`-----BEGIN CERTIFICATE-----
MIIEpDCCAowCCQCxsoq2/xOeGzANBgkqhkiG9w0BAQsFADAUMRIwEAYDVQQDDAls
b2NhbGhvc3QwHhcNMjAxMTE3MjEzNjQ1WhcNMjUxMTE2MjEzNjQ1WjAUMRIwEAYD
VQQDDAlsb2NhbGhvc3QwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQCm
4DGKwgM1h5zXdQIGsfbQ6RWixyBLtSfM0XaEZcoTPnBEByVodJV/5pMmhvXUHPd3
ojW6UNjWqyKbwMIihUdpkcYK6FD5yliSAvVMoaH+KZVs0ANJRfbrebG68GqWr8YA
w+5DYHEWtpCt/+1mUX1bEgWoKeA+KhqWV3LR96mQY7qyrFRfinXZLpEdo3eHHPtY
d6DJNXk1S95smPWNkXwjOU1CuF24xZcetKkeOo0eIdt573JTNMqB8FfaPO6uDaV7
LDuHrndaGJfjo9CvLhbLGqpHrIeFu2HYr6QjRd4RcamN849RI6GQBtbVTrzPdjTV
UaKYf1NU9WpcUR1Vi1Yy0H4Fa62Jh6EOqA1r9sHAdbx/OcYg2EnQQj+0xegC7QIN
ZiR3eDnxYGIhXpbfMZ9w2AXyamPcUm0gWMLFnPRDvORMWFU4TQTz+W2UCcuwNDZT
JDo7A6rYC+8tBhwhIPxiudMIaUVECUuf6ooTPDeXhzWTRcHgTN47imsBp26jbS5W
BEGX/Hdz5a/3Rz3EJ39RJXemhbPaVNdNW/ehqswB1CVD7zxkxPs45qCTDDGARby+
PjEgzFAKPwUVhmtmc9hFSripTXt/1KNl9z4ToIt+oDNSIHFk79luudqb/wZXErSH
xJqJe6bxRAqsmFa62tDqYHdEUfCN8bNIXERtdIDlfQIDAQABMA0GCSqGSIb3DQEB
CwUAA4ICAQAlQD+zX6whPyGujwzV+yx7GHlXpEJqVjS7SoVRiSdugtpmvTkVuG3a
nejRUgntdlBFofJj5btwm6f7Eu2NvE0d2vopUZlkAnjtgVRFlpNguVVpuNyRMS5E
uM+cqIjyJdhDGtU7iXZZ6xzL4alBhfVdRNRqdXzHvnKHjc5tSFAZN9/6ldRnmDxM
nbaiPTFKxc45iDuXyY1Mh5V7bFWCS0GZVke1czgWtVRXPE1JN6ycv5YPbnT5o1D6
9lWUIRTw36VxNfyGOsWQxtc9TJiRXru9VspE0y+VBjuVQdnKbdGk39RTfpZxNEgg
X5hldAvExnW7dxZeFiXLGACXLhWoYqirFO8RAuvv367SJmXG1oOvt81VP2OZrpeJ
p84unsHw4N1pVo7v25Qz60ECB/nLbIfCcWozlHw7pJnMeRvvsTB8O84Gq9OMytAF
6/ZH0NiHgC6UrwZLoUtd7gty6sInqzqX4LMHqV0boeS+pj2Guh4/eGWE9lMTDb1P
dyupkNAIm2h4hhoiWqae/BHn07S4k9PfMcog8YpmUlSvr1BGmMhIrbGEbw7M/qSh
KwtTvM/pR0uc/2ifU4afsSzP8n7BgvhqG3AS86pFjxcUUQaTqycBHJGadOuxdGWO
iFamIylTKTHaJyP0fcxpIK5x5/vsxC01UKqiU7lDUDTg2Ie2LSJK9g==
-----END CERTIFICATE-----`)
