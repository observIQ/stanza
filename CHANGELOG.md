# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased
### Added
- Link `carbon` into `/usr/local/bin` so it's available on most users' `PATH` ([PR28](https://github.com/observIQ/carbon/pull/28))

## [0.9.1] - 2020-07-13
### Added
- More specific warning and error messages for common configuration issues ([PR12](https://github.com/observIQ/carbon/pull/12),[PR13](https://github.com/observIQ/carbon/pull/13),[PR14](https://github.com/observIQ/carbon/pull/14))
### Fixed
- Writing from files being actively written to will sometimes read partial entries ([PR21](https://github.com/observIQ/carbon/pull/21))
- Minor documentation omissions

## [0.9.0] - 2020-07-07
### Added
- Initial open source release. See documentation for full list of supported features in this version.
