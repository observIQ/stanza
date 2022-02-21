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
	cd $(TOOLS_MOD_DIR) && go install github.com/vektra/mockery/cmd/mockery
	cd $(TOOLS_MOD_DIR) && go install github.com/uw-labs/lichen
	cd $(TOOLS_MOD_DIR) && go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment
	cd $(TOOLS_MOD_DIR) && go install github.com/observiq/amazon-log-agent-benchmark-tool/cmd/logbench
	cd $(TOOLS_MOD_DIR) && go install github.com/goreleaser/goreleaser
	cd $(TOOLS_MOD_DIR) && go install github.com/securego/gosec/v2/cmd/gosec@v2.8.1
	cd $(TOOLS_MOD_DIR) && go install github.com/mgechev/revive

.PHONY: scan-license
scan-license: build-all
	$$GOPATH/bin/lichen --config=./license.yaml "./artifacts/stanza_linux_amd64"
	$$GOPATH/bin/lichen --config=./license.yaml "./artifacts/stanza_windows_amd64"
	$$GOPATH/bin/lichen --config=./license.yaml "./artifacts/stanza_darwin_amd64"

.PHONY: test
test: 
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

.PHONY: build-linux
build-linux: build-linux-amd64 build-linux-arm64

.PHONY: build-darwin
build-darwin: build-darwin-amd64 build-darwin-arm64

.PHONY: build-windows
build-windows: build-windows-amd64

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

.PHONY: lint
lint:
	revive -config revive/config.toml -formatter friendly ./...

.PHONY: for-all
for-all:
	@set -e; for dir in $(ALL_MODULES); do \
	  (cd "$${dir}" && $${CMD} ); \
	done

# Prepare the vagrant system by installing go-msi, wix, inspec and configuring the path.
# Assumes stanza-plugins has already been cloned and checked out with the correct tag.
.PHONY: vagrant-prep
vagrant-prep: workdir = "build/windows"
vagrant-prep:
	file $(workdir)/go-msi.exe >/dev/null || wget -O $(workdir)/go-msi.exe https://github.com/observIQ/go-msi/releases/download/v2.0.0/go-msi.exe
	file $(workdir)/cinc-auditor.msi >/dev/null || wget -O $(workdir)/cinc-auditor.msi http://downloads.cinc.sh/files/stable/cinc-auditor/4.17.7/windows/2012r2/cinc-auditor-4.17.7-1-x64.msi
	
	file wix310-binaries.zip >/dev/null || wget -O wix310-binaries.zip http://wixtoolset.org/downloads/v3.10.3.3007/wix310-binaries.zip
	mkdir -p $(workdir)/wix310
	ls $(workdir)/wix310/sdk >/dev/null || unzip -o wix310-binaries.zip -d $(workdir)/wix310

	cp -r stanza-plugins/plugins $(workdir)/plugins

	cd $(workdir) && vagrant up
	cd $(workdir) && vagrant winrm -c "setx PATH \"%PATH%;C:\\vagrant\\wix310\;C:\\vagrant\""
	cd $(workdir) && vagrant winrm -c "C:\\vagrant\\cinc-auditor.msi"

# Assumes $GIT_TAG exists in the env: v0.0.1
.PHONY: wix
wix: workdir = "build/windows"
wix:
	cp artifacts/stanza_windows_amd64 $(workdir)/stanza.exe

	cd $(workdir) && \
		vagrant winrm -c \
		"cd C:\\vagrant; go-msi.exe make -m stanza-$${GIT_TAG}.msi --version $${GIT_TAG} --arch amd64"

.PHONY: wix-test
wix-test: workdir = "build/windows"
wix-test: vagrant-prep wix
	cd $(workdir) && vagrant winrm -c "C:\\vagrant\\stanza-$${GIT_TAG}.msi"
	sleep 10
	cd $(workdir) && vagrant winrm -c "cinc-auditor exec C:\\vagrant\test\install.rb"