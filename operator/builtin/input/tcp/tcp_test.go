package tcp

import (
	"crypto/tls"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const testTLSPrivateKey = `
-----BEGIN PRIVATE KEY-----
MIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQDdNdVRHDoOlwrQ
YNlzP6MdLEIvN03Pv3A/Cdyy8LgKgSEf3kmw8o/75tSQzIAR6v7ts/qq1iAwE3OL
s4r8lASj2wirF2fNxX12OvIP8g3mrs4tCANBh413IywVKcEOrry71/s1k7+hscMv
Fe3NLxD1mNKJogwKyifvSc15zx8ge8SLjp875NiLCni2YYWXBt1pqd4wCol8lX6v
3u2rbNXrQf2sLncD0CE45EWHnzLzK33a0BwxyTXAOdd9kindL2IFct9C2HRQEk5h
GaXbNN0f6EMOZOzadJHfMledKVJ1XOd+t/kaPzY4NLDaGad04pNa+jph54qIVL5b
gCTOivX1AgMBAAECggEBAKPll/hxrn5S4LtFlrdyJfueaCctlaRgFd1PBEs8WU/H
HvDKtNS6031zKHlkW1trPpiF6iqbXdvg/ZI7Y7YCQXHZ/pEtVUa7lVp9EA5KbIxH
ZhEtR6RMt77Wu3mupxCm3MVcoA6xOqGl4JTJbZjBz5H4Ob2p57wyzeXYS7p9gHWC
fSj8tEqJdjLt7lqtqaWg/3iqqnLPdT3fGL6uyVbCDn9VZ23C7+sHiUfG67xHiF97
UT+O+dfADMY6rLY1njxdD0QGPS7MQLHAgL/ESjROSL4cj1f9VYJFgweAE/UxnDVQ
n3pTzHFItjYWtK75o7Yc/zaHKp5hsXMsiVb9gtmBcaECgYEA+i2viVdZQqItIDiJ
rc7M42Fo6mLv1gToOVaIst7qPmW6BlwSQbX/x2V/2UsMWtcL95mrmRVjK9iH/Pg8
ZaMlJynpgTM/x0jlZ2gZW1DPJWiCJ97xsdbOBA4JiGExc7odkbZhecfdlf66h0N6
Ll32k80PNqTDJV8wWuUxsEnJaLkCgYEA4luVgtnhiJx3FIfBM9p/EVearFsQFSil
PPeoJfc5GMGAnNeGBv5YI4wZ5Jaa0qHLg5ps5Y8vO1yWKiAuhgVKXhytOj86XsoL
MdisDYcxzskG/9ipX3fP1rBNgwdzBoP4QcpzV69weDsja8AU2pluKSd3r3nzwqsY
dc/NVJRsYR0CgYAw2scSrOoTZxQk3KWWOXItXRJd4yAuzRqER++97mYT9U2UfFpc
VqwyRhHnXw50ltYRbgLijBinsUstDVTODEPvF/IvdtCXnBagUOXSvT8WcQgpvRG5
xtbIV+1oooJDtS6dC96RJ4SQDARk8bpkX5kNV9gGtboeDC6nMWa4pFAekQKBgQCm
naM/3gEU/ZbplcOw13QQ39sKYz1DVdfLOMCcsY1lm4l/6WTOYQmfoNCuYe00fcO/
6zuc/fhWSaB/AZE9NUe4XoNkDIZ6n13+Iu8CRjFzdKWiTWjezOI/tSZY/HK+qQVj
6BFeydSPq3g3J/wxrB5aTKLcl3fGIwquLXeGenoMQQKBgQCWULypEeQwJsyKB57P
JzuCnFMvLL5qSNwot5c7I+AX5yi368dEurQl6pUUJ9VKNbpsUxFIMq9AHpddDoq/
+nIVt1DYr55ZsUJ6SgYtjvCMT9WOE/1Kqfh6p6y/mgRUl8m6v6gqi5/RfsNWJwfl
iBXhcGCQfkwZ8YIUyTW89qrwMw==
-----END PRIVATE KEY-----`

const testTLSCertificate = `
-----BEGIN CERTIFICATE-----
MIIDVDCCAjwCCQCwsE+LGRRtBTANBgkqhkiG9w0BAQsFADBsMQswCQYDVQQGEwJV
UzERMA8GA1UECAwITWljaGlnYW4xFTATBgNVBAcMDEdyYW5kIFJhcGlkczERMA8G
A1UECgwIb2JzZXJ2aVExDzANBgNVBAsMBlN0YW56YTEPMA0GA1UEAwwGU3Rhbnph
MB4XDTIxMDIyNTE3MzgxM1oXDTQ4MDcxMjE3MzgxM1owbDELMAkGA1UEBhMCVVMx
ETAPBgNVBAgMCE1pY2hpZ2FuMRUwEwYDVQQHDAxHcmFuZCBSYXBpZHMxETAPBgNV
BAoMCG9ic2VydmlRMQ8wDQYDVQQLDAZTdGFuemExDzANBgNVBAMMBlN0YW56YTCC
ASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAN011VEcOg6XCtBg2XM/ox0s
Qi83Tc+/cD8J3LLwuAqBIR/eSbDyj/vm1JDMgBHq/u2z+qrWIDATc4uzivyUBKPb
CKsXZ83FfXY68g/yDeauzi0IA0GHjXcjLBUpwQ6uvLvX+zWTv6Gxwy8V7c0vEPWY
0omiDArKJ+9JzXnPHyB7xIuOnzvk2IsKeLZhhZcG3Wmp3jAKiXyVfq/e7ats1etB
/awudwPQITjkRYefMvMrfdrQHDHJNcA5132SKd0vYgVy30LYdFASTmEZpds03R/o
Qw5k7Np0kd8yV50pUnVc5363+Ro/Njg0sNoZp3Tik1r6OmHniohUvluAJM6K9fUC
AwEAATANBgkqhkiG9w0BAQsFAAOCAQEA0u061goAXX7RxtdRO7Twz4zZIGS/oWvn
gj61zZIXt8LaTzRZFU9rs0rp7jPXKaszArJQc29anf1mWtRwQBAY0S0m4DkwoBln
7hMFf9MlisQvBVFjWgDo7QCJJmAxaPc1NZi8GQIANEMMZ+hLK17dhDB+6SdBbV4R
yx+7I3zcXQ+0H4Aym6KmvoIR3QAXsOYJ/43QzlYU63ryGYBAeg+JiD8fnr2W3QHb
BBdatHmcazlytT5KV+bANT/Ermw8y2tpWGWxMxQHveFh1zThYL8vkLi4fmZqqVCI
zv9WEy+9p05Aet+12x3dzRu93+yRIEYbSZ35NOUWfQ+gspF5rGgpxA==
-----END CERTIFICATE-----`

func tcpInputTest(input []byte, expected []string) func(t *testing.T) {
	return func(t *testing.T) {
		cfg := NewTCPInputConfig("test_id")
		cfg.ListenAddress = ":0"

		ops, err := cfg.Build(testutil.NewBuildContext(t))
		require.NoError(t, err)
		op := ops[0]

		mockOutput := testutil.Operator{}
		tcpInput := op.(*TCPInput)
		tcpInput.InputOperator.OutputOperators = []operator.Operator{&mockOutput}

		entryChan := make(chan *entry.Entry, 1)
		mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			entryChan <- args.Get(1).(*entry.Entry)
		}).Return(nil)

		err = tcpInput.Start()
		require.NoError(t, err)
		defer tcpInput.Stop()

		conn, err := net.Dial("tcp", tcpInput.listener.Addr().String())
		require.NoError(t, err)
		defer conn.Close()

		_, err = conn.Write(input)
		require.NoError(t, err)

		for _, expectedMessage := range expected {
			select {
			case entry := <-entryChan:
				require.Equal(t, expectedMessage, entry.Record)
			case <-time.After(time.Second):
				require.FailNow(t, "Timed out waiting for message to be written")
			}
		}

		select {
		case entry := <-entryChan:
			require.FailNow(t, "Unexpected entry: %s", entry)
		case <-time.After(100 * time.Millisecond):
			return
		}
	}
}

