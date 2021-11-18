# Expressions

Expressions give the config flexibility by allowing dynamic business logic rules to be included in static configs.
Most notably, expressions can be used to route messages and add new fields based on the contents of the log entry
being processed.

For reference documentation of the expression language, see [here](https://github.com/antonmedv/expr/blob/master/docs/Language-Definition.md).

Available to the expressions are a few special variables:
- `$body` contains the entry's body
- `$attributes` contains the entry's attributes
- `$resource` contains the entry's resource
- `$timestamp` contains the entry's timestamp
- `env()` is a function that allows you to read environment variables

## Examples

### Add a label from an environment variable

```yaml
- type: metadata
  attributes:
    stack: 'EXPR(env("STACK"))'
```
