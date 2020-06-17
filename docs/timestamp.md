## `timestamp` parsing parameters

Parser plugins can parse a timestamp and attach the resulting time value to a log entry.

| Field         | Default    | Description                                                                      |
| ---           | ---        | ---                                                                              |
| `parse_from`  | required   | A [field](/docs/field.md) that indicates the field to be parsed as JSON          |
| `layout_type` | `strptime` | The type of timestamp. Valid values are `strptime`, `gotime`, and `epoch`        |
| `layout`      | required   | The exact layout of the timestamp to be parsed                                   |
| `preserve`    | false      | Preserve the unparsed value on the record                                        |


### How to specify timestamp parsing parameters

Most parser plugins, such as [`regex_parser`](/docs/plugins/regex_parser.md) support these fields inside of a `timestamp` block.

If a timestamp block is specified, the parser plugin will perform the timestamp parsing _after_ performing its other parsing actions, but _before_ passing the entry to the specified output plugin.

```yaml
- id: my_regex_parser
  type: regex_parser
  regexp: '^Time=(?P<timestamp_field>\d{4}-\d{2}-\d{2}), Host=(?P<host>[^,]+)'
  timestamp:
    parse_from: timestamp_field
    layout_type: strptime
    layout: '%Y-%m-%d'
  output: my_next_plugin
```

---

As a special case, the [`time_parser`](/docs/plugins/time_parser.md) plugin supports these fields inline. This is because time parsing is the primary purpose of the plugin.
```yaml
- id: my_time_parser
  type: time_parser
  parse_from: timestamp_field
  layout_type: strptime
  layout: '%Y-%m-%d'
  output: my_next_plugin
```

### Example Configurations

#### Parse a timestamp using a `strptime` layout

The default `layout_type` is `strptime`, which uses "directives" such as `%Y` (4-digit year) and `%H` (2-digit hour). A full list of supported directives is found [here](https://github.com/BlueMedora/ctimefmt/blob/3e07deba22cf7a753f197ef33892023052f26614/ctimefmt.go#L63).

Configuration:
```yaml
- id: my_time_parser
  type: time_parser
  parse_from: timestamp_field
  layout_type: strptime
  layout: '%a %b %e %H:%M:%S %Z %Y'
  output: my_next_plugin
```

<table>
<tr><td> Input entry </td> <td> Output entry </td></tr>
<tr>
<td>

```json
{
  "timestamp": "",
  "record": {
    "timestamp_field": "Jun 5 13:50:27 EST 2020"
  }
}
```

</td>
<td>

```json
{
  "timestamp": "2020-06-05T13:50:27-05:00",
  "record": {}
}
```

</td>
</tr>
</table>

#### Parse a timestamp using a `gotime` layout

The `gotime` layout type uses Golang's native time parsing capabilities. Golang takes an [unconventional approach](https://www.pauladamsmith.com/blog/2011/05/go_time.html) to time parsing. Finer details are well-documented [here](https://golang.org/src/time/format.go?s=25102:25148#L9).

Configuration:
```yaml
- id: my_time_parser
  type: time_parser
  parse_from: timestamp_field
  layout_type: gotime
  layout: Jan 2 15:04:05 MST 2006
  output: my_next_plugin
```

<table>
<tr><td> Input entry </td> <td> Output entry </td></tr>
<tr>
<td>

```json
{
  "timestamp": "",
  "record": {
    "timestamp_field": "Jun 5 13:50:27 EST 2020"
  }
}
```

</td>
<td>

```json
{
  "timestamp": "2020-06-05T13:50:27-05:00",
  "record": {}
}
```

</td>
</tr>
</table>

#### Parse a timestamp using an `epoch` layout (and preserve the original value)

The `epoch` layout type uses can consume epoch-based timestamps. The following layouts are supported:

| Layout | Meaning                                   | Example              | `string` | `int64`    | `float64`  |
| ---    | ---                                       | ---                  | ---      | ---        | ---        |
| `s`    | Seconds since the epoch                   | 1136214245           | :heavy_check_mark:      | :heavy_check_mark:        | :heavy_check_mark:        |
| `ms`   | Milliseconds since the epoch              | 1136214245123        | :heavy_check_mark:      | :heavy_check_mark:        | :heavy_check_mark:        |
| `us`   | Microseconds since the epoch              | 1136214245123456     | :heavy_check_mark:      | :heavy_check_mark:        | :heavy_check_mark:        |
| `ns`   | Nanoseconds since the epoch               | 1136214245123456789  | :heavy_check_mark:      | :heavy_check_mark:        | :heavy_check_mark: (lossy) |
| `s.ms` | Seconds plus milliseconds since the epoch | 1136214245.123       | :heavy_check_mark:      | :heavy_check_mark: (lossy) | :heavy_check_mark:        |
| `s.us` | Seconds plus microseconds since the epoch | 1136214245.123456    | :heavy_check_mark:      | :heavy_check_mark: (lossy) | :heavy_check_mark:        |
| `s.ns` | Seconds plus nanoseconds since the epoch  | 1136214245.123456789 | :heavy_check_mark:      | :heavy_check_mark: (lossy) | :heavy_check_mark: (lossy) |

Configuration:
```yaml
- id: my_time_parser
  type: time_parser
  parse_from: timestamp_field
  layout_type: epoch
  layout: s
  preserve: true
  output: my_next_plugin
```

<table>
<tr><td> Input entry </td> <td> Output entry </td></tr>
<tr>
<td>

```json
{
  "timestamp": "",
  "record": {
    "timestamp_field": 1136214245
  }
}
```

</td>
<td>

```json
{
  "timestamp": "2006-01-02T15:04:05-07:00",
  "record": {
    "timestamp_field": 1136214245
  }
}
```

</td>
</tr>
</table>