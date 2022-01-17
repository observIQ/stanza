<p align="center"><img src="docs/images/logo_small.png?raw=true"></p>

<center>

[![<observIQ>](https://circleci.com/gh/observIQ/stanza.svg?style=shield&circle-token=980a514f9dc5a48ac2b8e61a4cdb7555ea5646ca)](https://app.circleci.com/pipelines/github/observIQ/stanza)
[![codecov](https://codecov.io/gh/observIQ/stanza/branch/master/graph/badge.svg)](https://codecov.io/gh/observIQ/stanza)
[![Go Report Card](https://goreportcard.com/badge/github.com/observIQ/stanza)](https://goreportcard.com/report/github.com/observIQ/stanza)
[![License](https://github.com/observIQ/stanza/workflows/license/badge.svg)](https://github.com/observIQ/stanza/license)
[![Gosec](https://github.com/observIQ/stanza/actions/workflows/gosec.yml/badge.svg)](https://github.com/observIQ/stanza/actions/workflows/gosec.yml)

</center>

# About Stanza

Stanza is a fast and lightweight log transport and processing agent. It's designed as a modern replacement for Fluentd, Fluent Bit, and Logstash and can run as a standalone agent on all major operating systems. Stanza is also highly integrated to perform seamlessly with the applications in Google Cloud Platform (GCP) based production environments.

## OpenTelemetry

Stanza has been contributed to the [OpenTelemetry](https://opentelemetry.io/) project and will be intergrated into the [OpenTelemetry collector](https://github.com/open-telemetry/opentelemetry-collector). 

# Features

- Flexible
    - Supports many different input types such as file, journald, windows events, tcp / udp, and external APIs (cloudwatch, azure log analytics) as well as parsing with json and regex.
    - Easily extended by writing an "operator" or "plugin" which is just a unit of code that performs a task such as reading data from a source, parsing data, or shipping data.
- Pre-built Plugins
    - Over 80 Plugins have been pre-built and are ready to be configured.
- Lightweight with low resource consumption
    - Uses next to no resource while idling. It does not pollute the system with tons of clutter, it exists strictly in /opt/observiq/stanza with just a few files.
- Written in pure Go
    - Everything is self contained into a single binary, there are no external dependencies.
- High Performance
    - Stanza is proven to be significantly faster than fluentd and more stable at high throughput than fluentbit.

## Supported Plugins

Utilize Plugins to get up and running quickly. Some of our top Plugins include:
- Kubernetes
- NGINX
- Apache
- Windows Events
- Syslog
- MySQL
- SQL Server
- PostgreSQL
- VMWare ESXI
- Redis

 Here are many of the Plugins supported by Stanza, with more being developed all the time.

<p align="center"><img src="docs/images/stanza_plugins.png?raw=true"></p>

# Documentation

## [Quick Start](./docs/README.md)

### Installation

To install Stanza, we recommend using our single-line installer provided with each release. Stanza will automatically be running as a service upon completion. 

#### Linux/macOS
```shell
sh -c "$(curl -fsSlL https://github.com/observiq/stanza/releases/latest/download/unix-install.sh)" unix-install.sh
```
#### Windows
```pwsh
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12 ; Invoke-Expression ((New-Object net.webclient).DownloadString('https://github.com/observiq/stanza/releases/latest/download/windows-install.ps1')); Log-Agent-Install
```

For more details on installation and configuration, check out our full [Quick Start Guide](./docs/README.md)!

# Community

Stanza is an open source project. If you'd like to contribute, take a look at our [contribution guidelines](./CONTRIBUTING.md) and [developer guide](./docs/development.md). We look forward to building with you.

## Code of Conduct

Stanza follows the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md). Please report violations of the Code of Conduct to any or all [maintainers](MAINTAINERS.md).


# Other questions?

Check out our [FAQ](/docs/faq.md), or open an issue with your question. We'd love to hear from you.
