## `journald_input` operator

The `journald_input` operator reads logs from the systemd journal using the `journalctl` binary, which must be in the `$PATH` of the agentt.

By default, `journalctl` will read from `/run/journal` or `/var/log/journal`. If either `directory` or `files` are set, `journalctl` will instead read from those.

The `journald_input` operator will use the `__REALTIME_TIMESTAMP` field of the journald entry as the parsed entry's timestamp. All other fields are added to the entry's record as returned by `journalctl`.

### Configuration Fields

| Field             | Default          | Description                                                                                      |
| ---               | ---              | ---                                                                                              |
| `id`              | `journald_input` | A unique identifier for the operator                                                             |
| `output`          | Next in pipeline | The connected operator(s) that will receive all outbound entries                                 |
| `poll_interval`   | 200ms            | The duration between journal polls                                                               |
| `directory`       |                  | A directory containing journal files to read entries from                                        |
| `files`           |                  | A list of journal files to read entries from                                                     |
| `write_to`        | $                | The record [field](/docs/types/field.md) written to when creating a new log entry                |
| `start_at`        | `end`            | At startup, where to start reading logs from the file. Options are `beginning` or `end`          |
| `labels`          | {}               | A map of `key: value` labels to add to the entry's labels                                        |
| `resource`        | {}               | A map of `key: value` labels to add to the entry's resource                                      |

### Example Configurations

#### Simple journald input

Configuration:
```yaml
- type: journald_input
```

Output entry sample:
```json
"entry": {
  "timestamp": "2020-04-16T11:05:49.516168-04:00",
  "record": {
    "CODE_FILE": "../src/core/unit.c",
    "CODE_FUNC": "unit_log_success",
    "CODE_LINE": "5487",
    "MESSAGE": "var-lib-docker-overlay2-bff8130ef3f66eeb81ce2102f1ac34cfa7a10fcbd1b8ae27c6c5a1543f64ddb7-merged.mount: Succeeded.",
    "MESSAGE_ID": "7ad2d189f7e94e70a38c781354912448",
    "PRIORITY": "6",
    "SYSLOG_FACILITY": "3",
    "SYSLOG_IDENTIFIER": "systemd",
    "USER_INVOCATION_ID": "de9283b4fd634213a50f5abe71b4d951",
    "USER_UNIT": "var-lib-docker-overlay2-bff8130ef3f66eeb81ce2102f1ac34cfa7a10fcbd1b8ae27c6c5a1543f64ddb7-merged.mount",
    "_AUDIT_LOGINUID": "1000",
    "_AUDIT_SESSION": "299",
    "_BOOT_ID": "c4fa36de06824d21835c05ff80c54468",
    "_CAP_EFFECTIVE": "0",
    "_CMDLINE": "/lib/systemd/systemd --user",
    "_COMM": "systemd",
    "_EXE": "/usr/lib/systemd/systemd",
    "_GID": "1000",
    "_HOSTNAME": "testhost",
    "_MACHINE_ID": "d777d00e7caf45fbadedceba3975520d",
    "_PID": "18667",
    "_SELINUX_CONTEXT": "unconfined\n",
    "_SOURCE_REALTIME_TIMESTAMP": "1587049549515868",
    "_SYSTEMD_CGROUP": "/user.slice/user-1000.slice/user@1000.service/init.scope",
    "_SYSTEMD_INVOCATION_ID": "da8b20bdc65e4f6f9ca35d6352199b56",
    "_SYSTEMD_OWNER_UID": "1000",
    "_SYSTEMD_SLICE": "user-1000.slice",
    "_SYSTEMD_UNIT": "user@1000.service",
    "_SYSTEMD_USER_SLICE": "-.slice",
    "_SYSTEMD_USER_UNIT": "init.scope",
    "_TRANSPORT": "journal",
    "_UID": "1000",
    "__CURSOR": "s=b1e713b587ae4001a9ca482c4b12c005;i=1efec9;b=c4fa36de06824d21835c05ff80c54468;m=a001b7ec5a;t=5a369c4a3cd88;x=f9717e0b5608807b",
    "__MONOTONIC_TIMESTAMP": "687223598170"
  }
}
```
