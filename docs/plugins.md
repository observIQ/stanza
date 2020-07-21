# Plugins

Plugins can be defined by using a file that contains a templated set of operators.

For example, a very simple plugin for monitoring Apache Tomcat access logs could look like this:
`tomcat.yaml`:
```yaml
---
pipeline:
  - type: file_input
    include:
      - {{ .path }}

  - type: regex_parser
    output: {{ .output }}
    regex: '(?P<remote_host>[^\s]+) - (?P<remote_user>[^\s]+) \[(?P<timestamp>[^\]]+)\] "(?P<http_method>[A-Z]+) (?P<path>[^\s]+)[^"]+" (?P<http_status>\d+) (?P<bytes_sent>[^\s]+)'
```

Once a plugin config has been defined, it can be used in the carbon config file with a `type` matching the filename of the plugin.

`config.yaml`:
```yaml
---
pipeline:
  - type: tomcat
    path: /var/log/tomcat/access.log

  - type: stdout
```

The `tomcat_access` plugin is replaced with the operators from the rendered config in `tomcat.yaml`.

## Building a plugin

Building a plugin is as easy as pulling out a set of operators in a working configuration file, then templatizing it with
any parts of the config that need to be treated as variable. In the example of the Tomcat access log plugin above, that just means
adding variables for `path` and `output`.

Plugins use Go's [`text/template`](https://golang.org/pkg/text/template/) package for template rendering. All fields from
the plugin configuration are available as variables in the templates except the `type` field.

For carbon to discover a plugin, it needs to be in the `plugins` directory. This can be customized with the
`--plugin_dir` argument. For a default installation, the plugin directory is located at `$CARBON_HOME/plugins`.
