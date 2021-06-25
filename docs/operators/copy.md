## `copy` operator

The `copy` operator copies a value from one [field](/docs/types/field.md) to another.

### Configuration Fields

| Field      | Default          | Description                                                                                                                                                                                                                              |
| ---        | ---              | ---                                                                                                                                                                                                                                      |
| `id`       | `copy`    | A unique identifier for the operator                                                                                                                                                                                                     |
| `output`   | Next in pipeline | The connected operator(s) that will receive all outbound entries                                                                                                                                                                         |
| `from`      | required       | The [field](/docs/types/field.md)  to copy the value of.   
| `to`      | required       | The [field](/docs/types/field.md)  to copy the value into.
| `on_error` | `send`           | The behavior of the operator if it encounters an error. See [on_error](/docs/types/on_error.md)                                                                                                                                          |
| `if`       |                  | An [expression](/docs/types/expression.md) that, when set, will be evaluated to determine whether this operator should be used for the given entry. This allows you to do easy conditional parsing without branching logic with routers. |

Example usage:

<hr>
Copy a value from the record to resource

```yaml
- type: copy
  from: key
  to: $resource.newkey
```

<table>
<tr><td> Input Entry</td> <td> Output Entry </td></tr>
<tr>
<td>

```json
{
  "resource": { },
  "labels": { },  
  "record": { 
    "key":"value"
  }
}
```

</td>
<td>

```json
{
  "resource": { 
       "newkey":"value"
  },
  "labels": { },  
  "record": {
    "key":"value"
  }
}
```

</td>
</tr>
</table>

<hr>

Copy a value from the record to labels
```yaml
- type: copy
  from: key2
  to: $labels.newkey
```

<table>
<tr><td> Input Entry</td> <td> Output Entry </td></tr>
<tr>
<td>

```json
{
  "resource": { },
  "labels": { },  
  "record": {
    "key1": "val1",
    "key2": "val2"
  }
}
```

</td>
<td>

```json
{
  "resource": { },
  "labels": { 
      "newkey": "val2"
  },  
  "record": {
    "key3": "val1",
    "key2": "val2"
  }
}
```

</td>
</tr>
</table>

<hr>

Copy a value from labels to the record
```yaml
- type: copy
  from: $labels.key
  to: newkey
```

<table>
<tr><td> Input Entry</td> <td> Output Entry </td></tr>
<tr>
<td>

```json
{
  "resource": { },
  "labels": { 
      "key": "newval"
  },  
  "record": {
    "key1": "val1",
    "key2": "val2"
  }
}
```

</td>
<td>

```json
{
  "resource": { },
  "labels": { 
      "key": "newval"
  },  
  "record": {
    "key3": "val1",
    "key2": "val2",
    "newkey": "newval"
  }
}
```

</td>
</tr>
</table>

<hr>

Copy a value within the record
```yaml
- type: copy
  from: obj.nested
  to: newkey
```

<table>
<tr><td> Input Entry</td> <td> Output Entry </td></tr>
<tr>
<td>

```json
{
  "resource": { },
  "labels": { },  
  "record": {
      "obj": {
        "nested":"nestedvalue"
    }
  }
}
```

</td>
<td>

```json
{
  "resource": { },
  "labels": { },  
  "record": {
    "obj": {
        "nested":"nestedvalue"
    },
    "newkey":"nestedvalue"
  }
}
```

</td>
</tr>
</table>