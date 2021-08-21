## `drop_output` operator

The `drop_output` operator does nothing. Useful for discarding entries while troubleshooting Stanza during development.

### Configuration Fields

Operator `drop_output` does not have configuration

### Example Configurations

Configuration
```yaml
pipeline:
- type: stdin
- type: drop_output
```

Send a message:
```bash
echo "hello world" | ./stanza -c ./config.yaml 
```

There will be no output:
```json
{"level":"info","timestamp":"2021-08-20T20:09:55.057-0400","message":"Starting stanza agent"}
{"level":"info","timestamp":"2021-08-20T20:09:55.057-0400","message":"Stanza agent started"}
{"level":"info","timestamp":"2021-08-20T20:09:55.057-0400","message":"Stdin has been closed","operator_id":"$.stdin","operator_type":"stdin"}
```

Compare with `stdout` output operator:
```json
{"level":"info","timestamp":"2021-08-20T20:09:28.314-0400","message":"Starting stanza agent"}
{"level":"info","timestamp":"2021-08-20T20:09:28.314-0400","message":"Stanza agent started"}
{"timestamp":"2021-08-20T20:09:28.314776719-04:00","severity":0,"record":"hello world"}
{"level":"info","timestamp":"2021-08-20T20:09:28.314-0400","message":"Stdin has been closed","operator_id":"$.stdin","operator_type":"stdin"}
```

