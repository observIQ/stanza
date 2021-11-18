## `file_output` operator

The `file_output` operator will write log entries to a file. By default, they will be written as JSON-formatted lines, but if a `Format` is provided, that format will be used as a template to render each log line

### Configuration Fields

| Field    | Default       | Description                                                                                                   |
| ---      | ---           | ---                                                                                                           |
| `id`     | `file_output` | A unique identifier for the operator                                                                          |
| `path`   | required      | A path to write the entries to                                                                                |
| `format` |               | A [go template](https://golang.org/pkg/text/template/) that will be used to render each entry into a log line |


### Example Configurations

#### Simple configuration

Configuration:
```yaml
- type: file_output
  path: /tmp/output.json
```

#### Custom format

Configuration:
```yaml
- type: file_output
  path: /tmp/output.log
  format: "Time: {{.Timestamp}} body: {{.body}}\n"
```
