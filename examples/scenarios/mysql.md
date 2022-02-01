# MySQL

Once you have Stanza installed and running from the [quickstart guide](./README.md#quick-start), you can follow these steps to configure MySQL monitoring via Linux.

## Prerequisites

To enable certain log files, it may be necessary to edit the MySQL configuration file: mysqld.cnf.

### Audit Log
To enable MariaDB Audit logs, it is necessary to install the MariaDB Audit Plugin. More information can be found <a href="https://mariadb.com/kb/en/mariadb-audit-plugin-installation/" target="_blank">here</a>.

### Error Log
This is generally enabled by default. To change the file path, you can set or update "log_error" within mysqld.cnf.

For more details, see the error log documentation [here](https://dev.mysql.com/doc/refman/5.7/en/error-log.html ).

### Query Log
To enable the query log, set *general_log_file* to the desired log path and set *general_log = 1*.

For more details, see the query log documentation [here](https://dev.mysql.com/doc/refman/5.7/en/query-log.html).

### Slow Query Log
To enable the slow query log, set *slow_query_log_file* to the desired log path. Set *slow_query_log = 1* and optionally, configure *long_query_time*. 

For more details, see the slow query log documentation [here](https://dev.mysql.com/doc/refman/5.7/en/slow-query-log.html).

## Configuration

This is an example config file that can be used in the Stanza install directory. The MySQL plugin supports general, error, and slow query logs by default, but can also support MariaDB audit logs if those have been enabled.

```yaml
pipeline:
  # To see the MySQL plugin, go to: https://github.com/observIQ/stanza-plugins/blob/master/plugins/mysql.yaml
  - type: mysql
    enable_general_log: true
    general_log_path: "/var/log/mysql/general.log"
    enable_slow_log: true
    slow_query_log_path: "/var/log/mysql/slow.log"
    enable_error_log: true
    error_log_path: "/var/log/mysql/mysqld.log"
    enable_mariadb_audit_log: false
    mariadb_audit_log_path: "/var/log/mysql/audit.log"

  # For more info on Google Cloud output, go to: https://github.com/observIQ/stanza/blob/master/docs/operators/google_cloud_output.md
  - type: google_cloud_output
    credentials_file: /tmp/credentials.json
```
The output is configured to go to Google Cloud utilizing a credentials file that can be generated following Google's documentation [here](https://cloud.google.com/iam/docs/creating-managing-service-account-keys).

## Next Steps

- Learn more about [plugins](/docs/plugins.md).
- Read up on how to write a stanza [pipeline](/docs/pipeline.md).
- Check out stanza's list of [operators](/docs/operators/README.md).
- Check out the [FAQ](/docs/faq.md).
