## `Retain` operator

The `retain` operator keeps the specified list of fields, and removes the rest.

### Configuration Fields

| Field      | Default          | Description                                                                                                                                                                                                                              |
| ---        | ---              | ---                                                                                                                                                                                                                                      |
| `id`       | `retain`    | A unique identifier for the operator                                                                                                                                                                                                     |
| `output`   | Next in pipeline | The connected operator(s) that will receive all outbound entries                                                                                                                                                                         |
| `fields`      | required         | A list of [fields](/docs/types/field.md)  to be kept.                                                                                                                                                     |
| `on_error` | `send`           | The behavior of the operator if it encounters an error. See [on_error](/docs/types/on_error.md)                                                                                                                                          |
| `if`       |                  | An [expression](/docs/types/expression.md) that, when set, will be evaluated to determine whether this operator should be used for the given entry. This allows you to do easy conditional parsing without branching logic with routers. |
<hr>
<b>NOTE:</b> If no fields in a group (labels, resource, or record) are specified, that entire group will be retained.
<hr>
Example usage:
<hr>
Retain fields in the record

```yaml
- type: retain
  fields:
    - key1
    - key2
```

<table>
<tr><td> Input Entry </td> <td> Output Entry </td></tr>
<tr>
<td> 

```json
{
  "resource": { },
  "labels": { },  
  "record": {
    "key1": "val1",
    "key2": "val2",
    "key3": "val3",
    "key4": "val4"
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
    "key1": "val1",
    "key2": "val2"
  }
}
```

</td>
</tr>
</table>

<hr>
Retain an object in the record

```yaml
- type: retain
  fields:
    - object
```

<table>
<tr><td> Input entry </td> <td> Output entry </td></tr>
<tr>
<td>

```json
{
  "resource": { },
  "labels": { },  
  "record": {
    "key1": "val1",
    "object": {
      "nestedkey": "val2",
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
    "object": {
      "nestedkey": "val2",
    }
  }
}
```

</td>
</tr>
</table>

<hr>
Retain fields from resource

```yaml
- type: retain
  fields:
    - $resource.key1
    - $resource.key2
```

<table>
<tr><td> Input entry </td> <td> Output entry </td></tr>
<tr>
<td>

```json
{
  "resource": { 
     "key1": "val1",
     "key2": "val2",
     "key3": "val3"
  },
  "labels": { },  
  "record": {
    "key1": "val1",
    }
  }
}
```

</td>
<td>

```json
{
  "resource": { 
     "key1": "val1",
     "key2": "val2",
  },
  "labels": { },  
  "record": { 
    "key1": "val1",
  }
}
```

</td>
</tr>
</table>

<hr>
Retain fields from labels

```yaml
- type: retain
  fields:
    - $labels.key1
    - $labels.key2
```

<table>
<tr><td> Input entry </td> <td> Output entry </td></tr>
<tr>
<td>

```json
{
  "resource": { },
  "labels": { 
     "key1": "val1",
     "key2": "val2",
     "key3": "val3"
  },  
  "record": { 
    "key1": "val1",
  }
}
```

</td>
<td>

```json
{
  "resource": { },
  "labels": { 
     "key1": "val1",
     "key2": "val2",
  },  
  "record": { 
    "key1": "val1",
  }
}
```

</td>
</tr>
</table>

<hr>
Retain fields from all sources

```yaml
- type: retain
  fields:
    - $resource.key1
    - $labels.key3
    - key5
```

<table>
<tr><td> Input entry </td> <td> Output entry </td></tr>
<tr>
<td>

```json
{
  "resource": { 
     "key1": "val1",
     "key2": "val2"
  },
  "labels": { 
     "key3": "val3",
     "key4": "val4"
  },  
  "record": { 
    "key5": "val5",
    "key6": "val6",
  }
}
```

</td>
<td>

```json
{
  "resource": { 
     "key1": "val1",
  },
  "labels": { 
     "key3": "val3",
  },  
  "record": { 
    "key5": "val5",
  }
}
```

</td>
</tr>
</table>