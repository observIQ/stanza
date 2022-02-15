GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
GOFLAGS=-mod=mod

GIT_SHA=$(shell git rev-parse --short HEAD)
GIT_COMMIT=$(shell git rev-parse HEAD)
TAGS=-tags timetzdata

PROJECT_ROOT = $(shell pwd)
ARTIFACTS = ${PROJECT_ROOT}/artifacts
ALL_MODULES := $(shell find . -type f -name "go.mod" -exec dirname {} \; | sort )
FIELDALIGNMENT_DIRS := ./...

TOOLS_MOD_DIR := ./internal/tools
.PHONY: install-tools
install-tools:
	cd $(TOOLS_MOD_DIR) && go install github.com/golangci/golangci-lint/cmd/golangci-lint
	cd $(TOOLS_MOD_DIR) && go install github.com/vektra/mockery/cmd/mockery
	cd $(TOOLS_MOD_DIR) && go install github.com/uw-labs/lichen
	cd $(TOOLS_MOD_DIR) && go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment
	cd $(TOOLS_MOD_DIR) && go install github.com/observiq/amazon-log-agent-benchmark-tool/cmd/logbench
	cd $(TOOLS_MOD_DIR) && go install github.com/goreleaser/goreleaser
	cd $(TOOLS_MOD_DIR) && go install github.com/securego/gosec/v2/cmd/gosec@v2.8.1

.PHONY: scan-license
scan-license: build-all
	$$GOPATH/bin/lichen --config=./license.yaml "./artifacts/stanza_linux_amd64"
	$$GOPATH/bin/lichen --config=./license.yaml "./artifacts/stanza_windows_amd64"
	$$GOPATH/bin/lichen --config=./license.yaml "./artifacts/stanza_darwin_amd64"

.PHONY: test
test: vet test-only

.PHONY: test-only
test-only:
	$(MAKE) for-all CMD="go test -race -coverprofile coverage.txt -coverpkg ./... ./..."

.PHONY: test-integration
test-integration:
	mkdir -p artifacts
	curl -fL https://github.com/observiq/stanza-plugins/releases/latest/download/stanza-plugins.tar.gz -o ./artifacts/stanza-plugins.tar.gz
	docker build . -t stanza-integration:latest
	$(MAKE) for-all CMD="go clean -testcache ./... ./..."
	$(MAKE) for-all CMD="go test -tags integration ./... ./..."

.PHONY: bench
bench:
	go test -benchmem -run=^$$ -bench ^* ./...

.PHONY: clean
clean:
	rm -fr ./artifacts
	$(MAKE) for-all CMD="rm -f coverage.txt coverage.html"

.PHONY: tidy
tidy:
	$(MAKE) for-all CMD="rm -fr go.sum"
	$(MAKE) for-all CMD="go mod tidy"

.PHONY: listmod
listmod:
	@set -e; for dir in $(ALL_MODULES); do \
		(echo "$${dir}"); \
	done

.PHONY: lint
lint:
	$$GOPATH/bin/golangci-lint run --timeout 2m0s --allow-parallel-runners ./...

.PHONY: fieldalignment
fieldalignment:
	fieldalignment $(FIELDALIGNMENT_DIRS)

.PHONY: fieldalignment-fix
fieldalignment-fix:
	fieldalignment -fix $(FIELDALIGNMENT_DIRS)

.PHONY: vet
vet: check-missing-modules
	GOOS=darwin $(MAKE) for-all CMD="go vet ./..."
	GOOS=linux $(MAKE) for-all CMD="go vet ./..."
	GOOS=windows $(MAKE) for-all CMD="go vet ./..."

.PHONY: secure
secure:
	gosec ./...

.PHONY: check-missing-modules
check-missing-modules:
	@find ./operator/builtin -type f -name "go.mod" -exec dirname {} \; | cut -d'/' -f2- | while read mod ; do \
		grep $$mod ./cmd/stanza/init_*.go > /dev/null ;\
		if [ $$? -ne 0 ] ; then \
			echo Stanza is not building with module $$mod ;\
			exit 1 ;\
		fi \
	done

.PHONY: generate
generate:
	go generate ./...

.PHONY: build
build:
	(cd ./cmd/stanza && \
		CGO_ENABLED=0 \
		go build \
		-ldflags "-X github.com/observiq/stanza/version.GitTag=${GIT_TAG} -X github.com/observiq/stanza/version.GitCommit=${GIT_COMMIT}" \
		-o ../../artifacts/stanza_$(GOOS)_$(GOARCH) \
		$(TAGS) .)

.PHONY: install
install:
	(cd ./cmd/stanza && CGO_ENABLED=0 go install .)

.PHONY: build-all
build-all: build-darwin-amd64 build-darwin-arm64 build-linux-amd64 build-linux-arm64 build-windows-amd64

.PHONY: build-darwin-amd64
build-darwin-amd64:
	@GOOS=darwin GOARCH=amd64 $(MAKE) build

.PHONY: build-darwin-amd64
build-darwin-arm64:
	@GOOS=darwin GOARCH=arm64 $(MAKE) build

.PHONY: build-linux-amd64
build-linux-amd64:
	@GOOS=linux GOARCH=amd64 $(MAKE) build

.PHONY: build-linux-arm64
build-linux-arm64:
	@GOOS=linux GOARCH=arm64 $(MAKE) build

.PHONY: build-windows-amd64
build-windows-amd64:
	@GOOS=windows GOARCH=amd64 $(MAKE) build

.PHONY: release-test
release-test: install-tools
	goreleaser release --rm-dist --skip-publish --skip-announce --skip-validate

.PHONY: for-all
for-all:
	@set -e; for dir in $(ALL_MODULES); do \
	  (cd "$${dir}" && $${CMD} ); \
	done
