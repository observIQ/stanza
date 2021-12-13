package googlecloud

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/buffer"
	"github.com/observiq/stanza/operator/helper"
	"github.com/observiq/stanza/testutil"
	"github.com/observiq/stanza/version"
	"github.com/stretchr/testify/require"
)

func TestUserAgentVar(t *testing.T) {
	require.Equal(t, getUserAgent(), "StanzaLogAgent/"+version.GetVersion())
	userAgent = "TestAgent"
	require.Equal(t, getUserAgent(), "TestAgent/"+version.GetVersion())
}

func TestInitHook(t *testing.T) {
	builderFunc, ok := operator.DefaultRegistry.Lookup(operatorType)
	require.True(t, ok)

	config := builderFunc()
	_, ok = config.(*GoogleCloudOutputConfig)
	require.True(t, ok)
}

func TestDefaultConfig(t *testing.T) {
	operatorID := "test_id"
	config := NewGoogleCloudOutputConfig(operatorID)
	require.Equal(t, operatorType, config.OperatorType)
	require.Equal(t, operatorID, config.OperatorID)
	require.Equal(t, defaultUseCompression, config.UseCompression)
	require.Equal(t, defaultMaxEntrySize, int(config.MaxEntrySize))
	require.Equal(t, defaultMaxRequestSize, int(config.MaxRequestSize))
	require.Equal(t, defaultTimeout, config.Timeout.Duration)
}

func TestValidCredentialsFromField(t *testing.T) {
	json := `{"type": "service_account", "project_id": "test"}`
	config := GoogleCloudOutputConfig{
		Credentials: json,
	}

	credentials, err := config.getCredentials()
	require.NoError(t, err)
	require.Equal(t, credentials.JSON, []byte(json))
	require.Equal(t, credentials.ProjectID, "test")
}

func TestInvalidCredentialsFromField(t *testing.T) {
	config := GoogleCloudOutputConfig{
		Credentials: "invalid",
	}

	credentials, err := config.getCredentials()
	require.Nil(t, credentials)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse credentials field")
}

func TestValidCredentialsFromFile(t *testing.T) {
	file, err := ioutil.TempFile("", "credentials.json")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	json := `{"type": "service_account", "project_id": "test"}`
	_, err = file.WriteString(json)
	require.NoError(t, err)

	config := GoogleCloudOutputConfig{
		CredentialsFile: file.Name(),
	}

	credentials, err := config.getCredentials()
	require.NoError(t, err)
	require.Equal(t, credentials.JSON, []byte(json))
	require.Equal(t, credentials.ProjectID, "test")
}

func TestInvalidCredentialsFromFile(t *testing.T) {
	file, err := ioutil.TempFile("", "credentials.json")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	_, err = file.WriteString("invalid")
	require.NoError(t, err)

	config := GoogleCloudOutputConfig{
		CredentialsFile: file.Name(),
	}

	credentials, err := config.getCredentials()
	require.Nil(t, credentials)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse credentials in credentials file")
}

func TestCredentialsFromMissingFile(t *testing.T) {
	dir, err := ioutil.TempDir("", "config")
	require.NoError(t, err)
	defer os.Remove(dir)

	config := GoogleCloudOutputConfig{
		CredentialsFile: path.Join(dir, "config.json"),
	}

	credentials, err := config.getCredentials()
	require.Nil(t, credentials)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to read credentials file")
}

func TestGetCredentialsPrecedence(t *testing.T) {
	file, err := ioutil.TempFile("", "credentials.json")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	fieldContents := `{"type": "service_account", "project_id": "test_1"}`
	fileContents := `{"type": "service_account", "project_id": "test_2"}`

	_, err = file.WriteString(fileContents)
	require.NoError(t, err)

	config := GoogleCloudOutputConfig{
		Credentials:     fieldContents,
		CredentialsFile: file.Name(),
	}

	credentials, err := config.getCredentials()
	require.NoError(t, err)
	require.Equal(t, credentials.JSON, []byte(fieldContents))
	require.Equal(t, credentials.ProjectID, "test_1")
}

