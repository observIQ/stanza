## Setup

If you are new to Go, it is recommended to work through the [How to Write Go Code](https://golang.org/doc/code.html) tutorial, which will ensure your Go environment is configured.

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