func tcpInputLabelsTest(input []byte, expected []string) func(t *testing.T) {
	return func(t *testing.T) {
		cfg := NewTCPInputConfig("test_id")
		cfg.ListenAddress = ":0"
		cfg.AddLabels = true

		ops, err := cfg.Build(testutil.NewBuildContext(t))
		require.NoError(t, err)
		op := ops[0]

		mockOutput := testutil.Operator{}
		tcpInput := op.(*TCPInput)
		tcpInput.InputOperator.OutputOperators = []operator.Operator{&mockOutput}

		entryChan := make(chan *entry.Entry, 1)
		mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			entryChan <- args.Get(1).(*entry.Entry)
		}).Return(nil)

		err = tcpInput.Start()
		require.NoError(t, err)
		defer tcpInput.Stop()

		conn, err := net.Dial("tcp", tcpInput.listener.Addr().String())
		require.NoError(t, err)
		defer conn.Close()

		_, err = conn.Write(input)
		require.NoError(t, err)

		for _, expectedMessage := range expected {
			select {
			case entry := <-entryChan:
				expectedLabels := map[string]string{
					"net.transport": "IP.TCP",
				}
				if addr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
					expectedLabels["net.host.ip"] = addr.IP.String()
					expectedLabels["net.host.port"] = strconv.FormatInt(int64(addr.Port), 10)
				}
				if addr, ok := conn.LocalAddr().(*net.TCPAddr); ok {
					expectedLabels["net.peer.ip"] = addr.IP.String()
					expectedLabels["net.peer.port"] = strconv.FormatInt(int64(addr.Port), 10)
				}
				require.Equal(t, expectedMessage, entry.Record)
				require.Equal(t, expectedLabels, entry.Labels)
			case <-time.After(time.Second):
				require.FailNow(t, "Timed out waiting for message to be written")
			}
		}

		select {
		case entry := <-entryChan:
			require.FailNow(t, "Unexpected entry: %s", entry)
		case <-time.After(100 * time.Millisecond):
			return
		}
	}
}

