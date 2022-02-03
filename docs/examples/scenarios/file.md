# File

Once you have Stanza installed and running from the [quickstart guide](/README.md#quick-start), you can follow these steps to configure a file to send logs via Stanza.

## Prerequisites

On the host that Stanza will read logs from, make sure you know the location of the file you wish to gather logs.

## Configuration

| Field | Default | Description |
| --- | --- | --- |
| `file_log_path` | `''` | Specify a single path or multiple paths to read one or many files. You may also use a wildcard (*) to read multiple files within a directory. |
| `exclude_file_log_path` | `''` | Specify a single path or multiple paths to exclude one or many files from being read. You may also use a wildcard (*) to exclude multiple files from being read within a directory. |
| `enable_multiline` | `false` | Enable to parse Multiline Log Files |
| `multiline_line_start_pattern` | `''` | A Regex pattern that matches the start of a multiline log entry in the log file. |
| `encoding` | `utf-8` | Specify the encoding of the file(s) being read. In most cases, you can leave the default option selected. |
| `log_type` | `file` | Adds the specified 'Type' as a label to each log message. |
| `start_at` | `beginning` | Start reading file from 'beginning' or 'end' |

This is an example config file that can be used in the Stanza install directory, noted in the [Configuration](/README.md#Configuration) section of the quickstart guide. It uses a simple [file operator](/docs/operators/file_input.md) to send logs to Google Cloud utilizing a credentials file that can be generated following Google's documentation [here](https://cloud.google.com/iam/docs/creating-managing-service-account-keys).

```yaml
pipeline:
  # For more details on the file operator, go to: https://github.com/observIQ/stanza/blob/master/docs/operators/file_input.md
  - type: file_input
    include:
      - /sample/file/path.log

  # For more info on Google Cloud output, go to: https://github.com/observIQ/stanza/blob/master/docs/operators/google_cloud_output.md
  - type: google_cloud_output
    credentials_file: /tmp/credentials.json
```

If your log file uses multiline log messages, you can use the `multiline` field to define a pattern for the beginning of each log message, as in the following example.

```yaml
pipeline:
  - type: file_input
    include:
      - /sample/file/path.log
    multiline:
      line_start_pattern: 'START '

  - type: google_cloud_output
    credentials_file: /tmp/credentials.json
```

More examples for multiline logs can be found in at the [file operator](/docs/operators/file_input.md#multiline-file-input) page.

## Next Steps

- Learn more about [plugins](/docs/plugins.md).
- Read up on how to write a stanza [pipeline](/docs/pipeline.md).
- Check out stanza's list of [operators](/docs/operators/README.md).
- Check out the [FAQ](/docs/faq.md).
