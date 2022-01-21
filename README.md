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

## Supported [Plugins](https://github.com/observIQ/stanza-plugins)

Utilize Plugins to get up and running quickly. Here are some of our top Plugins:

<p align="center"><img src="docs/images/stanza_plugins.png?raw=true"></p>

 These are many of the Plugins supported by Stanza, with more being developed all the time. View a full list of Plugins [here](https://github.com/observIQ/stanza-plugins/tree/master/plugins).

# Quick Start

## Installation

To install Stanza, we recommend using our single-line installer provided with each release. Stanza will automatically be running as a service upon completion. 

### Linux/macOS
```shell
sh -c "$(curl -fsSlL https://github.com/observiq/stanza/releases/latest/download/unix-install.sh)" unix-install.sh
```
### Windows
```pwsh
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12 ; Invoke-Expression ((New-Object net.webclient).DownloadString('https://github.com/observiq/stanza/releases/latest/download/windows-install.ps1')); Log-Agent-Install
```

### Kubernetes
For Kubernetes, there are several guides to install and configure Stanza found [here](./examples/k8s).

## Configuration

To configure Stanza, navigate to the `config.yaml` file located in the Stanza install directory. There are a number of [plugins](./docs/plugins.md) and [operators](./docs/operators/README.md) available to configure in Stanza, but as an example we'll configure a MySQL plugin and a file operator.

Stanza also offers several outputs to be configured for sending data, including [Google Cloud Logging](./docs/operators/google_cloud_output.md) and [Elasticsearch](./docs/operators/elastic_output.md). For this example, we'll send the output to Google Cloud Logging. In addition to the `config.yaml` file, we'll need to add a `credentials.json`. To generate this credentials file, follow Google's documentation [here](https://cloud.google.com/iam/docs/creating-managing-service-account-keys).

### Plugins
This `config.yaml` collects logs from MySQL via a plugin and sends them to Google Cloud. By default, MySQL plugin collects general, slow query, and error logs but can be configured to collect MariaDB Audit logs as well by adding `enable_mariadb_audit_log: true` to the config file. More details of the MySQL plugin can be viewed [here](https://github.com/observIQ/stanza-plugins/blob/master/plugins/mysql.yaml). A full list of available plugins can be found [here](https://github.com/observIQ/stanza-plugins/blob/master/plugins/).

```yaml
...
pipeline:
...
  # An example input that configures a MySQL plugin.
  # For more info: https://github.com/observIQ/stanza/blob/master/docs/plugins.md
  - type: mysql
    enable_general_log: true
    general_log_path: "/var/log/mysql/general.log"
  ...

  # An example output that sends captured logs to Google Cloud.
  # For more info: https://github.com/observIQ/stanza/blob/master/docs/operators/google_cloud_output.md
  - type: google_cloud_output
    project_id: sample_project
    credentials_file: "/tmp/credentials.json"
...
```

### Operators
This `config.yaml` collects logs from a file and sends them to Google Cloud. A full list of available operators can be found [here](./docs/operators/README.md).

```yaml
...
pipeline:
...
  # An example input that monitors the contents of a file.
  # For more info: https://github.com/observIQ/stanza/blob/master/docs/operators/file_input.md
  - type: file_input
    include:
    - /sample/file/path.log
  ...

  # An example output that sends captured logs to Google Cloud.
  # For more info: https://github.com/observIQ/stanza/blob/master/docs/operators/google_cloud_output.md
  - type: google_cloud_output
    project_id: sample_project
    credentials_file: /tmp/credentials.json
...
```

That's it! Logs should be streaming to Google Cloud.

For more details on installation and configuration, check out our full [Install Guide](./docs/README.md)!

# Community

Stanza is an open source project. If you'd like to contribute, take a look at our [contribution guidelines](./CONTRIBUTING.md) and [developer guide](./docs/development.md). We look forward to building with you.

## Code of Conduct

Stanza follows the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md). Please report violations of the Code of Conduct to any or all [maintainers](MAINTAINERS.md).


# Other questions?

Check out our [FAQ](/docs/faq.md), or open an issue with your question. We'd love to hear from you.
