// Copyright 2020 OpenTelemetry Authors
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

//go:build tools
// +build tools

package tools

// This file follows the recommendation at
// https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
// on how to pin tooling dependencies to a go.mod file.
// This ensures that all systems use the same version of tools in addition to regular dependencies.

import (
	_ "github.com/goreleaser/goreleaser"
	_ "github.com/mgechev/revive"
	_ "github.com/observiq/amazon-log-agent-benchmark-tool/cmd/logbench"
	_ "github.com/securego/gosec/v2/cmd/gosec"
	_ "github.com/uw-labs/lichen"
	_ "github.com/vektra/mockery"
	_ "golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment"
)
