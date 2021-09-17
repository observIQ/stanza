## `aws_cloudwatch_input` operator

The `aws_cloudwatch_input` operator reads logs from AWS Cloudwatch Logs using [AWS's SDK](https://github.com/aws/aws-sdk-go).

Fields `log_group`, `log_stream`,`region`, and `event_id` are promoted to resource field. The `Timestamp` field of the event is parsed as the entry's timestamp. 

Credentials are used in the following order.

- Environment Variables (Details [here](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html))
- Shared Credentials file (Details [here](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html)),
- Shared Configuration file (if SharedConfig is enabled details [here](https://docs.aws.amazon.com/sdkref/latest/guide/creds-config-files.html)) ,
- EC2 Instance Metadata (credentials only details [here](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-metadata.html)).

You can provide `profile` to specify which credential set to use from a Shared Credentials file.

### Configuration Fields

| Field                     | Default                | Description                                                                                                 |
| ---                       | ---                    | ---                                                                                                         |
| `id`                      | `aws_cloudwatch_input` | A unique identifier for the operator.                                                                       |
| `output`                  | Next in pipeline       | The connected operator(s) that will receive all outbound entries.                                           |
| `log_group_name`          |                        | The Cloudwatch Logs Log Group Name. Deprecated, use `log_groups` or `log_group_prefix`.                     |
| `log_groups`              |                        | List of Cloudwatch Log groups.                                                                              |
| `log_group_prefix`        |                        | Log group name prefix. This will detect any log group that starts with the prefix.                          |
| `region`                  | required               | The AWS Region to be used.                                                                                  |
| `log_stream_name_prefix`  |                        | The log stream name prefix to use. This will find any log stream name in the group with the starting prefix. Cannot be used with `log_stream_names` |
| `log_stream_names`        |                        | An array of log stream names to get events from. Cannot be used with `log_stream_name_prefix`               |
| `profile`                 |                        | Profile to use for authentication. Details on named profiles can be found [here](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html) |
| `event_limit`             | `10000`                | The maximum number of events to return per call.                                                            |
| `poll_interval`           | `1m`                   | The duration between event calls.                                                                           |
| `start_at`                | `end`                  | At startup, where to start reading events. Options are `beginning` or `end`                                 |

### Log Stream Name Prefix

The log_stream_prefix allows the use of "directives" such as `%Y` (4-digit year) and `%d` (2-digit zero-padded day). These directives are based on `strptime` directives. There are a limited set of the `strptime` directives. These directives are listed below. When directive is detected within the prefix it will replace the first occurance of directive with the data indicated in the description.

#### Supported directives

| Directive | Description                        |
| :---:     | :---                               |
| %Y        | Year, zero-padded                  |
| %y        | Year, last two digits, zero-padded |
| %m        | Month, zero-padded                 |
| %q        | Month as a unpadded number         |
| %b        | Abbreviated month name             |
| %h        | Abbreviated month name             |
| %B        | Full month name                    |
| %d        | Day of the month, zero-padded      |
| %g        | Day of the month, unpadded         |
| %a        | Abbreviated weekday name           |
| %A        | Full weekday name                  |

### Example Configurations

#### Simple AWS Cloudwatch Logs Example Input

Configuration:

```yaml
pipeline:
- type: aws_cloudwatch_input
  LogGroupName: "/aws/lambda/service"
  Region: us-east-2
```

### Simple AWS Cloudwatch Logs Example Output

```json
{
  "timestamp": "2021-05-10T13:00:55.023-04:00",
  "severity": 0,
  "record": {
    "event_id": "36142060744975733945009868546041203920891749688822923267",
    "ingestion_time": 1620666055330,
    "log_stream_name": "2021/05/10/[$LATEST]ff09d08f2836494690a1bd6b77365502",
    "message": "REPORT RequestId: 291fe36c-116a-42fd-a563-a8615671bab9\tDuration: 4577.28 ms\tBilled Duration: 4578 ms\tMemory Size: 128 MB\tMax Memory Used: 68 MB\tInit Duration: 401.54 ms\t\n"
  }
}
```

#### Log Stream Prefix Directives Example Input

Configuration:

```yaml
pipeline:
- type: aws_cloudwatch_input
  log_group_name: "/aws/lambda/service"
  region: us-east-2
  log_stream_name_prefix: "%Y/%m/%d"
```

### Log Stream Prefix Directives Example Output

```json
{
  "timestamp": "2021-05-12T13:03:47.941-04:00",
  "severity": 0,
  "resource": {
    "event_id": "36145918169946098276207227425947415203911741965970309123",
    "log_group": "/aws/lambda/service",
    "log_stream": "2021/05/12/[$LATEST]0f36de8f623a491c9305990130201669",
    "region": "us-east-2"
  },
  "record": {
    "ingestion_time": 1620839035104,
    "message": "REPORT RequestId: d64685ba-913b-456f-acd7-d00021416e68\tDuration: 1852.30 ms\tBilled Duration: 1853 ms\tMemory Size: 128 MB\tMax Memory Used: 68 MB\t\n"
  }
}
```

#### Log Stream Names Example Input

Configuration:

```yaml
pipeline:
- type: aws_cloudwatch_input
  log_group_name: "/aws/lambda/service"
  region: us-east-2
  log_stream_names:
    - "2021/05/09/[$LATEST]62e990bb0e72460c95b1dcfc5d96adc5"
    - "2021/05/08/[$LATEST]84d663604b6845e987d278272455ed95"
```

### Log Stream Names Example Output

```json
{
  "timestamp": "2021-05-09T13:04:02.686-04:00",
  "severity": 0,
  "resource": {
    "event_id": "36140138145615327091042663253954182481286730645124743171",
    "log_group": "/aws/lambda/service",
    "log_stream": "2021/05/09/[$LATEST]62e990bb0e72460c95b1dcfc5d96adc5",
    "region": "us-east-2"
  },
  "record": {
    "ingestion_time": 1620579849837,
    "message": "REPORT RequestId: 346b9fa2-9117-4d41-89f8-071f0100213b\tDuration: 1865.27 ms\tBilled Duration: 1866 ms\tMemory Size: 128 MB\tMax Memory Used: 68 MB\t\n"
  }
}
```

#### Log Group Name, Log Groups, Log Group Prefix

`log_group_prefix`, `log_groups`, and `log_group_name` can be combined.

Configuration

```yaml
pipeline:
- type: aws_cloudwatch_input
  region: us-east-2
  log_group_prefix: "/aws"
  log_group_name: /aws/rds/instance/backend/postgresql
  log_groups:
  - /aws/eks/arm64/cluster
```