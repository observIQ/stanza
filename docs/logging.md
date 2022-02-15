# Configuring Logging

The Stanza application itself emits logs. These logs may be configured in your config.yaml.

If no logging config is provided, Stanza defaults to logging to stdout, with "info" as its logging level.


## Configuration options

| Option          | Default      | Description                                                                  |
|-----------------|--------------|------------------------------------------------------------------------------|
| output          | "stdout"     | Where logs should be output. May be either "stdout" or "file".               |
| level           | "info"       | Zap log level; May be "debug", "info", "warn", "error", "panic", or "fatal". |
| file.filename   | "stanza.log" | If using file output, this will be the output path of your log file.         |
| file.maxsize    | 10           | Maximum size of the log file, in megabytes, before rotating.                 |
| file.maxbackups | 5            | Maximum number of rotated log backups to keep.                               |
| file.maxage     | 7            | Maximum number of days to keep rotated log backups.                          |

## Sample configs

Enabling debug logs:
```yaml
pipeline:
    # Your stanza pipeline goes here!
logging:
    output: stdout
    level: debug
```

Logging to a rotating log file:
```yaml
pipeline:
    # Your stanza pipeline goes here!
logging:
    output: file
    file:
        filename: "stanza-log.log"
```

Logging to a rotating file, tweaking the default rotation parameters (let log files rotate when they reach 5 mb, keep only 1 file as backup when rotating, and keep backup files for up to 3 days):
```yaml
pipeline:
    # Your stanza pipeline goes here!
logging:
    output: file
    file:
        filename: "stanza-log.log"
        maxsize: 5
        maxbackups: 1
        maxage: 3
```
