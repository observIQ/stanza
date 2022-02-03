# Windows Events

Once you have Stanza installed and running from the [quickstart guide](/README.md#quick-start), you can follow these steps to configure Windows Event monitoring via Stanza.

## Prerequisites

### Custom Channels

The Windows Events plugin can monitor custom channels in addition to the three configured by default. In order to add a custom channel, you need the name of it which can be found in the left sidebar of the Windows Event Viewer window.

## Configuration

| Field | Default | Description |
| --- | --- | --- |
| `enable_system_events` | `true` | Enable to collect system event logs |
| `enable_application_events` | `true` | Enable to collect application event logs |
| `enable_security_events` | `true`  | Enable to collect security event logs |
| `enable_custom_channels` | `false` | Enable to collect custom event logs from provided channels |
| `custom_channels` | `-''` | Add custom channels to get event logs  |
| `max_reads` | `100` | The maximum number of records read into memory, before beginning a new batch |
| `poll_interval` | `1` | The interval, in seconds, at which the channel is checked for new log entries. This check begins again after all new records have been read |
| `start_at` | `end` | Start reading file from 'beginning' or 'end' |

This is an example config file that can be used in the Stanza install directory, noted in the [Configuration](/README.md#Configuration) section of the quickstart guide. The Windows Events plugin supports system, application, and security events by default, but can also support custom channels if those have been configured.

```yaml
pipeline:
  # To see the Windows Events plugin, go to: https://github.com/observIQ/stanza-plugins/blob/master/plugins/windows_event.yaml
  - type: windows_event
    enable_system_events: true
    enable_application_events: true
    enable_security_events: true

  # For more info on Google Cloud output, go to: https://github.com/observIQ/stanza/blob/master/docs/operators/google_cloud_output.md
  - type: google_cloud_output
    credentials_file: C:\credentials.json
```

With custom channels enabled, the `custom_channels` field can be populated with any channel found in the Windows Event Viewer.

```yaml
pipeline:
  # To see the Windows Events plugin, go to: https://github.com/observIQ/stanza-plugins/blob/master/plugins/windows_event.yaml
  - type: windows_event
    enable_system_events: true
    enable_application_events: true
    enable_security_events: true
    enable_custom_channels: true
    custom_channels:
      - 'Hardware Events'
      - 'Key Management Service'

  # For more info on Google Cloud output, go to: https://github.com/observIQ/stanza/blob/master/docs/operators/google_cloud_output.md
  - type: google_cloud_output
    credentials_file: C:\credentials.json
```

The output is configured to go to Google Cloud utilizing a credentials file that can be generated following Google's documentation [here](https://cloud.google.com/iam/docs/creating-managing-service-account-keys).

## Next Steps

- Learn more about [plugins](/docs/plugins.md).
- Read up on how to write a stanza [pipeline](/docs/pipeline.md).
- Check out stanza's list of [operators](/docs/operators/README.md).
- Check out the [FAQ](/docs/faq.md).
