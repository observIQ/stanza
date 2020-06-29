# Entry

Entry is the base representation of log data as it moves through a pipeline. All plugins either create, modify, or consume entries.

## Structure
| Field       | Description                                                                                                                 |
| ---         | ---                                                                                                                         |
| `timestamp` | The timestamp associated with the log                                                                                       |
| `severity`  | The [severity](/docs/types/field.md) of the log.                                                                            |
| `labels`    | A map of key/value pairs that describes the metadata of the log. This value is often used by a consumer to categorize logs. |
| `tags`      | An array of values that describes the metadata of the log. This value is often used by a consumer to tag incoming logs.     |
| `record`    | The contents of the log. This value is often modified and restructured in the pipeline.                                     |
