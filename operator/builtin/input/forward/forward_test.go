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

	_, err = http.Post(fmt.Sprintf("http://127.0.0.1:%s", port), "application/json", &buf)
	require.NoError(t, err)

	select {
	case <-time.After(time.Second):
		require.FailNow(t, "Timed out waiting for entry to be received")
	case e := <-fake.Received:
		require.True(t, newEntry.Timestamp.Equal(e.Timestamp))
		require.Equal(t, newEntry.Record, e.Record)
		require.Equal(t, newEntry.Severity, e.Severity)
		require.Equal(t, newEntry.SeverityText, e.SeverityText)
		require.Equal(t, newEntry.Labels, e.Labels)
		require.Equal(t, newEntry.Resource, e.Resource)
	}
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

	_, err = client.Post(fmt.Sprintf("https://127.0.0.1:%s", port), "application/json", &buf)
	require.NoError(t, err)

	select {
	case <-time.After(time.Second):
		require.FailNow(t, "Timed out waiting for entry to be received")
	case e := <-fake.Received:
		require.True(t, newEntry.Timestamp.Equal(e.Timestamp))
		require.Equal(t, newEntry.Record, e.Record)
		require.Equal(t, newEntry.Severity, e.Severity)
		require.Equal(t, newEntry.SeverityText, e.SeverityText)
		require.Equal(t, newEntry.Labels, e.Labels)
		require.Equal(t, newEntry.Resource, e.Resource)
	}
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

/*
 openssl req -x509 -nodes -newkey rsa:2048 -keyout key.pem -out cert.pem -config san.cnf
 Generated with the following san.cnf:

 [req]
 default_bits  = 2048
 distinguished_name = req_distinguished_name
 req_extensions = req_ext
 x509_extensions = v3_req
 prompt = no

 [req_distinguished_name]
 countryName = XX
 stateOrProvinceName = N/A
 localityName = N/A
 organizationName = Self-signed certificate
 commonName = 120.0.0.1: Self-signed certificate

 [req_ext]
 subjectAltName = @alt_names

 [v3_req]
 subjectAltName = @alt_names

 [alt_names]
 IP.1 = 127.0.0.1
*/
var privateKey = []byte(`
-----BEGIN PRIVATE KEY-----
MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQDCvxfVAUCSatIe
+UXJLZnrCNrcstDsU0Ca5uWFS75jL+OBHU90vK5vI7/buhNcV18LDSGLgYc9JUiq
Hk6AtT5lJgYcMu9lsj2Rziy8dF1s6/k2PfeGPTtphfmcWpCgSCMgt83wqABr/ild
w49HoH5JDrItomCOt+t2B1oHA+xP1dFIYyJyIZHLYWOhs44Lw9Klx/UGGQvJzRk/
Vc4OfEtYcIUyYmZlJO+V96d/ymqvza4pG8BgCfoljV3TqmYUUdoZE4hJtJ8n9aK9
0efMTpqFm5CTi/ck+GNGOV9YndST9VMJiwBzdTQTYRa6ZTzz5rzxVdhzL4GRh0xM
s/GpZTLPAgMBAAECggEAL30DpbhZc5rCxDTK1KTfDJYrMHgWRBqE/YDiZR+0PGGY
G4r3LiM4cfeIuF7mi7TugzZfgLJENR/bWUhsoiwQHAAqq0OsZuMQ6nYZKJdDlOTx
700rB7v0ueWmmX7oF32fu0G24UFGYQ8oLSobzT6QrOX9gu0+mG625yAhzuYhANJZ
AgVnOhztogfvtBVOjrX2u/sI+XzdllCyDWY0yO3SAZ8w9NAZJcvTtWoJPrDMdT6s
e6CN1TSoT87be3d5+I9jJa9li45hXyftw+kRZvi6xeeq7Gd1InCS2rl/d58Drukq
PbeAPMNXRQ6WsipPzYnA5cKsT/p9x4SWcLf97A5eIQKBgQDf31wAYXQwwGs48EQI
Yh9IBW9FW+sAKA6SXBAz0VfLY4wvmpoOwq+MYpeCzO5eaheEPBx/r8haZFZ1RpbK
7Y87hU99EXNQU2KxLIPOHZWCfI9sjcYHnAgMLtzHEc3eWaSqHAi2ZJjbEaF6S8Ic
VywHxabsDxak8Z1O55eiMmtisQKBgQDesbMX4oIJIY4o1aeL4LUxtKVvHaGrZxP7
hVDw5ksz7FnCssovZYGumS43r4LJndBoRFkk4iG4zRDby3SP6+aOB0UvfNFWC93p
9g0neWjb02BrZvRZpYV9FWbYC+L80BRQVBU3bHd1mCtb5T3vjGGJ8W3SKE0ep6lO
QQt67HBNfwKBgCmr4u0zNrSIbKz5lEBXO2llkZPAi1rJGgVGW8G5evUh/4sw5PJQ
bOrdw0QWr1wltWDo64kdCFdDDBDiZdk6JQo4Q1aNdACEtP8zwQkR2q2iT/Qt46mw
8pKJ+pCXkNGNsCf19e01hnpoqr0f8u7hjxGXSf3wxQ9I5jY0x7XqWrDRAoGAGFRh
tKJSgpzf4yY0f9u08BFEYbdjCk7gqAIQrcD7Rlj0FYli/XqhiGnD2uGZ8F0Ff963
vofWF1KQHPNFcNPdBHdW37FLTZNOU1lGZqGlCx800YcV+xVSWDAWZFQoIGa7UWte
RgfbpUVSt198PMehgGiYDvXINykqHhqNHojmXBECgYAOI2IpzMFA5+xJ2D50H+xg
4Ai+vrZobJz1dZa2yQgD/HNFaqsM1E/X/nR72ba4AVTBmHRrw/aaxSPZlqx6XsNk
9D/F8Us/2ReJ+zksByzZ6xwhE9K6Wh6jQPHOk6yX/xGpJOlCQGJ/4z8EXEQYyNUz
wdgYU6bFYZVX/g63N5UmYw==
-----END PRIVATE KEY-----
`)