func tlsTCPInputTest(input []byte, expected []string) func(t *testing.T) {
	return func(t *testing.T) {

		f, err := os.Create("test.crt")
		require.NoError(t, err)
		defer f.Close()
		defer os.Remove("test.crt")
		_, err = f.WriteString(testTLSCertificate + "\n")
		require.NoError(t, err)
		f.Close()

		f, err = os.Create("test.key")
		require.NoError(t, err)
		defer f.Close()
		defer os.Remove("test.key")
		_, err = f.WriteString(testTLSPrivateKey + "\n")
		require.NoError(t, err)
		f.Close()

		cfg := NewTCPInputConfig("test_id")
		cfg.ListenAddress = ":0"
		cfg.TLS.Enable = true
		cfg.TLS.Certificate = "test.crt"
		cfg.TLS.PrivateKey = "test.key"

		ops, err := cfg.Build(testutil.NewBuildContext(t))
		require.NoError(t, err)
		op := ops[0]

		mockOutput := testutil.Operator{}
		tcpInput := op.(*TCPInput)
		tcpInput.InputOperator.OutputOperators = []operator.Operator{&mockOutput}

		entryChan := make(chan *entry.Entry, 1)
		mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			entryChan <- args.Get(1).(*entry.Entry)
		}).Return(nil)

		err = tcpInput.Start()
		require.NoError(t, err)
		defer tcpInput.Stop()

		conn, err := tls.Dial("tcp", tcpInput.listener.Addr().String(), &tls.Config{InsecureSkipVerify: true})
		require.NoError(t, err)
		defer conn.Close()

		_, err = conn.Write(input)
		require.NoError(t, err)

		for _, expectedMessage := range expected {
			select {
			case entry := <-entryChan:
				require.Equal(t, expectedMessage, entry.Record)
			case <-time.After(time.Second):
				require.FailNow(t, "Timed out waiting for message to be written")
			}
		}

		select {
		case entry := <-entryChan:
			require.FailNow(t, "Unexpected entry: %s", entry)
		case <-time.After(100 * time.Millisecond):
			return
		}
	}
}

