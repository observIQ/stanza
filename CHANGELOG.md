# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added
- Builtin plugins can now write to multiple outputs without the copy plugin (#140)
- Severity parsing and a top level severity field (#137)
- Releases now include version-pinned install scripts (#125)
- More complete timestamp parsing (#111)
- File input plugin `start_at` parameter to configure starting from the end of the file (#108)
- Metadata plugin and top-level labels and tags (#86)
- Embedded expression syntax, currently only for metadata plugin (#107)
- Configurable max_log_size for file input plugin
### Changed
- Parsing the config fails on unknown fields (#127)
- Offset tracking is now opt-in by including the --database flag, which is used by default in the service installations (#126)
### Fixed
- Syslog plugin now uses current year rather than 1970 for rfc3164 parsing (#110)
