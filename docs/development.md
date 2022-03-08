## Setup

If you are new to Go, it is recommended to work through the [How to Write Go Code](https://golang.org/doc/code.html) tutorial, which will ensure your Go environment is configured.

Clone this repo into your Go workspace:
```
cd $GOPATH/src
mkdir -p github.com/observiq && cd github.com/observiq
git clone git@github.com:observiq/stanza.git
cd $GOPATH/src/github.com/observiq/stanza
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

## Integration Tests with CINC Test Kitchen

End to end tests can be accomplished with  the following tools:
- [CINC Test Kitchen and Auditor](https://cinc.sh/)
- [Vagrant](https://www.vagrantup.com/)
- [Virtual Box](https://www.virtualbox.org/)

Build the release
```bash
make release-test
```

Run CINC against all supported operating systems:
```bash
kitchen create
kitchen converge -c 8
kitchen verify -c 1
kitchen destroy
```

If you want to target a single instance, you can use a regex. For example, you can use the `test`
command to create, converge, verify, and destroy Ubuntu 18.
```
kitchen test ubuntu-18
```

## Building Windows MSI

A Windows MSI installer can be built using the following tools:

- [go-msi observiq fork](https://github.com/observIQ/go-msi/)
- [Wix toolset](https://wixtoolset.org/)
- A valid Microsoft Windows system

Github actions is responsible for building and publishing MSI installers. If you wish to build an MSI
on your machine, the Makefile has several targets that can help streamline the build process.

- `vagrant-prep`: Will deploy a Windows system and prep it for building MSI package. **PLEASE NOTE** that valid Windows licensing is your responsibility.
- `wix`: Will build the MSI.
- `wix-test`: Will prep, build, and test.
- `wix-test-uninstall`: Will run an uninstall test against a system after **a manual uninstall**.