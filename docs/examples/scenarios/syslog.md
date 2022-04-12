# Syslog

Once you have Stanza installed and running from the [quickstart guide](/README.md#quick-start), you can follow these steps to configure Syslog monitoring.

## Prerequisites

It may be necessary to add an inbound firewall rule.

### Windows

- Navigate to Windows Firewall Advanced Settings, and then Inbound Rules
- Create a new rule and set the Rule Type to "Port"
- For Protocol and Ports, select "UDP" and a specific local port of 514
- For Action, select "Allow the connection"
- For Profile, apply to "Domain", "Private", and "Public"
- Set a name to easily identify rule, such as "Allow Syslog Inbound Connections to 514 UDP"

 ### Linux

- Using Firewalld:
```shell
firewall-cmd --permanent --add-port=514/udp
firewall-cmd --reload
```
- Using UFW:
```shell
ufw allow 514
```

## Configuration

| Field              | Default          | Description |                                                                                                                                                                                                  
| ---                | ---              | ---         |                                                                                                                                                                                                     
| `listen_port`      | `514`            | Network port to listen on                                                              |                                                     
| `listen_ip`        | `0.0.0.0`        | A network interface for the agent to bind. Typically 0.0.0.0 for most configurations.  |
| `connection_type`  | `udp`            | Transport protocol to use (`udp` or `tcp`)                                             |
| `protocol`         | `rfc5424 (IETF)` | Protocol of received syslog messages (`rfc3164 (BSD)` or `rfc5424 (IETF)`)             |
| `location`         | `UTC`            | [Geographic location (timezone)](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) to use when [parsing the timestamp](https://github.com/observIQ/stanza/blob/master/docs/types/timestamp.md) (Syslog RFC 3164 only) |
| `tls_enable`       | `false`          | Set to `true` to enable TLS |
| `tls_certificate`  |                  | Path to x509 PEM encoded TLS certificate file |
| `tls_private_key`  |                  | Path to x509 PEM encoded TLS private key file |
| `tls_min_version`  | `"1.2"`          | Minimum TLS version to support (string)       |

This is an example config file that can be used in the Stanza install directory, noted in the [Configuration](/README.md#Configuration) section of the quickstart guide. The Syslog plugin supports UDP and TCP logs, using UDP by default.

```yaml
pipeline:
  # To see the Syslog plugin, go to: https://github.com/observIQ/stanza-plugins/blob/main/plugins/syslog.yaml
  - type: syslog
    listen_port: 514
    listen_ip: "0.0.0.0"

  # For more info on Google Cloud output, go to: https://github.com/observIQ/stanza/blob/master/docs/operators/google_cloud_output.md
  - type: google_cloud_output
    credentials_file: /tmp/credentials.json
```

This is an example that uses TCP with TLS.

```yaml
pipeline:
  # To see the Syslog plugin, go to: https://github.com/observIQ/stanza-plugins/blob/main/plugins/syslog.yaml
  - type: syslog
    listen_port: 514
    listen_ip: "0.0.0.0"
    connection_type: tcp
    tls_enable: true
    tls_certificate: /path/to/certificate
    tls_private_key: /path/to/privateKey
    tls_min_version: "1.2"

  # For more info on Google Cloud output, go to: https://github.com/observIQ/stanza/blob/master/docs/operators/google_cloud_output.md
  - type: google_cloud_output
    credentials_file: /tmp/credentials.json
```

The output is configured to go to Google Cloud utilizing a credentials file that can be generated following Google's documentation [here](https://cloud.google.com/iam/docs/creating-managing-service-account-keys).

## Next Steps

- Learn more about [plugins](/docs/plugins.md).
- Read up on how to write a stanza [pipeline](/docs/pipeline.md).
- Check out stanza's list of [operators](/docs/operators/README.md).
- Check out the [FAQ](/docs/faq.md).
