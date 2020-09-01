GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

GIT_SHA=$(shell git rev-parse --short HEAD)

PROJECT_ROOT = $(shell pwd)
ARTIFACTS = ${PROJECT_ROOT}/artifacts
ALL_MODULES := $(shell find . -type f -name "go.mod" -exec dirname {} \; | sort )

BUILD_INFO_IMPORT_PATH=github.com/observiq/stanza/internal/version
BUILD_X1=-X $(BUILD_INFO_IMPORT_PATH).GitHash=$(GIT_SHA)
ifdef VERSION
BUILD_X2=-X $(BUILD_INFO_IMPORT_PATH).Version=$(VERSION)
endif
BUILD_INFO=-ldflags "${BUILD_X1} ${BUILD_X2}"


.PHONY: install-tools
install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint
	go install github.com/vektra/mockery/cmd/mockery

.PHONY: test
test: clean test-all

.PHONY: test-all
test-all:
	mkdir -p $(ARTIFACTS)
	touch $(ARTIFACTS)/coverage.txt
	@set -e; for dir in $(ALL_MODULES); do \
		(cd "$${dir}" && \
			go test -race -coverprofile coverage.txt -coverpkg ./... ./...); \
		cat "$${dir}"/coverage.txt >> $(ARTIFACTS)/coverage.txt; \
	done

.PHONY: bench
bench:
	@set -e; for dir in $(ALL_MODULES); do \
		(cd "$${dir}" && go test -run=NONE -bench '.*' ./... -benchmem); \
	done

.PHONY: clean
clean:
	rm -fr ./artifacts
	@set -e; for dir in $(ALL_MODULES); do \
		(cd "$${dir}" && rm -f coverage.txt coverage.html); \
	done

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: generate
generate:
	go generate ./...

.PHONY: build
build:
	(cd ./cmd/stanza && CGO_ENABLED=0 go build -o ../../artifacts/stanza_$(GOOS)_$(GOARCH) $(BUILD_INFO) .)

.PHONY: install
install:
	(cd ./cmd/stanza && CGO_ENABLED=0 go install $(BUILD_INFO) .)

.PHONY: build-all
build-all: build-darwin-amd64 build-linux-amd64 build-windows-amd64

.PHONY: build-darwin-amd64
build-darwin-amd64:
	@GOOS=darwin GOARCH=amd64 $(MAKE) build

.PHONY: build-linux-amd64
build-linux-amd64:
	@GOOS=linux GOARCH=amd64 $(MAKE) build

.PHONY: build-windows-amd64
build-windows-amd64:
	@GOOS=windows GOARCH=amd64 $(MAKE) build
	