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
	newEntry.Body = "test"
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
		require.Equal(t, newEntry.Body, e.Body)
		require.Equal(t, newEntry.Severity, e.Severity)
		require.Equal(t, newEntry.SeverityText, e.SeverityText)
		require.Equal(t, newEntry.Attributes, e.Attributes)
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
	newEntry.Body = "test"
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
		require.Equal(t, newEntry.Body, e.Body)
		require.Equal(t, newEntry.Severity, e.Severity)
		require.Equal(t, newEntry.SeverityText, e.SeverityText)
		require.Equal(t, newEntry.Attributes, e.Attributes)
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
 openssl req -x509 -nodes -newkey rsa:2048 -keyout key.pem -out cert.pem -config san.cnf -days 9999
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
MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQCcRbjYcjAKDGcg
hgO8MrS/ClwNQkOq6fRhc/kyfmA4GMUh99+NGt93f30RF/Jk7DI6kJOJHWVgKY70
wwKfAZM3TDccXLDX3Lt8X7/bB2Uvx3vI1Dr+yowbwxJzihjqJrKCbzMtdqBeUvhM
g7/uzq1sRTxi3N3zmkG6VRCQYqIxd8gtG3iSuzTTALUSHuP3oFZs8DGIYswW6EvS
D0FDFff9OkvFoFpoz6gjTFpf+vsvQ/mWezX3SIQUxhJNJbwEYnSZBZGzXihh7HKo
0NKvZJtXSfP2ktplq9f/2aZSZjo+Rv2kTJh/HjFggNDZeOc8+9mpbeUV9Ogs8/g5
Uimnn8otAgMBAAECggEAIx8EWRjotQluj/+ujTh0KM9iOtSesqXb958B7Zg7dcAT
Zfv4cRUODiRH7nSMVKRE8aaWkeVaaE9OwrGlQCkxdecaJ7SpRgpk1KIMU2SJGEDk
EBGqpKLO9FpWJkNuMAm8atYlEV2s0yYgicm+dCRdE41H8gwjkeEkToVZsKmKPEWO
cYDL8bjO9uLajTi6OeML6zosqm+7PxjCz2LXGB3I7OKgUxwV3nUVDy8fWGdBN6wH
X7tDz3TbmOaxyBQGX4q3Gtsw5+8Htb63CfsLRj0Abtw2hMkC2GjYz+Z4gaG2FQmI
zFJm6LX5uhEo5dazjor4sLklPFfH5BBNSCtQPxaxnQKBgQDO9QsMsnXFSyvUM4Pn
BBUNzbJMI/bBqFTCPze+ibxsB0M7sT2s9ZY4kmkgUd2LWV3J3LEnY0befmXBBjHx
5RHtVGZXSWY96bjZrpiSvT9ZQuGURVkg/u7ve9U+VajJBi0oBYVdN7MuWOhkv+o3
bDtGTyvgIStBCpwysPbjVIWAlwKBgQDBTehQ/xvd1WwMRI49goe5WS274GisphcP
AKRDmSDMT08/rCnuFFOjCvvUIC4yy76cHaXGKF+ocSi9VjdFV2r4dV6g2n32PXtz
+rAD/rlvIgFr5L//ErmelSrCHyZkStOaHH087Z0zGpgt9sSqDuV4gTX5qutm7sDv
o0lHSnuf2wKBgBLxrUxBPbSMl/t5p7ZK0l6MGKkNlbXOYcvSG5kuZHgDBi19oOan
KFQPWt4hgEUULhifQfwYA1G0gj30AjhhPo3Z7vBIgLpkHY6Xg9HSzuytyZZX7rut
elOjozZsguG71gBW2QlaYuV4L/Wg96CRIK/j6WE/yATRItElD8RpZTsLAoGAInBa
33NT56XKZjUgklzbCW6V80771yaQHSAkI9b4PO40VEe8AKqma/nc++Hv2STrhKzT
iAZRZJUkiPb/Sd9VM4bVoRrMLj6t6+/RxCRxrRcF4c8TVcJkR5iT0ZnzIRMjt+Uz
etNqmlw2mJnKV/HneBytHRoSbnhC727L82OVutkCgYAb1/mrJ3umNl/uf4njCLhC
lcLqIedir4Gr/YwZfc4+Nh4QHpz0MZl25ZIyfD2W5YN8A/VhushAqMNB6gCynxh6
m/obkZlZgneE1sd5ruE6KgtH7+vV6OppBTGlFAvD3Fy0zumcjdSifurzVO/cdsgJ
Xj6r9TpVIMDm2xrDGbyldA==
-----END PRIVATE KEY-----
`)

var publicCrt = []byte(`
-----BEGIN CERTIFICATE-----
MIIDhjCCAm6gAwIBAgIJAOv4aa8eo/aOMA0GCSqGSIb3DQEBCwUAMHgxCzAJBgNV
BAYTAlhYMQwwCgYDVQQIDANOL0ExDDAKBgNVBAcMA04vQTEgMB4GA1UECgwXU2Vs
Zi1zaWduZWQgY2VydGlmaWNhdGUxKzApBgNVBAMMIjEyMC4wLjAuMTogU2VsZi1z
aWduZWQgY2VydGlmaWNhdGUwHhcNMjEwMjI1MTgwOTMyWhcNNDgwNzEyMTgwOTMy
WjB4MQswCQYDVQQGEwJYWDEMMAoGA1UECAwDTi9BMQwwCgYDVQQHDANOL0ExIDAe
BgNVBAoMF1NlbGYtc2lnbmVkIGNlcnRpZmljYXRlMSswKQYDVQQDDCIxMjAuMC4w
LjE6IFNlbGYtc2lnbmVkIGNlcnRpZmljYXRlMIIBIjANBgkqhkiG9w0BAQEFAAOC
AQ8AMIIBCgKCAQEAnEW42HIwCgxnIIYDvDK0vwpcDUJDqun0YXP5Mn5gOBjFIfff
jRrfd399ERfyZOwyOpCTiR1lYCmO9MMCnwGTN0w3HFyw19y7fF+/2wdlL8d7yNQ6
/sqMG8MSc4oY6iaygm8zLXagXlL4TIO/7s6tbEU8Ytzd85pBulUQkGKiMXfILRt4
krs00wC1Eh7j96BWbPAxiGLMFuhL0g9BQxX3/TpLxaBaaM+oI0xaX/r7L0P5lns1
90iEFMYSTSW8BGJ0mQWRs14oYexyqNDSr2SbV0nz9pLaZavX/9mmUmY6Pkb9pEyY
fx4xYIDQ2XjnPPvZqW3lFfToLPP4OVIpp5/KLQIDAQABoxMwETAPBgNVHREECDAG
hwR/AAABMA0GCSqGSIb3DQEBCwUAA4IBAQBCH9P06wfRRT2ok5QVCBdAskBc/Xvn
H9AL8qkCHNgv5+gXa4a7FAP1KRMf7mUcQQpv9qDhZlFCc7Gc/NGTvnmyOuhaTKYU
bKSdIqFzJp0qqhM6BiGqcRC8t3dLWE/Om1+lVfwT9f0Iq/14mHwZYxzMKbo+WteF
E7TANU6C5Fog0/YlH7fQrRY0m/ZIK70TKWlDzAyXIFI9YFgfo0Z0Peynw/iDFtaC
Ta4ai0P8ztyvlPjahgfz4BMdfNXyridGkWV6PQyBA8Wbng+11HnPW42JtOCYgTtV
2P/HjYMIRJDkCitvNpjPIAIHSJLhMLGduyZziKwZjyWfiaxP0BW226tl
-----END CERTIFICATE-----
`)
