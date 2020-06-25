## Severity Parsing

`bplogagent` uses a flexible severity parsing system based on the integers 0 to 100. Standard severities are provided at multiples of 10.

This severity system allows each output plugin to interpret the values 0 to 100 as appropriate for the corresponding backend.

The following named severity levels are supported.

| Severity    | Numeric Value | Aliases (Case Insensitive) |
| ---         | ---           | ---                        |
| Default     |        0      | `"default"`                |
| Trace       |       10      | `"trace"`                  |
| Debug       |       20      | `"debug"`                  |
| Info        |       30      | `"info"`                   |
| Notice      |       40      | `"notice"`                 |
| Warning     |       50      | `"warning"`, `"warn"`      |
| Error       |       60      | `"error"`, `"err"`         |
| Critical    |       70      | `"critical"`, `"crit"`     |
| Alert       |       80      | `"alert"`                  |
| Emergenccy  |       90      | `"emergency"`              |
| Catastrophe |      100      | `"catastrophe"`            |


### How `severity` mapping works

Severity parsing behavior is defined in a config file using a severity `mapping`. The general structure of the `mapping` is as follows:

```yaml
...
  mapping:
    severity_as_number_or_alias: value | list of values | range | special
    severity_as_number_or_alias: value | list of values | range | special
```

The following example :
```yaml
...
  mapping:

    # single value to be parsed as "error"
    error: "oops" 

    # list of values to be parsed as "warning"
    warning: 
      - "hey!" 
      - "YSK" 

    # range of values to be parsed as "info"
    info: 
      - min: 300
      - max: 399

    # special value representing the range 200-299, to be parsed as "debug"
    debug: "2xx"

    # single value to be parsed as a custom level of 36
    36: "medium"

    # mix and match the above concepts
    95:
      - "really serious"
      - min: 9001
        max: 9050
      - "5xx"
```

Notice that it is not necessary to specify the aliases defined in the table above. These will be automatically recognized. Aliases can be overridden.

### `severity` parsing parameters

Parser plugins can parse a severity and attach the resulting value to a log entry.

| Field         | Default  | Description                                                                   |
| ---           | ---      | ---                                                                           |
| `parse_from`  | required | A [field](/docs/types/field.md) that indicates the field to be parsed as JSON |
| `preserve`    | false    | Preserve the unparsed value on the record                                     |
| `mapping`     | Aliases  | A formatted set of values that should be interpretted as severity levels.     |


### How to use severity parsing

All parser plugins, such as [`regex_parser`](/docs/plugins/regex_parser.md) support these fields inside of a `severity` block.

If a severity block is specified, the parser plugin will perform the severity parsing _after_ performing its other parsing actions, but _before_ passing the entry to the specified output plugin.

```yaml
- id: my_regex_parser
  type: regex_parser
  regexp: '^StatusCode=(?P<severity_field>\d{3}), Host=(?P<host>[^,]+)'
  timestamp:
    parse_from: severity_field
    mapping:
      critical: "5xx"
      error: "4xx"
      info: "3xx"
      debug: "2xx"
  output: my_next_plugin
```

---

As a special case, the [`severity_parser`](/docs/plugins/severity_parser.md) plugin supports these fields inline. This is because severity parsing is the primary purpose of the plugin.
```yaml
- id: my_severity_parser
  type: severity_parser
  parse_from: severity_field
  mapping:
    critical: "5xx"
    error: "4xx"
    info: "3xx"
    debug: "2xx"
  output: my_next_plugin
```

### Example Configurations

#### Parse a severity from a standard value

Configuration:
```yaml
- id: my_severity_parser
  type: severity_parser
  parse_from: severity_field
  output: my_next_plugin
```

<table>
<tr><td> Input entry </td> <td> Output entry </td></tr>
<tr>
<td>

```json
{
  "severity": 0, // Default
  "record": {
    "severity_field": "ERROR"
  }
}
```

</td>
<td>

```json
{
  "severity": 60, // Error
  "record": {}
}
```

</td>
</tr>
</table>

#### Parse a severity from a non-standard value

Configuration:
```yaml
```yaml
- id: my_severity_parser
  type: severity_parser
  parse_from: severity_field
  mapping:
    error: "nooo!"
  output: my_next_plugin
