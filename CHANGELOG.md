# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased

### Added

### Changed

## 1.6.0

### Added 

- Adds `count_output` operator [PR570](https://github.com/observIQ/stanza/pull/570)
- Add ShiftJIS to supported encodings [PR546](https://github.com/observIQ/stanza/pull/546)

## 1.3.0

### Added

- Added the `xml_parser` operator [PR482](https://github.com/observIQ/stanza/pull/482)

## 1.2.13 - 2021-10-29

### Added

- Added the `lazy_quotes` parameter to the csv parser [PR472](https://github.com/observIQ/stanza/pull/472)

### Removed

- Removed OTLP output operator [PR470](https://github.com/observIQ/stanza/pull/470)

## 1.2.10 - 2021-10-04

### Fixed

- Fixed an issue where TLS configuration fails to enable TLS: [PR466](https://github.com/observIQ/stanza/pull/466)
- Fixed an incorrect sync call when saving offsets: [PR465](https://github.com/observIQ/stanza/pull/465)
- Windows Event Log operator fix for Windows Server 2022: [PR456](https://github.com/observIQ/stanza/pull/456)

## 1.2.9 - 2021-09-29

### Fixed

Fixed a bug when flushing a range from the memory buffer: [PR455](https://github.com/observIQ/stanza/pull/455)

## 1.2.8 - 2021-09-28

This release includes [Stanza Plugins v0.0.82](https://github.com/observIQ/stanza-plugins/releases/tag/v0.0.82)

## 1.2.7 - 2021-09-22

### Added

- Key Value Parser: [PR426](https://github.com/observIQ/stanza/pull/426)
  - Parse `key=value` pairs

### Changed

- Google Output: entry.Resources are now mapped as labels, because Google Cloud Logging does not support custom resources [PR425](https://github.com/observIQ/stanza/pull/425)
- File Input: Optimize excluded file detection [PR444](https://github.com/observIQ/stanza/pull/444)
  - Significant startup time reduction when reading from a directory with 50,000+ files
- File Input: Optimize `delete_after_read`
  - Do not store deleted files in database: https://github.com/observIQ/stanza/pull/442
  - Added filename_recall_period parameter: https://github.com/observIQ/stanza/pull/440

### Fixed

- TCP / UDP Input: Resolve panic when closing nil connection: [PR437](https://github.com/observIQ/stanza/pull/437)
- Cloudwatch Input: Do not store pointers on the entry [PR445](https://github.com/observIQ/stanza/pull/445)
  - Resolves an issue where an expression cannot be used against an entry from Cloudwatch input
- File Input: Resolve issue where doublestar does not correctly detect files [PR433](https://github.com/observIQ/stanza/pull/433)
  - Ported from otel: https://github.com/open-telemetry/opentelemetry-log-collection/pull/268

## 1.2.6 - 2021-09-14

### Changed

- Upgrade from Go 1.16 to 1.17
- AWS Cloudwatch Input: Added abilty to monitor more than one log group [PR 420](https://github.com/observIQ/stanza/pull/420)
- File Input: Changed default max files from 1024 to 512 [PR 423](https://github.com/observIQ/stanza/pull/423)

## 1.2.5 - 2021-09-13

Use plugins version 0.0.76

## 1.2.4 - 2021-09-10

### Added

- File Input: Added optional delete_after_read parameter [PR 417](https://github.com/observIQ/stanza/pull/417)
  - Useful for cleaning up files after they are read. Important when reading from a directory that is constantly adding new files and never modifying old files.
  - Can only be used with `start_at: beginning`

### Changed

- Plugins Version [0.0.75](https://github.com/observIQ/stanza-plugins/releases/tag/v0.0.75)

## 1.2.3 - 2021-09-08

### Fixed

Resolved an issue where journald will omit logs above 4096 bytes [PR 414](https://github.com/observIQ/stanza/pull/414)

## 1.2.2 - 2021-09-02

### Changed

Added debug logging to journald_input and google_cloud_output operators [PR 413](https://github.com/observIQ/stanza/pull/413)

## 1.2.1 - 2021-08-25

### Changed

This release includes Stanza Plugins 0.0.72, which includes the new [W3C plugin](https://github.com/observIQ/stanza-plugins/pull/307)

### Fixed

- ARM64 container image release process [PR 407](https://github.com/observIQ/stanza/pull/407)
- CI benchmark failures [PR 408](https://github.com/observIQ/stanza/pull/408)

## 1.2.0 - 2021-08-24

### Changed

- File Input: Added optional LabelRegex parameter, for parsing log file headers as labels [PR 376](https://github.com/observIQ/stanza/pull/376)
- CSV Parser: Dynamic field names [PR 404](https://github.com/observIQ/stanza/pull/404)
- ARM64 Container Image: [PR 381](https://github.com/observIQ/stanza/pull/381)
- TCP Input: Minimum TLS version is now configurable: [PR 400](https://github.com/observIQ/stanza/pull/400)
- Systemd service: Set `TimeoutSec` [PR 402](https://github.com/observIQ/stanza/pull/402)
- Updated dependencies:
  - go.uber.org/multierr [PR 387](https://github.com/observIQ/stanza/pull/387)
  - go.etcd.io/bbolt [PR 385](https://github.com/observIQ/stanza/pull/385)
  - k8s client [PR 377](https://github.com/observIQ/stanza/pull/377)
    - k8s.io/api
    - k8s.io/apimachinery
    - k8s.io/client-go
  - github.com/golangci/golangci-lint [PR 382](https://github.com/observIQ/stanza/pull/382)
  - cloud.google.com/go/logging [PR 394](https://github.com/observIQ/stanza/pull/394)
  - google.golang.org/grpc [PR 383](https://github.com/observIQ/stanza/pull/383)
  - github.com/aws/aws-sdk-go [PR 395](https://github.com/observIQ/stanza/pull/395)
  - golang.org/x/text  [PR 386](https://github.com/observIQ/stanza/pull/386)
  - github.com/antonmedv/expr [PR 396](https://github.com/observIQ/stanza/pull/396)


## 1.1.8 - 2021-08-19

### Changed

- Docker base image switch to [stanza base image](https://github.com/observIQ/stanza-base-image) [PR 393](https://github.com/observIQ/stanza/pull/393)
  - 100MB image size reduction
  - Pinned package versions

### Fixed

- Resolved an issue where log type is not set correctly when using kubernetes_container plugin and Google output [PR 392](https://github.com/observIQ/stanza/pull/392)

## 1.1.7 - 2021-08-19

### Added
- Enabled [flatten operator](https://github.com/observIQ/stanza/blob/master/docs/operators/flatten.md) [PR 390](https://github.com/observIQ/stanza/pull/390)

### Fixed
- Resolved journald bug introduced in previous patch release (1.1.6) [PR 389](https://github.com/observIQ/stanza/pull/389)

## 1.1.6 - 2021-08-17

### Added
- File input: Added optional labels for resolved symlink file name and path [PR 364](https://github.com/observIQ/stanza/pull/364)
- CSV Parser: Added optional configuration field `header_delimiter` [PR 370](https://github.com/observIQ/stanza/pull/370)

### Changed
- Journald input: Switched from long running process to polling strategy [PR380](https://github.com/observIQ/stanza/pull/380)

## 1.1.5 - 2021-07-15

### Changed
- Goflow now includes a string representation of the `proto` field as `proto_name` [PR 359](https://github.com/observIQ/stanza/pull/359)
- Goflow parse function refactored for significant performance increase [PR 361](https://github.com/observIQ/stanza/pull/361)

### Fixed
- Goflow zero values (such as `proto`) are no longer ommited as they are valid values [PR 361](https://github.com/observIQ/stanza/pull/361)
  - `proto`: 0 represents `proto_name`: HOPOPT

## 1.1.4 - 2021-07-08

### Added
- Added [license scanner](https://github.com/uw-labs/lichen) to CI [PR 347](https://github.com/observIQ/stanza/pull/347)
- Added [Gosec](https://github.com/securego/gosec) to CI [PR 358](https://github.com/observIQ/stanza/pull/358)

### Fixed
- Fixed file input issue that resulted in minor dataloss when log files are rotating symlinks 
  - Based on Open Telemetry implementation
    - [Origonal Issue](https://github.com/open-telemetry/opentelemetry-log-collection/issues/85)
    - [Open Telemetry Log Collection PR 182](https://github.com/open-telemetry/opentelemetry-log-collection/pull/182)
  - https://github.com/observIQ/stanza/pull/346
  - https://github.com/observIQ/stanza/pull/345
  - https://github.com/observIQ/stanza/pull/344
  - https://github.com/observIQ/stanza/pull/343
- Resolved [Gosec](https://github.com/securego/gosec) suggestions
  - https://github.com/observIQ/stanza/pull/338
  - https://github.com/observIQ/stanza/pull/357

### Changed
- K8s daemonset example refreshed [PR 348](https://github.com/observIQ/stanza/pull/348)

## 1.1.3 - 2021-06-30

No changes, releasing for [Stanza Plugins v0.0.66](https://github.com/observIQ/stanza-plugins/releases/tag/v0.0.66)

## 1.1.2 - 2021-06-24

### Fixed
- Resolved an issue where empty ip address fields result in failed parsing [PR 336](https://github.com/observIQ/stanza/pull/336)

## 1.1.1 - 2021-06-21

### Fixed
- Log error returned by publisher.Open in `operator/builtin/input/windows/operator.go` [PR 334](https://github.com/observIQ/stanza/pull/334)

## 1.1.0 - 2021-06-18

### Added
- Added Goflow operator for receiving Netflow (v5, v9, ipfix) and Sflow [PR 332](https://github.com/observIQ/stanza/pull/332)

## 1.0.1 - 2021-06-16

### Fixed
- Fixed panic during shutdown when Google Cloud Output credential file not found [Issue 264](https://github.com/observIQ/stanza/issues/264)
- Fixed bug where logs can be duplicated when a parser has on_error=send [PR 330](https://github.com/observIQ/stanza/pull/330)

## [1.0.0] - 2021-05-27

### Changed
- Stanza is now a single module [PR304](https://github.com/observIQ/stanza/pull/304)

## [0.14.2] - 2021-05-24

### Changed
- Make buffer max chunk delay reconfigurable on the fly [PR313](https://github.com/observIQ/stanza/pull/313)

## [0.14.1] - 2021-05-20

### Added
- Added optional network metadata labels to tcp / udp operators [PR302](https://github.com/observIQ/stanza/pull/302)
- Added AWS Cloudwatch Logs input operator [PR289](https://github.com/observIQ/stanza/pull/289)

## [0.14.0] - 2021-05-07

### Added
- Added Move operator [PR271](https://github.com/observIQ/stanza/pull/271)
- Added Add operator [PR272](https://github.com/observIQ/stanza/pull/272)
- Added Remove operator [PR273](https://github.com/observIQ/stanza/pull/273)
- Added Copy operator [PR278](https://github.com/observIQ/stanza/pull/278)
- Added Retain operator [PR279](https://github.com/observIQ/stanza/pull/279)
- Added Flatten operator [PR286](https://github.com/observIQ/stanza/pull/286)

### Changed
- Renamed Azure Event Hub event_data field to message [PR297](https://github.com/observIQ/stanza/pull/297)
- Added doublestar support to File Input [PR283](https://github.com/observIQ/stanza/pull/283)

### Fixed
- Fixed TCP Input Operator panic [PR296](https://github.com/observIQ/stanza/pull/296)
- Fixed Syslog parser race condition [PR284](https://github.com/observIQ/stanza/pull/284)

## [0.13.20] - 2021-05-06 

### Added
- Added flatten Operator [PR 286](https://github.com/observIQ/stanza/pull/286)
- Added Azure Event Hub Operator [PR 287](https://github.com/observIQ/stanza/pull/287)
- Added Azure Log Analytics Operator [PR 287](https://github.com/observIQ/stanza/pull/287)

## [0.13.19] - 2021-04-15
 
### Added
- Added float64 to Severity parser's supported types [PR 267](https://github.com/observIQ/stanza/issues/267)

### Changed
- Switched to Go 1.16, from Go 1.14
- Updated syslog operator to v0.1.5

## [0.13.18] - 2021-04-02

### Changed
- Google Output will split batched entries if the batch is too large [PR 263](https://github.com/observIQ/stanza/pull/263)

### Fixed
- Issue where Google Output does not drop entries that are too big [issue 257](https://github.com/observIQ/stanza/issues/257)
- Issue where partially successful flushes were treated as fully successful, [operator/buffer/memory.go](https://github.com/observIQ/stanza/pull/263/commits/6419445588062bcf8bae84aa05fa2f1b28dbdd44)

## [0.13.17] - 2021-03-17

### Added
- Added new operator `csv_parser`

## [0.13.16] - 2021-01-01
- Added optional `max_buffer_size` parameter to tcp input operator

## [0.13.15] - 2021-02-26
- Same as 0.13.14, but released with [plugins v0.0.48](https://github.com/observIQ/stanza-plugins/releases/tag/v0.0.48)
  - Adds TLS support to `vmware_vcenter` and `vmware_esxi`

## [0.13.14] - 2021-02-25

### Changed
- Added TLS support to tcp input operator [pr253](https://github.com/observIQ/stanza/pull/253)

## [0.13.13] - 2021-02-18

### Added
- uri_parser operator for parsing [absolute uri, relative uri, and uri query strings](https://tools.ietf.org/html/rfc3986)
- container image: added package [tzdata](https://github.com/observIQ/stanza/pull/245)

### Changed
- Added optional `location` parameter to Syslog operator [pr247](https://github.com/observIQ/stanza/pull/247)
- Updated Google Cloud output version to v0.1.2 [pr250](https://github.com/observIQ/stanza/pull/250)

## [0.13.12] - 2020-01-26

### Changed
- Allow plugin parameters to have a default value even if they are required

## [0.13.11] - 2020-01-15

### Changed
- Updated version of stanza used in several isolated modules

## [0.13.10] - 2020-01-15

### Added
- `timestamp` parser now supports a `location` parameter

## [0.13.9] - 2020-01-04

### Fixed
- `k8s_metadata_decorator` using a proxy causes internal API timeout

## [0.13.8] - 2020-12-30
### Fixed
- `file_input` exclude processing could result in extra exclusions

## [0.13.7] - 2020-12-23
### Added
- Ability to customize `file_input`'s `fingerprint_size`
## [0.13.6] - 2020-12-18
### Fixed
- Issue where timestamps ending 'Z' were not treated as UTC
- Issue where recognized timezones may not properly calculate offsets
- Issue where `file_output` would escape html special characters

## [0.13.5] - 2020-12-09
### Fixed
- Issue where flushers would retry indefinitely
- Issue where flushers would improperly reuse the same http request multiple times

## [0.13.4] - 2020-12-07
### Added
- Recombine operator to combine multiline logs after ingestion and parsing

### Fixed
- Issue where entries skipped by `if` would be output twice

## [0.13.3] - 2020-12-01
### Added
- New operators `forward_output` and `forward_input` to easily send log entries between stanza instances.
- Override default timestamp with `STANZA_DEFAULT_TIMESTAMP` for integration testing
- Add new `bytesize` type for easier configuration of byte sizes
- Automatic severity promotion in the syslog parser
### Fixed
- Open files in chunks so that we don't hit open file limit and cause performance issues

## [0.13.2] - 2020-11-17
### Added
- New parameter `if` to parser plugins to allow for easy conditional parsing without routers
- New `default` parameter to the router to explicitly send unmatched entries to a specific operator(s)

## [0.13.1] - 2020-11-11
### Fixed
- Missing default configuration of `elastic_output` flusher
### Changed
- A plugin that fails to parse will now log an error, but will not cause stanza to fail to start
### Added
- New `stdin` operator

## [0.13.0] - 2020-11-09
### Added
- OTLP severity level recognition
- Severity Text field on Entry
### Changed
- Removed `preserve` in favor of `preserve_to` to make it more clear that it may overwrite parsed fields
- Updated our internal log sampling numbers to more aggressively sample repeated logs
### Added
- Log message whenever a new file is detected

## [0.12.5] - 2020-10-07
### Added
- `windows_eventlog_input` can now parse messages from the Security channel.

## [0.12.4] - 2020-10-07
### Fixed
- Router outputs were not namespaced correctly

## [0.12.3] - 2020-10-07
### Fixed
- (De)serialization of JSON for plugin config structs

## [0.12.2] - 2020-10-06
### Added
- New Relic Logs output operator
- Additional resource values with parent object names (service name, replica set name, etc.) in the k8s metadata operator
- Publicly available `version.GetVersion()` for consumers of the `stanza` module

## [0.12.0] - 2020-09-21
### Changed
- Most operators are no longer part of dedicated modules

## [0.11.0] - 2020-09-15
### Changed
- File input improvements and rotation tests

## [0.10.0] - 2020-09-11
### Added
- Disk buffer for output operators ([PR109](https://github.com/observIQ/stanza/pull/109))
### Changed
- Split buffers into buffers and flushers for better modularity ([PR109](https://github.com/observIQ/stanza/pull/109))
- New memory buffer design for a uniform interface between disk and memory buffers ([PR109](https://github.com/observIQ/stanza/pull/109))
- Most operators are now dedicated modules, so that they may be imported individually ([PR108](https://github.com/observIQ/stanza/pull/108))

## [0.9.14] - 2020-08-31
### Fixed
- Rendering issue with the `kubernetes_events` plugin

## [0.9.13] - 2020-08-31
### Added
- Support for accessing the resource with fields ([PR105](https://github.com/observIQ/stanza/pull/105))
- Support for using fields to select keys that contain dots like `$record['field.with.dots']` ([PR105](https://github.com/observIQ/stanza/pull/105))
- `google_cloud_output` will use resource create a monitored resource for supported resource types (currently only k8s resources) ([PR105](https://github.com/observIQ/stanza/pull/105))
### Changed
- The operators `host_metadata`, `k8s_event_input`, and `k8s_metadata_decorator` will now use the top-level resource field ([PR105](https://github.com/observIQ/stanza/pull/105))
- `k8s_metadata_decorator` now generates pod labels that match those generated by GKE ([PR105](https://github.com/observIQ/stanza/pull/105))
### Fixed
- Issue with `k8s_event_input` generating entries with zero-valued time ([PR105](https://github.com/observIQ/stanza/pull/105))
- Plugin ID in templates will now correctly default to the plugin type if unset ([PR105](https://github.com/observIQ/stanza/pull/105))


## [0.9.12] - 2020-08-25
### Changed
- Agent is now embeddable with a default output

## [0.9.11] - 2020-08-24
### Added
- The 'filter' operator

### Changed
- Renamed project to `stanza`
- Move `testutil` package out of `internal`

## [0.9.10] - 2020-08-20
### Added
- The `Resource` field was added to Entry ([PR95](https://github.com/observIQ/stanza/pull/95))
- The `Identifier` helper was created to assist with writing to `Resource` ([PR95](https://github.com/observIQ/stanza/pull/95))

### Removed
- The `Tags` field was removed from Entry ([PR95](https://github.com/observIQ/stanza/pull/95))

### Changed
- The `host_metadata` operator now writes to an entry's `Resource` field, instead of Labels
- The `host_labeler` helper has been renamed `host_identifier`
- The `metadata` operator embeds the `Identifier` helper and supports writing to `Resource`
- Input operators embed the `Identifier` helper and support writing to `Resource`
- The `k8s_event` operator now supports the `write_to`, `labels`, and `resource` configuration options
- Multiline for `file_input` now supports matching on new lines characters ([PR96](https://github.com/observIQ/stanza/pull/96))

## [0.9.9] - 2020-08-14
### Added
- Kubernetes events input operator ([PR88](https://github.com/observIQ/stanza/pull/88))
### Fixed
- Small improvements to test stability
- Fallback to reflection to convert entries to Google Cloud log entries ([PR93](https://github.com/observIQ/stanza/pull/93))

## [0.9.8] - 2020-08-12
### Fixed
- Google Cloud Output failure when sent a field of type uint16 ([PR82](https://github.com/observIQ/stanza/pull/82))
### Added
- Added a default function to plugin templates ([PR84](https://github.com/observIQ/stanza/pull/84))
- Add a host metadata operator that adds hostname and IP to entries ([PR85](https://github.com/observIQ/stanza/pull/85))
- Google Cloud Output option to enable gzip compression ([PR86](https://github.com/observIQ/stanza/pull/86))

## [0.9.7] - 2020-08-05
### Changed
- In the file input operator, file name and path fields are now added with `include_file_name` (default `true`) and `include_file_path` (default `false`)
- Input and router operators can define labels on entries using the `labels` field
- Add Event ID to windows event log entries
- Use the `go-syslog` fork directly rather than relying on a `replace` directive so that the agent can be used as a library successfully

## [0.9.6] - 2020-08-04
### Changed
- Fork go-syslog to support long sdnames that are not rfc5424-compliant
- Reduce noise in debug messages for TCP and UDP inputs
### Added
- `log_type` label added by default to input operators
### Fixed
- Trim carriage returns from TCP input

## [0.9.5] - 2020-07-28
### Added
- Configurable `timeout` parameter for the `k8s_metadata_decorator` ([PR54](https://github.com/observIQ/stanza/pull/54))
- Journald operator now supports `start_at` parameter ([PR55](https://github.com/observIQ/stanza/pull/55))

### Changed
- Enhanced plugin parameter metadata structure, to support required/optional and default values ([PR59](https://github.com/observIQ/stanza/pull/59))

### Fixed
- Issue where multiple instances of `syslog_parser` would cause parsing errors ([PR61](https://github.com/observIQ/stanza/pull/61))
- `short destination buffer` error now is handled by increasing encoding buffer size ([PR58](https://github.com/observIQ/stanza/pull/58))
- Issue where omitting the output field in a plugin could result in errors ([PR56](https://github.com/observIQ/stanza/pull/56))

## [0.9.4] - 2020-07-21
- Allow omitting `id`, defaulting to plugin type if unique within namespace
- Allow omitting `output`, defaulting to the next operator in the pipeline if valid

## [0.9.3] - 2020-07-20
### Added
- Support for multiple encodings in the file input plugin ([PR39](https://github.com/observIQ/stanza/pull/39))
- Install scripts and docker image now include plugins from `stanza-plugins` repository ([PR45](https://github.com/observIQ/stanza/pull/45))
- Publish image to dockerhub ([PR42](https://github.com/observIQ/stanza/pull/42))
- Improved default configuration ([PR41](https://github.com/observIQ/stanza/pull/41))
- Basic developer documentation ([PR43](https://github.com/observIQ/stanza/pull/43))
### Fixed
- JournalD emits `map[string]interface{}` ([PR38](https://github.com/observIQ/stanza/pull/38))

## [0.9.2] - 2020-07-13
### Added
- Link `stanza` into `/usr/local/bin` so it's available on most users' `PATH` ([PR28](https://github.com/observIQ/stanza/pull/28))
- New parameter `file_name_path` to the file input plugin for cases when just the file name is needed
### Changed
- Renamed `path_field` to `file_path_field` in the file input plugin
### Fixed
- Failure in Google Cloud Output to convert some data types to protocol buffers

## [0.9.1] - 2020-07-13
### Added
- More specific warning and error messages for common configuration issues ([PR12](https://github.com/observIQ/stanza/pull/12),[PR13](https://github.com/observIQ/stanza/pull/13),[PR14](https://github.com/observIQ/stanza/pull/14))
### Fixed
- Writing from files being actively written to will sometimes read partial entries ([PR21](https://github.com/observIQ/stanza/pull/21))
- Minor documentation omissions

## [0.9.0] - 2020-07-07
### Added
- Initial open source release. See documentation for full list of supported features in this version.
