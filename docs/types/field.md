## Fields

_Fields_ are the primary way to tell stanza which values of an entry to use in its operators.
Most often, these will be things like fields to parse for a parser operator, or the field to write a new value to.

Fields are `.`-delimited strings which allow you to select attributes or bodys on the entry. Fields can currently be used to select attributes, values on a body, or resource values. To select a attribute, prefix your field with `$attribute` such as with `$attribute.my_attribute`. For values on the body, use the prefix `$body` such as `$body.my_value`. For resource values, use the prefix `$resource`.

If a key contains a dot in it, a field can alternatively use bracket syntax for traversing through a map. For example, to select the key `k8s.cluster.name` on the entry's body, you can use the field `$body["k8s.cluster.name"]`.

body fields can be nested arbitrarily deeply, such as `$body.my_value.my_nested_value`.

If a field does not start with either `$attribute` or `$body`, `$body` is assumed. For example, `my_value` is equivalent to `$body.my_value`.

## Examples

Using fields with the restructure operator.

Config:
```yaml
- type: restructure
  ops:
    - add:
        field: "key3"
        value: "value3"
    - remove: "$body.key2.nested_key1"
    - add:
        field: "$attributes.my_attribute"
        value: "my_attribute_value"
```

<table>
<tr><td> Input entry </td> <td> Output entry </td></tr>
<tr>
<td>

```json
{
  "timestamp": "",
  "attributes": {},
  "body": {
    "key1": "value1",
    "key2": {
      "nested_key1": "nested_value1",
      "nested_key2": "nested_value2"
    }
  }
}
```

</td>
<td>

```json
{
  "timestamp": "",
  "attributes": {
    "my_attribute": "my_attribute_value"
  },
  "body": {
    "key1": "value1",
    "key2": {
      "nested_key2": "nested_value2"
    },
    "key3": "value3"
  }
}
```

</td>
</tr>
</table>
