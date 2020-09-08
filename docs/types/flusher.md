# Flushers

Flushers handle reading entries from buffers in chunks, flushing them to their final destination,
and retrying on failure.

## Flusher configuration

Flushers are configured with the `flusher` block on output plugins.

| Field               | Default | Description                                                                                                                                   |
| ---                 | ---     | ---                                                                                                                                           |
| `max_concurrent`    | `16`    | The maximum number of goroutines flushing entries concurrently                                                                                |
| `max_wait`          | 1s      | The maximum amount of time to wait for a chunk to fill before flushing it. Higher values can reduce load, but also increase delivery latency. |
| `max_chunk_entries` | 1000    | The maximum number of entries to flush at in a single chunk.                                                                                  |
