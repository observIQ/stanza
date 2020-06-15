## `copy` plugin

The `copy` plugin copies logs it receives to multiple output plugins. If one output blocks, copy will block as well.

### Configuration Fields

| Field     | Default  | Description                        |
| ---       | ---      | ---                                |
| `id`      | required | A unique identifier for the plugin |
| `outputs` | required | A list of IDs to write entries to  |

### Example Configurations

#### Simple multiple outputs

Configuration:
```yaml
- id: my_copy
  type: copy
  outputs:
    - copy_receiver1
    - copy_receiver2
```
