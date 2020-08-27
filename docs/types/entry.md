# Entry

Entry is the base representation of log data as it moves through a pipeline. All operators either create, modify, or consume entries.

## Structure
| Field       | Description                                                                                                                 |
| ---         | ---                                                                                                                         |
| `timestamp` | The timestamp associated with the log (RFC 3339).                                                                           |
| `severity`  | The [severity](/docs/types/field.md) of the log.                                                                            |
| `resource`  | A map of key/value pairs that describe the resource from which the log originated.                                          |
| `labels`    | A map of key/value pairs that provide additional context to the log. This value is often used by a consumer to filter logs. |
| `record`    | The contents of the log. This value is often modified and restructured in the pipeline.                                     |
