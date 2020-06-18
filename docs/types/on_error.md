# `on_error` parameter
The `on_error` parameter determines the error handling strategy a parser should use when it fails to parse an entry. There are 3 supported values: `fail`, `drop`, and `ignore`. 

Regardless of the method selected, all parsing errors will be logged.

### `fail`
In this mode, any entries that fail to parse will be dropped. The calling plugin will be notified of the error, so that it can treat the failure as blocking.

### `drop`
In this mode, any entries that fail to parse will be dropped. However, the calling plugin will not be notified of the error. This will stop the failure from blocking future entries.

### `ignore`
In this mode, any entries that fail to parse will be sent to the connected plugin in their pre-parsed state. This will allow entries to proceed to their final destination, even if a parser error has occurred.
