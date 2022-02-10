# Configuring Logging

The Stanza application itself emits logs. A separate logging config file may be used to control the stanza logs. This logging config can be provided to stanza on startup using the `--log_config` flag:

```
./stanza --log_config=/path/to/log/config.yaml --config=/path/to/stanza/config.yaml
```

If no logging config is provided, Stanza defaults to logging to stdout, with "info" as its logging level.


## Sample configs

Enabling debug logs:
```yaml
output: stdout
level: debug
```

Logging to a rotating log file:
```yaml
output: file
file:
    filename: "stanza-log.log"
```

Logging to a rotating file, tweaking the default rotation parameters (let log files rotate when they reach 5 mb, keep only 1 file as backup when rotating, and keep backup files for up to 3 days):
```yaml
output: file
file:
    filename: "stanza-log.log"
    maxsize: 5
    maxbackups: 1
    maxage: 3
```
