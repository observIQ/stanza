## `flatten` operator

The `flatten` operator flattens a field by moving its children up to the same level as the field.
The operator only flattens a single level deep.

### Configuration Fields

| Field      | Default          | Description                                                                                                                                                                                                                              |
| ---        | ---              | ---                                                                                                                                                                                                                                      |
| `id`       | `flatten`    | A unique identifier for the operator                                                                                                                                                                                                     |
| `output`   | Next in pipeline | The connected operator(s) that will receive all outbound entries                                                                                                                                                                         |
| `field`      | required       | The [field](/docs/types/field.md) to be flattened.                                                       |
| `on_error` | `send`           | The behavior of the operator if it encounters an error. See [on_error](/docs/types/on_error.md)                                                                                                                                          |
| `if`       |                  | An [expression](/docs/types/expression.md) that, when set, will be evaluated to determine whether this operator should be used for the given entry. This allows you to do easy conditional parsing without branching logic with routers. |

Example usage:

<hr>
Flatten an object to the base of the record
<br>
<br>

```yaml
- type: flatten
  field: key1
```

<table>
<tr><td> Input Entry </td> <td> Output Entry </td></tr>
<tr>
<td>

```json
{
  "resource": { },
  "attributes": { },  
  "record": {
    "key1": {
      "nested1": "nestedval1",
      "nested2": "nestedval2"
    },
    "key2": "val2"
  }
}
```

</td>
<td>

```json
  {
    "resource": { },
    "attributes": { },  
    "record": {
      "nested1": "nestedval1",
      "nested2": "nestedval2",
      "key2": "val2"
    }
  }
```

</td>
</tr>
</table>

<hr>
Flatten an object within another object
<br>
<br>

```yaml
- type: flatten
  field: wrapper.key1
```

<table>
<tr><td> Input Entry </td> <td> Output Entry </td></tr>
<tr>
<td>

```json
{
  "resource": { },
  "attributes": { },  
  "record": {
    "wrapper": {
      "key1": {
        "nested1": "nestedval1",
        "nested2": "nestedval2"
      },
      "key2": "val2"
    }
  }
}
```

</td>
<td>

```json
{
  "resource": { },
  "attributes": { },  
  "record": {
    "wrapper": {
      "nested1": "nestedval1",
      "nested2": "nestedval2",
      "key2": "val2"
    }
  }
}
```

</td>
</tr>
</table>
