## Fields

_Fields_ are the primary way to tell the log agent which fields of a log's record to use for the operations of its plugins.
Most often, these will be things like fields to parse for a parser plugin, or the field to write a new value to.

Fields are `.`-delimited strings which allow you to selected into nested records in the field. The root level is specified by `$` such as in `$.key1`, but since all fields are expected to be relative to root, the `$` is implied and be omitted. For example, in the record below, `nested_key` can be equivalently selected with `$.key2.nested_key` or `key2.nested_key`.

```json
{
  "key1": "value1",
  "key2": {
    "nested_key": "nested_value"
  }
}
```

## Examples

Using fields with the restructure plugin.

Config:
```yaml
- id: my_restructure
  type: restructure
  output: my_restructure_receiver
  ops:
    - add:
        field: "key3"
        value: "value3"
    - remove: "key2.nested_key1"
```

<table>
<tr><td> Input record </td> <td> Output record </td></tr>
<tr>
<td>

```json
{
  "key1": "value1",
  "key2": {
    "nested_key1": "nested_value1",
    "nested_key2": "nested_value2"
  }
}
```

</td>
<td>

```json
{
  "key1": "value1",
  "key2": {
    "nested_key2": "nested_value2"
  },
  "key3": "value3"
}
```

</td>
</tr>
</table>
