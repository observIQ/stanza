## `router` plugin

The `router` plugin allows logs to be routed dynamically based on their content.

The plugin is configured with a list of routes, where each route has an associated expression.
An entry sent to the router plugin is forwarded to the first route in the list whose associated
expression returns `true`.

An entry that does not match any of the routes is dropped and not processed further.

### Configuration Fields

| Field    | Default  | Description                              |
| ---      | ---      | ---                                      |
| `id`     | required | A unique identifier for the plugin       |
| `routes` | required | A list of routes. See below for details  |

#### Route configuration

| Field    | Default  | Description                                                                                                     |
| ---      | ---      | ---                                                                                                             |
| `output` | required | A plugin id to send an entry to if `expr` returns `true`                                                        |
| `expr`   | required | An [expression](/docs/expression.md) that returns a boolean. The record of the routed entry is available as `$` |


### Examples

#### Forward entries to different parsers based on content

```yaml
- id: my_router
  type: router
  routes:
    - output: my_json_parser
      expr: '$.format == "json"'
    - output: my_syslog_parser
      expr: '$.format == "syslog"'
```

#### Drop entries based on content

```yaml
- id: my_router
  type: router
  routes:
    - output: my_output
      expr: '$.message matches "^LOG: .* END$"'
```

#### Route with a default

```yaml
- id: my_router
  type: router
  routes:
    - output: my_json_parser
      expr: '$.format == "json"'
    - output: catchall
      expr: 'true'
```
