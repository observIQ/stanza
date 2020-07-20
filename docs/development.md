## Setup

If you are new to Go, it is recommended to work through the [How to Write Go Code](https://golang.org/doc/code.html) tutorial, which will ensure your Go environment is configured.

Clone this repo into your Go workspace:
```
cd $GOPATH/src
mkdir -p github.com/observIQ && cd github.com/observIQ
git clone git@github.com:observIQ/carbon.git
cd $GOPATH/src/github.com/observIQ/carbon
```

## Building

To build the agent for another OS, run one of the following: 
```
make build // local OS
make build-windows-amd64
make build-linux-amd64
make build-darwin-amd64
```

## Running Tests

Tests can be run with `make test`.