## `time_parser` plugin

The `time_parser` plugin sets the timestamp on an entry by parsing a value from the record.

### Configuration Fields

| Field           | Default    | Description                                                                      |
| ---             | ---        | ---                                                                              |
| `id`            | required   | A unique identifier for the plugin                                               |
| `output`        | required   | The `id` for the plugin to send parsed entries to                                |
| `parse_from`    | required   | A [field](/docs/field.md) that indicates the field to be parsed as JSON          |
| `layout_flavor` | `strptime` | The type of timestamp. Valid values are `strptime`, `gotime`, and `epoch`        |
| `layout`        | required   | The exact layout of the timestamp to be parsed                                   |


### Example Configurations


#### Parse a timestamp using a `strptime` layout

The default timestamp parsing flavor is `strptime`, which uses "directives" such as `%Y` (4-digit year) and `%H` (2-digit hour). A full list of supported directives is found [here](https://github.com/BlueMedora/ctimefmt/blob/3e07deba22cf7a753f197ef33892023052f26614/ctimefmt.go#L63).

Configuration:
```yaml
- id: my_time_parser
  type: time_parser
  output: time_parser_receiver
  parse_from: timestamp_field
  layout_flavor: strptime
  layout: '%a %b %e %H:%M:%S %Z %Y'
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
  "record": {
    "timestamp_field": "Jun 5 13:50:27 EST 2020"
  }
}
```

</td>
</tr>
</table>

#### Parse a timestamp using a `gotime` layout

The `gotime` layout flavor uses Golang's native time parsing capabilities. Golang takes an [unconventional approach](https://www.pauladamsmith.com/blog/2011/05/go_time.html) to time parsing. Finer details are well-documented [here](https://golang.org/src/time/format.go?s=25102:25148#L9).

Configuration:
```yaml
- id: my_time_parser
  type: time_parser
  output: time_parser_receiver
  parse_from: timestamp_field
  layout_flavor: gotime
  layout: Jan 2 15:04:05 MST 2006
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
  "record": {
    "timestamp_field": "Jun 5 13:50:27 EST 2020"
  }
}
```

</td>
</tr>
</table>

#### Parse a timestamp using an `epoch` layout

The `epoch` layout flavor uses can consume epoch-based timestamps. The following layouts are supported:

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
  output: time_parser_receiver
  parse_from: timestamp_field
  layout_flavor: epoch
  layout: s
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