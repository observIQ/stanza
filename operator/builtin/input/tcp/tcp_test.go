package tcp

import (
	"os"
	"net"
	"testing"
	"time"
	"crypto/tls"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const testTLSPrivateKey = `
-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDjKFqtAaZ/Uj53
Wk2r0xfH9IQqxCjH9gYFI+Kblf8Jvk2sZrQZbLrdjdHsZz/rLrt1YvvBSZ5LtkzK
P99wEi6goESCL0FmYP6Jkg0AKOrnfs8AGX/6PyQPfHBeK/767YV47ug0wJT2/92U
1K2rHIb154rPLp3l1kZyUqIj9MNphjW1jZ62mo2Jp4HkcivjR/cN8jz5UjQHBvO7
KMbhWDc0GLukxoctw/JmigqIrEFqfGcjANTzQZjwqIdscHkVEY8OLzQn+vIiXEF3
9VqLIDRjp3vExWXyVLcLZ1T0rQ84ICE1rBmeHfkteZlYPnuM92kmz/erXOCoDreY
1nwyp1YbAgMBAAECggEAG7RbQsh1vweP2MyptnAbcWawC+s6shCecVgMFj+4CD8u
h/1Kr+Mj80uNs9Bv6kYb1NhKritFZCSKvwwFO0zsZOjHEj2jM1JXGP44GbHj3HIJ
2xBBHItA4aaiqcmh4oa/hZ1VssFeKbXRF4rM15a2Gx2vP0+HMHXux5iub8Y1YxQh
eieSgLlMB4KXrqoosPCPZwSPrh7OzmapUPCeBFnVGUcy4UCRm3HL8RkSsiGJCRk5
mlevdKCy1YEGAojSAJrQOD6vrIXfoB+W0HUfkMdqvCfOQ00szTG9OTKK+tgBc55P
iI9IyNu6J7JxXC6iF/Z5CuXxiHztWz1drios/zKT2QKBgQD1NKqLtAgo1TsGnopK
3I6OTkCFVwZvw90wE0KhtHFAneIe6E2Q8bS9I7fFFlDVjRe4YvuyMYxoRdAlIp+r
qgl/18GZh2xuYcE081EbXSDdSu0yh5AGvZ5QRZO5DwngJZKbFYYPy/7UPO5m2WdV
Sa3TJMVzU7Fndfg1PVQL24snFQKBgQDtKEvAuHclOz3oGJdZYmmVmf+WTZDzFKS6
ms06kjYvqDxO5MgLJgLdaVnBpRUEttrwjKt+7F058vHk0RNOs5zmNwH13koIs2c6
w93ttBltNanoB9X1BWv8qntuHdjad2qsLdSUf2B7JT4i0FHnb9H+P0+m0qSHQCg5
KAuLriTUbwKBgEuDr64cgJLKsEXml2JcsE5lDPvDhEjxQfInTFLudh5XQScRlam4
tle1Y0gACl7p988iNK95EOuf7G0zT4cXc5t6f7XffeY0lsLO2ECcGp3sEEaKdzGM
PfAsrUTFu93a1F6Mb1/4C/+i0Cy+cVNTwIORBHny4WSicRE8VODd+OnNAoGBAK/7
zvrb584BABdS6Dy0ApW5CSiHtqArGXI/nTtxdDQ5K0eADdH4CvgyTSCdV9N/vUfz
mu88hpGR7l5Vp3YnYq6S8yl4IogCWQAKiIzzsEqSH9rGtcZ0l4WPHLjB/UFgjA/o
km7/dqDrKgi7fYu4NqPsZzbr6JtUyIRhau/j8gCRAoGBAPBptqrwdz39Sx7L1i29
nIEssRVQ8XKJoCwcVCtUDYCRtK6SNkac9I712ShW9MiSkwk2YrVGW/tZbyK/wMd0
cFseuHGPmUhW273or666QdFttgPhvtpy0ttMO9cp0px8SzT6ZNlFWHYtoh07fJWC
Zd4aQ9iUbXs0rMIV+0EMrxRf
-----END PRIVATE KEY-----
`

const testTLSCertificate = `
-----BEGIN CERTIFICATE-----
MIIDVDCCAjwCCQDA9fUVDYKppDANBgkqhkiG9w0BAQsFADBsMQswCQYDVQQGEwJV
UzERMA8GA1UECAwITWljaGlnYW4xFTATBgNVBAcMDEdyYW5kIFJhcGlkczERMA8G
A1UECgwIb2JzZXJ2SVExDzANBgNVBAsMBmdpdGh1YjEPMA0GA1UEAwwGc3Rhbnph
MB4XDTIxMDIyNDE0NTQ0OVoXDTIxMDMyNjE0NTQ0OVowbDELMAkGA1UEBhMCVVMx
ETAPBgNVBAgMCE1pY2hpZ2FuMRUwEwYDVQQHDAxHcmFuZCBSYXBpZHMxETAPBgNV
BAoMCG9ic2VydklRMQ8wDQYDVQQLDAZnaXRodWIxDzANBgNVBAMMBnN0YW56YTCC
ASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAOMoWq0Bpn9SPndaTavTF8f0
hCrEKMf2BgUj4puV/wm+TaxmtBlsut2N0exnP+suu3Vi+8FJnku2TMo/33ASLqCg
RIIvQWZg/omSDQAo6ud+zwAZf/o/JA98cF4r/vrthXju6DTAlPb/3ZTUraschvXn
is8uneXWRnJSoiP0w2mGNbWNnraajYmngeRyK+NH9w3yPPlSNAcG87soxuFYNzQY
u6TGhy3D8maKCoisQWp8ZyMA1PNBmPCoh2xweRURjw4vNCf68iJcQXf1WosgNGOn
e8TFZfJUtwtnVPStDzggITWsGZ4d+S15mVg+e4z3aSbP96tc4KgOt5jWfDKnVhsC
AwEAATANBgkqhkiG9w0BAQsFAAOCAQEAJRGMTrn7d4xFmQNzpApSSae3fkxVgV9Y
MytgjowvLV9vYarM0Pc/u64SMcx5z3wfMIkbOtF/dPZDzR3bt26Dr1rGBfx97grG
esKfxurrxdqxMiqTRj8MO7mKPa9NwO0M1BR4T29jnoKVcjy8zSlWO0ROAtZmbM74
ez+cfG6859ZLaFZZwY2H0lE4GzFlmkA1FuoR2biyUzRuCH4hMGrHZeiS8KR5ltn2
C/soJcXCDxtHbbfeDKclyRIIpwsXxGfaWehysMcfZavzJ0ZZioeilwdAZK7PcLY8
Y3YVtmCDXFa0Hy0jPMN4UMSvPmxRbcVpGSoEx2qnfOqHGmjrKcJ1kA==
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
		cfg.TLS.PrivateKey  = "test.key"

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

func TestTcpInput(t *testing.T) {
	t.Run("Simple", tcpInputTest([]byte("message\n"), []string{"message"}))
	t.Run("CarriageReturn", tcpInputTest([]byte("message\r\n"), []string{"message"}))
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
