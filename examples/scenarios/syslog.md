# Syslog

Once you have Stanza installed and running from the [quickstart guide](./README.md#quick-start), you can follow these steps to configure Syslog monitoring.

## Prerequisites

It may be necessary to add an inbound firewall rule on Windows. To do this:
 * Navigate to Windows Firewall Advanced Settings, and then Inbound Rules
 * Create a new rule and set the Rule Type to "Port"
 * For Protocol and Ports, select "UDP" and a specific local port of 514
 * For Action, select "Allow the connection"
 * For Profile, apply to "Domain", "Private", and "Public"
 * Set a name to easily identify rule, such as "Allow Syslog Inbound Connections to 514 UDP"

## Configuration

This is an example config file that can be used in the Stanza install directory. The Syslog plugin supports UDP and TCP logs, using UDP by default.

```yaml
pipeline:
  # To see the Syslog plugin, go to: https://github.com/observIQ/stanza-plugins/blob/master/plugins/syslog.yaml
  - type: syslog
    listen_port: 514
    listen_ip: "0.0.0.0"
    connection_type: udp
    protocol: rfc5424
    location: UTC

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
