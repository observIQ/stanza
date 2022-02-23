## `count_output` operator

The `count_output` operator prints lines of encoded json to stdout or a file detailing the number of entries the output operator has gotten since stanza started running.

Count Information has this current JSON representation
```json
{
    "entries": <number of entries this operator has received>,
    "elapsedMinutes": <number of minutes stanza has been running since the start of this operator>,
    "entries/minute": <number of entries per minute the output operator received>,
    "timestamp": <current time that this message is being recorded formatted in RFC 3339>
}
```

### Configuration Fields

| Field      | Default        | Description                                                                                                      |
| ---------- | -------------- | ---------------------------------------------------------------------------------------------------------------- |
| `id`       | `count_output` | A unique identifier for the operator                                                                             |
| `path`     |                | A file path to write the count information. If no path is provided then count information is outputted to stdout |
| `duration` | `1m`           | The frequency of when to output the count information                                                            |

### Example Configurations

Configuration

```yaml
pipeline:
  - type: generate_input
    count: 500
  - type: count_output
```

#### Counting 500 generated lines printed to stdout:

`./stanza -c ./config.yaml`

```json
{"level":"info","timestamp":"2021-08-20T20:09:55.057-0400","message":"Starting stanza agent"}
{"level":"info","timestamp":"2021-08-20T20:09:55.057-0400","message":"Stanza agent started"}
{"entries":500,"elapsedMinutes":2,"entries/minute":250, "timestamp":"2021-08-20T20:09:55.057-0400"}
```

#### Configuration going to file:
```yaml
pipeline:
  - type: generate_input
    count: 500
  - type: count_output
    path: ./count.json
```

`./stanza -c ./config.yml`
> no output
```json
{"level":"info","timestamp":"2021-08-20T20:09:28.314-0400","message":"Starting stanza agent"}
{"level":"info","timestamp":"2021-08-20T20:09:28.314-0400","message":"Stanza agent started"}
```

Printing out results of specified file:
```sh
> cat count.json | jq 
```
```json
{
  "entries": 500,
  "elapsedMinutes": 1,
  "entries/minute": 500,
  "timestamp": "2021-08-20T20:09:28.314-0400"
},
{
  "entries": 500,
  "elapsedMinutes": 2,
  "entries/minute": 250,
  "timestamp": "2021-08-20T20:09:29.414-0400
}
```