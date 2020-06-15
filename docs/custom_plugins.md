# Custom Plugins

Custom plugins are bplogagent's native solution for building curated, technology-specific sets of configurations.

## Overview

A custom plugin is a file that contains a templated set of builtin plugins.

For example, a very simple custom plugin for monitoring Apache Tomcat access logs could look like this:
`tomcat.yaml`:
```yaml
---
pipeline:
  - id: tomcat_access_reader
    type: file_input
    include:
      - {{ .path }}
    output: tomcat_regex_parser

  - id: tomcat_regex_parser
    type: regex_parser
    output: {{ .output }}
    regex: '(?P<remote_host>[^\s]+) - (?P<remote_user>[^\s]+) \[(?P<timestamp>[^\]]+)\] "(?P<http_method>[A-Z]+) (?P<path>[^\s]+)[^"]+" (?P<http_status>\d+) (?P<bytes_sent>[^\s]+)'
```

Once a custom plugin config has been created, it can be used in the log agent's config file with a `type` matching the filename of the custom plugin.

`config.yaml`:
```yaml
---
pipeline:
  - id: tomcat_access
    type: tomcat
    output: stdout
    path: /var/log/tomcat/access.log

  - id: stdout
    type: stdout
```

The `tomcat_access` plugin is replaced with the plugins from the rendered config in `tomcat.yaml`.

## Building a custom plugin

Building a custom plugin is as easy as pulling out a set of plugins in a working configuration file, then templatizing it with
any parts of the config that need to be treated as variable. In the example of the Tomcat access log plugin above, that just means
adding variables for `path` and `output`.

Custom plugins use Go's [`text/template`](https://golang.org/pkg/text/template/) package for template rendering. All fields from
the custom plugin configuration are available as variables in the templates except the `type` field.

For the log agent to discover a custom plugin, it needs to be in the log agent's `plugin` directory. This can be set with the
`--plugin_dir` argument. For a default installation, the plugin directory is located at `$BPLOGAGENT_HOME/plugins`.
