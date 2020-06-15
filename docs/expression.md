# Expressions

Expressions give the config flexibility by allowing dynamic business logic rules to be included in static configs.
Most notably, expressions can be used to route messages and add new fields based on the contents of the log entry
being processed.

For reference documentation of the expression language, see [here](https://github.com/antonmedv/expr/blob/master/docs/Language-Definition.md).

In most cases, the record of the entry being processed can be accessed with the `$` variable in the expression. See the examples below for syntax.

## Examples

### Map severity values to standard values

```yaml
- id: map_severity
  type: restructure
  output: my_receiver
  ops:
    - add:
        field: severity
        value_expr: '$.raw_severity in ["critical", "super_critical"] ? "error" : $.raw_severity'
```
