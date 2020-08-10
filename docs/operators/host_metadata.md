## `host_decorator` operator

The `host_decorator` operator adds labels to incoming entries.

### Configuration Fields

| Field              | Default          | Description                                                                                     |
| ---                | ---              | ---                                                                                             |
| `id`               | `metadata`       | A unique identifier for the operator                                                            |
| `output`           | Next in pipeline | The connected operator(s) that will receive all outbound entries                                |
| `include_hostname` | `true`           | Whether to set the `hostname` label on entries                                                  |
| `on_error`         | `send`           | The behavior of the operator if it encounters an error. See [on_error](/docs/types/on_error.md) |

### Example Configurations

#### Add static tags and labels

Configuration:
```yaml
- type: host_decorator
  include_hostname: true
```

<table>
<tr><td> Input entry </td> <td> Output entry </td></tr>
<tr>
<td>

```json
{
  "timestamp": "2020-06-15T11:15:50.475364-04:00",
  "labels": {},
  "record": {
    "message": "test"
  }
}
```

</td>
<td>

```json
{
  "timestamp": "2020-06-15T11:15:50.475364-04:00",
  "labels": {
    "hostname": "my_host"
  },
  "record": {
    "message": "test"
  }
}
```

</td>
</tr>
</table>