```

<table>
<tr><td> Input entry </td> <td> Output entry </td></tr>
<tr>
<td>

```json
{
  "severity": 0, // Default
  "record": {
    "severity_field": "nooo!"
  }
}
```

</td>
<td>

```json
{
  "severity": 60, // Error
  "record": {}
}
```

</td>
</tr>
</table>

#### Parse a severity from any of several non-standard values

Configuration:
```yaml
```yaml
- id: my_severity_parser
  type: severity_parser
  parse_from: severity_field
  mapping:
    error: 
      - "nooo!"
      - "nooooooo"
    info: "HEY"
    debug: 1234
  output: my_next_plugin
```

<table>
<tr><td> Input entry </td> <td> Output entry </td></tr>
<tr>
<td>

```json
{
  "severity": 0, // Default
  "record": {
    "severity_field": "nooo!"
  }
}
```

</td>
<td>

```json
{
  "severity": 60, // Error
  "record": {}
}
```

</td>
</tr>
<tr>
<td>

```json
{
  "severity": 0, // Default
  "record": {
    "severity_field": "nooooooo"
  }
}
```

</td>
<td>

```json
{
  "severity": 60, // Error
  "record": {}
}
```

</td>
</tr>
<tr>
<td>

```json
{
  "severity": 0, // Default
  "record": {
    "severity_field": "hey"
  }
}
```

</td>
<td>

```json
{
  "severity": 30, // Info
  "record": {}
}
```

</td>
</tr>
<tr>
<td>

```json
{
  "severity": 0, // Default
  "record": {
    "severity_field": 1234
  }
}
```

</td>
<td>

```json
{
  "severity": 20, // Debug
  "record": {}
}
```

</td>
</tr>
<tr>
<td>

```json
{
  "severity": 0, // Default
  "record": {
    "severity_field": "unknown"
  }
}
```

</td>
<td>

```json
{
  "severity": 0, // Default
  "record": {}
}
```

</td>
</tr>
</table>

#### Parse a severity from a range of values

Configuration:
```yaml
```yaml
- id: my_severity_parser
  type: severity_parser
  parse_from: severity_field
  mapping:
    error:
      - min: 1
        max: 5
    alert:
      - min: 6
        max: 10
  output: my_next_plugin
```

<table>
<tr><td> Input entry </td> <td> Output entry </td></tr>
<tr>
<td>

```json
{
  "severity": 0, // Default
  "record": {
    "severity_field": 3
  }
}
```

</td>
<td>

```json
{
  "severity": 60, // Error
  "record": {}
}
```

</td>
</tr>
<tr>
<td>

```json
{
  "severity": 0, // Default
  "record": {
    "severity_field": 9
  }
}
```

</td>
<td>

```json
{
  "severity": 80, // Alert
  "record": {}
}
```

</td>
</tr>
<tr>
<td>

```json
{
  "severity": 0, // Default
  "record": {
    "severity_field": 12
  }
}
```

</td>
<td>

```json
{
  "severity": 0, // Default
  "record": {}
}
```

</td>
</tr>
</table>

#### Parse a severity from a HTTP Status Codes value

Special values are provided to represent http status code ranges.

| Value | Meaning   |
| ---   | ---       |
| 2xx   | 200 - 299 |
| 3xx   | 300 - 399 |
| 4xx   | 400 - 499 |
| 5xx   | 500 - 599 |

Configuration:
```yaml
```yaml
- id: my_severity_parser
  type: severity_parser
  parse_from: severity_field
  mapping:
    critical: "5xx"
    error: "4xx"
    info: "3xx"
    debug: "2xx"
  output: my_next_plugin
```

<table>
<tr><td> Input entry </td> <td> Output entry </td></tr>
<tr>
<td>

```json
{
  "severity": 0, // Default
  "record": {
    "severity_field": 302
  }
}
```

</td>
<td>

```json
{
  "severity": 30, // Warn
  "record": {}
}
```

</td>
</tr>
<tr>
<td>

```json
{
  "severity": 0, // Default
  "record": {
    "severity_field": 404
  }
}
```

</td>
<td>

```json
{
  "severity": 60, // Error
  "record": {}
}
```

</td>
</tr>
<tr>
<td>

```json
{
  "severity": 0, // Default
  "record": {
    "severity_field": 200
  }
}
```

</td>
<td>

```json
{
  "severity": 20, // Debug
  "record": {}
}
```

</td>
</tr>
</table>
