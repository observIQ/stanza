# Entry

Entry is the base representation of log data as it moves through a pipeline. All operators either create, modify, or consume entries.

## Structure
| Field       | Description                                                                                                                 |
| ---         | ---                                                                                                                         |
| `timestamp` | The timestamp associated with the log (RFC 3339).                                                                           |
| `severity`  | The [severity](/docs/types/field.md) of the log.                                                                            |
| `labels`    | A map of key/value pairs that describes the metadata of the log. This value is often used by a consumer to categorize logs. |
| `resource`  | A map of key/value pairs that describes the origin of the log.                                                              |
| `record`    | The contents of the log. This value is often modified and restructured in the pipeline.                                     |
