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
MIIEwAIBADANBgkqhkiG9w0BAQEFAASCBKowggSmAgEAAoIBAQDM4VDk2UICdib7
b+TNb7KVhRElr9392ecD7ICCYRJgycPr7y3r12JVr0nrq2+Cwp7jMaUiT1Ad9gOk
6U+uzS+T364eg3G7wi3E5oQ8CH/7YJKrrP59e//osMJRN4+NPvxDWs51RhLHIKb5
x2zgHfcohkvCk2TRYHG3yaRRjmoh/VYZkYPVHO7s7zR/7rpcv2i/df4By+CgKITl
tMfUtY5d+lyEqZnIJmsgT/iRTZI4g7QuyGCqmwfqy/wB3JWQ90Uaz11jX7ywp4wz
PVOWg6F6X0KbMK1wyo401qoPy7idM56En0L39d10vOmjb5cEzbnCtI2yCXVN9fkR
5e3N5Yc5AgMBAAECggEBAL/sEatPGdbUd4/yMZOAnvoRvQ5gwMOb7Bxw37FC3cRt
PWs2kv3qteMuYUCzR7JmPhD14ItTYOmwG5nQNSS6cWdEkgdjepc4P0fD6PuTus/w
l3TaiUtjbUa8zkrmkULvTcCKv/x7t/txSvmRJxyK9YywwSd0i2zXu68+5P7BOgq7
Z8itU636NCQhzG7hESoiPZHDV71AiKc5jOL+xbmO1RGcaoSHlrVBNnzJo8XmYCyD
4Fiq7ggPus0wcfC0D6dQizcb2RjJRRvCwuVdYYGNF76tF9MRI1kTvcb2lKURlrWc
LLlmE3QiByzEKiffa9pVyZDzGiqvc/T7w05OfJd6WAECgYEA+dwiQFbG50HOyzFM
5gm9ofuzsDghRNzriR7gxhkNW4DYi+ARA+sHMVsXrz6s0r46PwemAK3yNb59ecTa
2uD17U9Ki8yMY8SgXodarDdfjzfSncTtvQnimcqKRU4uaiMFXs7ti4yhj7DSUEEe
NUV5KTdtX/Y88/bezc9rxLdCL/ECgYEA0eo3CO8ZgdZm/1raI85sKSiAJVqLj0D+
c2r5d2PnHseTYvQMoRXu8GNXxn04CnUh+SMViXk7lyPJV3d+fe0NV38HljCrUynG
3fmtpoI9pw99RX7BelV2gRSdAhjsYsvmLs6AfbtNYnQBuMInIh+qWUXWcbqw9L7F
zeDLHJZPE8kCgYEA6YMLW7f+AmklXA9CQAdAfB+hioKazSHu2uLJzTnimu7q8qbB
IDlKKp1ooDZiDD8ObpO2WBI5OHNED0akB0WRcWzWTZsoZaGBA3dajXLe0xmntB00
1qRja7m3yhfMFxON1FJt/Sq8X28wzyJcmgrItnV/udyGkLba+dvtaxaeO/ECgYEA
iR397xcHyVj8lIaLAWKgIk5zTnMTwHKLA2d4JvWaDe/9pWCXM035cwrhViWLSsFy
fKPfOJp5Q2O77CeA9861rVar5P5Lmxop7ete8+oVTZ//izqeNUPIEc8eNDWFi492
/1Iien6zsMDoMwCXwWF/y6qjxkxVtLk8yhuxcS3534kCgYEAxt/mKYR6okqbZdLk
5uahQ6/cvKxYCcMp+D5Ob1OsWhfZUU/Y1OyJcF9y4pntUpWWOog74bE6nit47plx
17dSQ/8SqucN31nijHQY9gGCxEYfQrexMPnvwO0QN5NqiI+gheyK8phMe/CUB0E3
npwkFDpM7x5uAah3cknlJbWaPsk=
-----END PRIVATE KEY-----
`)

var publicCrt = []byte(`
-----BEGIN CERTIFICATE-----
MIIDhjCCAm6gAwIBAgIJAJDnIxyW8o35MA0GCSqGSIb3DQEBCwUAMHgxCzAJBgNV
BAYTAlhYMQwwCgYDVQQIDANOL0ExDDAKBgNVBAcMA04vQTEgMB4GA1UECgwXU2Vs
Zi1zaWduZWQgY2VydGlmaWNhdGUxKzApBgNVBAMMIjEyMC4wLjAuMTogU2VsZi1z
aWduZWQgY2VydGlmaWNhdGUwHhcNMjAxMTE5MTkwNTU3WhcNMjAxMjE5MTkwNTU3
WjB4MQswCQYDVQQGEwJYWDEMMAoGA1UECAwDTi9BMQwwCgYDVQQHDANOL0ExIDAe
BgNVBAoMF1NlbGYtc2lnbmVkIGNlcnRpZmljYXRlMSswKQYDVQQDDCIxMjAuMC4w
LjE6IFNlbGYtc2lnbmVkIGNlcnRpZmljYXRlMIIBIjANBgkqhkiG9w0BAQEFAAOC
AQ8AMIIBCgKCAQEAzOFQ5NlCAnYm+2/kzW+ylYURJa/d/dnnA+yAgmESYMnD6+8t
69diVa9J66tvgsKe4zGlIk9QHfYDpOlPrs0vk9+uHoNxu8ItxOaEPAh/+2CSq6z+
fXv/6LDCUTePjT78Q1rOdUYSxyCm+cds4B33KIZLwpNk0WBxt8mkUY5qIf1WGZGD
1Rzu7O80f+66XL9ov3X+AcvgoCiE5bTH1LWOXfpchKmZyCZrIE/4kU2SOIO0Lshg
qpsH6sv8AdyVkPdFGs9dY1+8sKeMMz1TloOhel9CmzCtcMqONNaqD8u4nTOehJ9C
9/XddLzpo2+XBM25wrSNsgl1TfX5EeXtzeWHOQIDAQABoxMwETAPBgNVHREECDAG
hwR/AAABMA0GCSqGSIb3DQEBCwUAA4IBAQBBZgfLSX+FPdevk4vakdGMPPtDJcN7
OZzeSUcFF10lWiLu6TrD70Cp1iqn5f2pw8i77VHHjuix3uaiUG1g7EJVZiU94rdY
4k9n1YzbbBu0kZjPLV/y78oEWFaw6nm7sQmCnYD9h5DiEvEkUdzH1txMBedvtIG6
NKd1KurKAZDnAFH4x/pcq+G3AF740IrkgoGoTZm4W7fP2cxkx46m5iweNbm/Uf2B
fzpJr8wA9qtf+uOXulQwPZ5mZRAbaroa3iEuvaN2Zr0tKEp34iTdLjb0JIyjclh/
8nsHehEjG05lMiehmD8NQ1nClggVH9gIL7j2xj9HRWxrFwHuDEqAnA+Y
-----END CERTIFICATE-----
`)