func TestBuild(t *testing.T) {
	cases := []struct {
		name        string
		inputRecord TCPInputConfig
		expectErr   bool
	}{
		{
			"default-auto-address",
			TCPInputConfig{
				ListenAddress: ":0",
			},
			false,
		},
		{
			"default-fixed-address",
			TCPInputConfig{
				ListenAddress: "10.0.0.1:0",
			},
			false,
		},
		{
			"default-fixed-address-port",
			TCPInputConfig{
				ListenAddress: "10.0.0.1:9000",
			},
			false,
		},
		{
			"buffer-size-valid-default",
			TCPInputConfig{
				MaxBufferSize: 0,
				ListenAddress: "10.0.0.1:9000",
			},
			false,
		},
		{
			"buffer-size-valid-min",
			TCPInputConfig{
				MaxBufferSize: 65536,
				ListenAddress: "10.0.0.1:9000",
			},
			false,
		},
		{
			"buffer-size-negative",
			TCPInputConfig{
				MaxBufferSize: -1,
				ListenAddress: "10.0.0.1:9000",
			},
			true,
		},
		{
			"tls-disabled-with-keypair-set",
			TCPInputConfig{
				MaxBufferSize: 65536,
				ListenAddress: "10.0.0.1:9000",
				TLS: TLSConfig{
					Enable:      false,
					Certificate: "/tmp/cert",
					PrivateKey:  "/tmp/key",
				},
			},
			false,
		},
		{
			"tls-min-version-default",
			TCPInputConfig{
				MaxBufferSize: 65536,
				ListenAddress: "10.0.0.1:9000",
				TLS: TLSConfig{
					Enable:      false,
					Certificate: "/tmp/cert",
					PrivateKey:  "/tmp/key",
					MinVersion:  0,
				},
			},
			false,
		},
		{
			"tls-min-version-1.0",
			TCPInputConfig{
				MaxBufferSize: 65536,
				ListenAddress: "10.0.0.1:9000",
				TLS: TLSConfig{
					Enable:      false,
					Certificate: "/tmp/cert",
					PrivateKey:  "/tmp/key",
					MinVersion:  1.0,
				},
			},
			false,
		},
		{
			"tls-min-version-1.1",
			TCPInputConfig{
				MaxBufferSize: 65536,
				ListenAddress: "10.0.0.1:9000",
				TLS: TLSConfig{
					Enable:      false,
					Certificate: "/tmp/cert",
					PrivateKey:  "/tmp/key",
					MinVersion:  1.1,
				},
			},
			false,
		},
		{
			"tls-min-version-1.2",
			TCPInputConfig{
				MaxBufferSize: 65536,
				ListenAddress: "10.0.0.1:9000",
				TLS: TLSConfig{
					Enable:      false,
					Certificate: "/tmp/cert",
					PrivateKey:  "/tmp/key",
					MinVersion:  1.2,
				},
			},
			false,
		},
		{
			"tls-min-version-1.3",
			TCPInputConfig{
				MaxBufferSize: 65536,
				ListenAddress: "10.0.0.1:9000",
				TLS: TLSConfig{
					Enable:      false,
					Certificate: "/tmp/cert",
					PrivateKey:  "/tmp/key",
					MinVersion:  1.3,
				},
			},
			false,
		},
		{
			"tls-invalid-min-version-1.4",
			TCPInputConfig{
				MaxBufferSize: 65536,
				ListenAddress: "10.0.0.1:9000",
				TLS: TLSConfig{
					Enable:      false,
					Certificate: "/tmp/cert",
					PrivateKey:  "/tmp/key",
					MinVersion:  1.4,
				},
			},
			true,
		},
		{
			"tls-enabled-with-no-such-file-error",
			TCPInputConfig{
				MaxBufferSize: 65536,
				ListenAddress: "10.0.0.1:9000",
				TLS: TLSConfig{
					Enable:      true,
					Certificate: "/tmp/cert/missing",
					PrivateKey:  "/tmp/key/missing",
				},
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := NewTCPInputConfig("test_id")
			cfg.ListenAddress = tc.inputRecord.ListenAddress
			cfg.MaxBufferSize = tc.inputRecord.MaxBufferSize
			cfg.TLS = tc.inputRecord.TLS
			_, err := cfg.Build(testutil.NewBuildContext(t))
			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestTcpInput(t *testing.T) {
	t.Run("Simple", tcpInputTest([]byte("message\n"), []string{"message"}))
	t.Run("CarriageReturn", tcpInputTest([]byte("message\r\n"), []string{"message"}))
}

func TestTcpInputAattributes(t *testing.T) {
	t.Run("Simple", tcpInputLabelsTest([]byte("message\n"), []string{"message"}))
	t.Run("CarriageReturn", tcpInputLabelsTest([]byte("message\r\n"), []string{"message"}))
}

func TestTLSTcpInput(t *testing.T) {
	t.Run("Simple", tlsTCPInputTest([]byte("message\n"), []string{"message"}))
	t.Run("CarriageReturn", tlsTCPInputTest([]byte("message\r\n"), []string{"message"}))
}

func BenchmarkTcpInput(b *testing.B) {
	cfg := NewTCPInputConfig("test_id")
	cfg.ListenAddress = ":0"

	ops, err := cfg.Build(testutil.NewBuildContext(b))
	require.NoError(b, err)
	op := ops[0]

	fakeOutput := testutil.NewFakeOutput(b)
	tcpInput := op.(*TCPInput)
	tcpInput.InputOperator.OutputOperators = []operator.Operator{fakeOutput}

	err = tcpInput.Start()
	require.NoError(b, err)

	done := make(chan struct{})
	go func() {
		conn, err := net.Dial("tcp", tcpInput.listener.Addr().String())
		require.NoError(b, err)
		defer tcpInput.Stop()
		defer conn.Close()
		message := []byte("message\n")
		for {
			select {
			case <-done:
				return
			default:
				_, err := conn.Write(message)
				require.NoError(b, err)
			}
		}
	}()

	for i := 0; i < b.N; i++ {
		<-fakeOutput.Received
	}

	defer close(done)
}
