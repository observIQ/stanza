# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased
### Added
- Configurable `timeout` parameter for the `k8s_metadata_decorator`


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
