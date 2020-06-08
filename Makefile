GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

GIT_SHA=$(shell git rev-parse --short HEAD)

BUILD_INFO_IMPORT_PATH=github.com/bluemedora/bplogagent/internal/version
BUILD_X1=-X $(BUILD_INFO_IMPORT_PATH).GitHash=$(GIT_SHA)
ifdef VERSION
BUILD_X2=-X $(BUILD_INFO_IMPORT_PATH).Version=$(VERSION)
ARTIFACT_VERSION=$(VERSION)
else
ARTIFACT_VERSION=$(GIT_SHA)
endif
BUILD_INFO=-ldflags "${BUILD_X1} ${BUILD_X2}"


.PHONY: install-tools
install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint
	go install github.com/vektra/mockery/cmd/mockery

.PHONY: test
test:
	go test -race -coverprofile coverage.txt -coverpkg ./... ./...

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: generate
generate:
	go generate ./...

.PHONY: build
build:
	CGO_ENABLED=0 go build -o ./artifacts/bplogagent_$(ARTIFACT_VERSION)_$(GOOS)_$(GOARCH) $(BUILD_INFO) .

.PHONY: install
install:
	CGO_ENABLED=0 go install $(BUILD_INFO) .

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
