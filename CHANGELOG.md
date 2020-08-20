# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.9.10] - 2020-08-20
### Added
- The `Resource` field was added to Entry ([PR95](https://github.com/observIQ/carbon/pull/95))
- The `Identifier` helper was created to assist with writing to `Resource` ([PR95](https://github.com/observIQ/carbon/pull/95))

### Removed
- The `Tags` field was removed from Entry ([PR95](https://github.com/observIQ/carbon/pull/95))

### Changed
- The `host_metadata` operator now writes to an entry's `Resource` field, instead of Labels
- The `host_labeler` helper has been renamed `host_identifier`
- The `metadata` operator embeds the `Identifier` helper and supports writing to `Resource`
- Input operators embed the `Identifier` helper and support writing to `Resource`
- The `k8s_event` operator now supports the `write_to`, `labels`, and `resource` configuration options
- Multiline for `file_input` now supports matching on new lines characters ([PR96](https://github.com/observIQ/carbon/pull/96))

## [0.9.9] - 2020-08-14
### Added
- Kubernetes events input operator ([PR88](https://github.com/observIQ/carbon/pull/88))
### Fixed
- Small improvements to test stability
- Fallback to reflection to convert entries to Google Cloud log entries ([PR93](https://github.com/observIQ/carbon/pull/93))

## [0.9.8] - 2020-08-12
### Fixed
- Google Cloud Output failure when sent a field of type uint16 ([PR82](https://github.com/observIQ/carbon/pull/82))
### Added
- Added a default function to plugin templates ([PR84](https://github.com/observIQ/carbon/pull/84))
- Add a host metadata operator that adds hostname and IP to entries ([PR85](https://github.com/observIQ/carbon/pull/85))
- Google Cloud Output option to enable gzip compression ([PR86](https://github.com/observIQ/carbon/pull/86))

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
- Configurable `timeout` parameter for the `k8s_metadata_decorator` ([PR54](https://github.com/observIQ/carbon/pull/54))
- Journald operator now supports `start_at` parameter ([PR55](https://github.com/observIQ/carbon/pull/55))

### Changed
- Enhanced plugin parameter metadata structure, to support required/optional and default values ([PR59](https://github.com/observIQ/carbon/pull/59))

### Fixed
- Issue where multiple instances of `syslog_parser` would cause parsing errors ([PR61](https://github.com/observIQ/carbon/pull/61))
- `short destination buffer` error now is handled by increasing encoding buffer size ([PR58](https://github.com/observIQ/carbon/pull/58))
- Issue where omitting the output field in a plugin could result in errors ([PR56](https://github.com/observIQ/carbon/pull/56))

## [0.9.4] - 2020-07-21
- Allow omitting `id`, defaulting to plugin type if unique within namespace
- Allow omitting `output`, defaulting to the next operator in the pipeline if valid

## [0.9.3] - 2020-07-20
### Added
- Support for multiple encodings in the file input plugin ([PR39](https://github.com/observIQ/carbon/pull/39))
- Install scripts and docker image now include plugins from `carbon-plugins` repository ([PR45](https://github.com/observIQ/carbon/pull/45))
- Publish image to dockerhub ([PR42](https://github.com/observIQ/carbon/pull/42))
- Improved default configuration ([PR41](https://github.com/observIQ/carbon/pull/41))
- Basic developer documentation ([PR43](https://github.com/observIQ/carbon/pull/43))
### Fixed
- JournalD emits `map[string]interface{}` ([PR38](https://github.com/observIQ/carbon/pull/38))

## [0.9.2] - 2020-07-13
### Added
- Link `carbon` into `/usr/local/bin` so it's available on most users' `PATH` ([PR28](https://github.com/observIQ/carbon/pull/28))
- New parameter `file_name_path` to the file input plugin for cases when just the file name is needed
### Changed
- Renamed `path_field` to `file_path_field` in the file input plugin
### Fixed
- Failure in Google Cloud Output to convert some data types to protocol buffers

## [0.9.1] - 2020-07-13
### Added
- More specific warning and error messages for common configuration issues ([PR12](https://github.com/observIQ/carbon/pull/12),[PR13](https://github.com/observIQ/carbon/pull/13),[PR14](https://github.com/observIQ/carbon/pull/14))
### Fixed
- Writing from files being actively written to will sometimes read partial entries ([PR21](https://github.com/observIQ/carbon/pull/21))
- Minor documentation omissions

## [0.9.0] - 2020-07-07
### Added
- Initial open source release. See documentation for full list of supported features in this version.
