## `metadata` plugin

The `metadata` plugin adds tags and labels to the entry.

### Configuration Fields

| Field    | Default  | Description                                       |
| ---      | ---      | ---                                               |
| `id`     | required | A unique identifier for the plugin                |
| `output` | required | The `id` for the plugin to send parsed entries to |
| `labels` | {}       | An map of `key: value` labels to add to the entry |
| `tags`   | []       | An array of tags to add to the entry              |

Inside the label and tag values, an [expression](/expression.md) surrounded by double curly braces
(`{{  }}`) will be replaced with the evaluated form of the expression. The entry's record can be accessed
with the `$` variable in the expression so labels and tags can be added dynamically from fields.

### Example Configurations


#### Add static tags and labels

Configuration:
```yaml
- id: my_metadata
  type: metadata
  output: metadata_receiver
  tags:
    - "production"
  labels:
    environment: "production"
```

<table>
<tr><td> Input entry </td> <td> Output entry </td></tr>
<tr>
<td>

```json
{
  "timestamp": "2020-06-15T11:15:50.475364-04:00",
  "tags": [],
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
  "tags": [
    "production"
  ],
  "labels": {
    "environment": "production"
  },
  "record": {
    "message": "test"
  }
}
```

</td>
</tr>
</table>

#### Add dynamic tags and labels

Configuration:
```yaml
- id: my_metadata
  type: metadata
  output: metadata_receiver
  tags:
    - "production-{{ $.production_location }}"
  labels:
    environment: '{{ $.environment == "production" ? "prod" : "dev" }}'
```

<table>
<tr><td> Input entry </td> <td> Output entry </td></tr>
<tr>
<td>

```json
{
  "timestamp": "2020-06-15T11:15:50.475364-04:00",
  "tags": [],
  "labels": {},
  "record": {
    "production_location": "us_east",
    "environment": "nonproduction"
  }
}
```

</td>
<td>

```json
{
  "timestamp": "2020-06-15T11:15:50.475364-04:00",
  "tags": [
    "production-us_east"
  ],
  "labels": {
    "environment": "dev"
  },
  "record": {
    "production_location": "us_east",
    "environment": "nonproduction"
  }
}
```

</td>
</tr>
</table>