func TestValidDefaultCredentials(t *testing.T) {
	credentialsEnv := "GOOGLE_APPLICATION_CREDENTIALS"
	prevValue := os.Getenv(credentialsEnv)
	defer func() { _ = os.Setenv(credentialsEnv, prevValue) }()

	file, err := ioutil.TempFile("", "credentials.json")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	json := `{"type": "service_account", "project_id": "test"}`
	_, err = file.WriteString(json)
	require.NoError(t, err)

	err = os.Setenv(credentialsEnv, file.Name())
	require.NoError(t, err)

	config := GoogleCloudOutputConfig{}
	credentials, err := config.getCredentials()
	require.NoError(t, err)
	require.Equal(t, credentials.JSON, []byte(json))
	require.Equal(t, credentials.ProjectID, "test")
}

func TestInvalidDefaultCredentials(t *testing.T) {
	credentialsEnv := "GOOGLE_APPLICATION_CREDENTIALS"
	prevValue := os.Getenv(credentialsEnv)
	defer func() { _ = os.Setenv(credentialsEnv, prevValue) }()

	file, err := ioutil.TempFile("", "credentials.json")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	_, err = file.WriteString("invalid")
	require.NoError(t, err)

	err = os.Setenv(credentialsEnv, file.Name())
	require.NoError(t, err)

	config := GoogleCloudOutputConfig{}
	credentials, err := config.getCredentials()
	require.Nil(t, credentials)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to find default credentials")
}

func TestBuild(t *testing.T) {
	testCases := []struct {
		name   string
		config GoogleCloudOutputConfig
		err    string
	}{
		{
			name: "valid config",
			config: GoogleCloudOutputConfig{
				OutputConfig:   helper.NewOutputConfig("test", operatorType),
				Credentials:    `{"type": "service_account", "project_id": "test"}`,
				BufferConfig:   buffer.NewConfig(),
				UseCompression: true,
			},
		},
		{
			name: "invalid type",
			config: GoogleCloudOutputConfig{
				OutputConfig: helper.NewOutputConfig("test", ""),
				Credentials:  `{"type": "service_account", "project_id": "test"}`,
				BufferConfig: buffer.NewConfig(),
			},
			err: "failed to build output operator",
		},
		{
			name: "invalid buffer",
			config: GoogleCloudOutputConfig{
				OutputConfig: helper.NewOutputConfig("test", operatorType),
				Credentials:  `{"type": "service_account", "project_id": "test"}`,
				BufferConfig: buffer.Config{
					Builder: buffer.DiskBufferConfig{},
				},
			},
			err: "failed to build buffer",
		},
		{
			name: "invalid credentials",
			config: GoogleCloudOutputConfig{
				OutputConfig: helper.NewOutputConfig("test", operatorType),
				Credentials:  "invalid",
				BufferConfig: buffer.NewConfig(),
			},
			err: "failed to get credentials",
		},
		{
			name: "missing project id",
			config: GoogleCloudOutputConfig{
				OutputConfig: helper.NewOutputConfig("test", operatorType),
				Credentials:  `{"type": "service_account"}`,
				BufferConfig: buffer.NewConfig(),
			},
			err: "failed to get project id from config or credentials",
		},
		{
			name: "configured project id",
			config: GoogleCloudOutputConfig{
				OutputConfig: helper.NewOutputConfig("test", operatorType),
				Credentials:  `{"type": "service_account"}`,
				BufferConfig: buffer.NewConfig(),
				ProjectID:    "test",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			buildContext := testutil.NewBuildContext(t)
			operators, err := tc.config.Build(buildContext)
			if tc.err != "" {
				require.Nil(t, operators)
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, 1, len(operators))
				_, ok := operators[0].(*GoogleCloudOutput)
				require.True(t, ok)
			}
		})
	}
}