var publicCrt = []byte(`
-----BEGIN CERTIFICATE-----
MIIDhjCCAm6gAwIBAgIJAK52IbFHtoXVMA0GCSqGSIb3DQEBCwUAMHgxCzAJBgNV
BAYTAlhYMQwwCgYDVQQIDANOL0ExDDAKBgNVBAcMA04vQTEgMB4GA1UECgwXU2Vs
Zi1zaWduZWQgY2VydGlmaWNhdGUxKzApBgNVBAMMIjEyMC4wLjAuMTogU2VsZi1z
aWduZWQgY2VydGlmaWNhdGUwHhcNMjAxMjIzMTcwODUwWhcNMjEwMTIyMTcwODUw
WjB4MQswCQYDVQQGEwJYWDEMMAoGA1UECAwDTi9BMQwwCgYDVQQHDANOL0ExIDAe
BgNVBAoMF1NlbGYtc2lnbmVkIGNlcnRpZmljYXRlMSswKQYDVQQDDCIxMjAuMC4w
LjE6IFNlbGYtc2lnbmVkIGNlcnRpZmljYXRlMIIBIjANBgkqhkiG9w0BAQEFAAOC
AQ8AMIIBCgKCAQEAwr8X1QFAkmrSHvlFyS2Z6wja3LLQ7FNAmublhUu+Yy/jgR1P
dLyubyO/27oTXFdfCw0hi4GHPSVIqh5OgLU+ZSYGHDLvZbI9kc4svHRdbOv5Nj33
hj07aYX5nFqQoEgjILfN8KgAa/4pXcOPR6B+SQ6yLaJgjrfrdgdaBwPsT9XRSGMi
ciGRy2FjobOOC8PSpcf1BhkLyc0ZP1XODnxLWHCFMmJmZSTvlfenf8pqr82uKRvA
YAn6JY1d06pmFFHaGROISbSfJ/WivdHnzE6ahZuQk4v3JPhjRjlfWJ3Uk/VTCYsA
c3U0E2EWumU88+a88VXYcy+BkYdMTLPxqWUyzwIDAQABoxMwETAPBgNVHREECDAG
hwR/AAABMA0GCSqGSIb3DQEBCwUAA4IBAQBj8rQ7EA6uCzWhHtGB3g1u9H6NUVFx
Nx8ogKMRMl5JpNC4tcoQhJQ9OAnvzlcSMmP9+kMV705cNFMGizruKlVjTlEAqIbh
asKkecSq7Fjv/VkC1U5LaZ2/S7Qy6aeF4pTRqA2PJ8wBFI3h2dp652i67lDn+lAB
0JDJAeKagH5rPZtVy8g/KOrUfQcsDtKql5nlqJr3OfQ/S/YYKwVQwb98se4tQ12n
ju1QszhCwf8WPOUIGp+UutyggeneARhSp29j7pNm8tqfIJ46vEpQX817S/E8Oj3e
72nGD3oDMpjtks724qPQi9Wz+wy8tQEHJD+BAQ3Bakh2fZsvpQdgxgH7
-----END CERTIFICATE-----
`)
