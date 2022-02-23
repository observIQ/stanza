// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helper

import (
	"fmt"
	"strings"

	"golang.org/x/text/encoding/japanese"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"

	"github.com/observiq/stanza/operator"
)

// NewEncodingConfig creates a new Encoding config
func NewEncodingConfig() EncodingConfig {
	return EncodingConfig{
		Encoding: "nop",
	}
}

// EncodingConfig is the configuration of a Encoding helper
type EncodingConfig struct {
	Encoding string `mapstructure:"encoding,omitempty"              json:"encoding,omitempty"             yaml:"encoding,omitempty"`
}

// Build will build an Encoding operator.
func (c EncodingConfig) Build(_ operator.BuildContext) (Encoding, error) {
	enc, err := lookupEncoding(c.Encoding)
	if err != nil {
		return Encoding{}, err
	}

	return Encoding{
		Encoding: enc,
	}, nil
}

// Encoding represents an text encoding
type Encoding struct {
	Encoding encoding.Encoding
}

// Decode converts the bytes in msgBuf to utf-8 from the configured encoding
func (e *Encoding) Decode(msgBuf []byte) (string, error) {
	decodeBuffer := make([]byte, 1<<12)
	decoder := e.Encoding.NewDecoder()

	for {
		decoder.Reset()
		nDst, _, err := decoder.Transform(decodeBuffer, msgBuf, true)
		if err == nil {
			return string(decodeBuffer[:nDst]), nil
		}
		if err == transform.ErrShortDst {
			decodeBuffer = make([]byte, len(decodeBuffer)*2)
			continue
		}
		return "", fmt.Errorf("transform encoding: %s", err)
	}
}

var encodingOverrides = map[string]encoding.Encoding{
	"utf-16":    unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM),
	"utf16":     unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM),
	"utf8":      unicode.UTF8,
	"ascii":     unicode.UTF8,
	"us-ascii":  unicode.UTF8,
	"shift-jis": japanese.ShiftJIS,
	"nop":       encoding.Nop,
	"":          encoding.Nop,
}

func lookupEncoding(enc string) (encoding.Encoding, error) {
	if encoding, ok := encodingOverrides[strings.ToLower(enc)]; ok {
		return encoding, nil
	}
	encoding, err := ianaindex.IANA.Encoding(enc)
	if err != nil {
		return nil, fmt.Errorf("unsupported encoding '%s'", enc)
	}
	if encoding == nil {
		return nil, fmt.Errorf("no charmap defined for encoding '%s'", enc)
	}
	return encoding, nil
}
