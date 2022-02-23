package helper

import (
	"github.com/open-telemetry/opentelemetry-log-collection/operator"
	otelhelper "github.com/open-telemetry/opentelemetry-log-collection/operator/helper"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
)

var additionalEncodingOverrides = map[string]encoding.Encoding{
	"shift-jis": japanese.ShiftJIS,
}

// StanzaEncodingConfig wraps OTel EncodingConfig to add additional encodings
type StanzaEncodingConfig struct {
	otelhelper.EncodingConfig `mapstructure:",squash,omitempty" json:",inline,omitempty" yaml:",inline,omitempty"`
}

// NewEncodingConfig creates a new Stanza Encoding config
func NewEncodingConfig() StanzaEncodingConfig {
	return StanzaEncodingConfig{
		EncodingConfig: otelhelper.NewEncodingConfig(),
	}

}

// Build will build an Encoding operator
func (s StanzaEncodingConfig) Build(context operator.BuildContext) (otelhelper.Encoding, error) {
	enc, ok := additionalEncodingOverrides[s.Encoding]
	if ok {
		return otelhelper.Encoding{
			Encoding: enc,
		}, nil
	}

	return s.EncodingConfig.Build(context)
}
