# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.3] - 2020-07-02
### Added
- Kubernetes metadata decorator plugin `k8s_metadata_decorator` (#149)

### Changed
- Custom plugin metadata format enhanced to allow title, description, and richer parameter descriptions (#156)
- Various performance improvements (#151)

### Fixed
- Builtin plugins writing to multiple outputs can be included in custom plugins (#152)


## [0.2.0] - 2020-06-26
### Added
- Builtin plugins can now write to multiple outputs without the copy plugin (#140)
- Severity parsing and a top level severity field (#137)
- Releases now include version-pinned install scripts (#125)
- More complete timestamp parsing (#111)
- File input plugin `start_at` parameter to configure starting from the end of the file (#108)
- Metadata plugin and top-level labels and tags (#86)
- Embedded expression syntax, currently only for metadata plugin (#107)
- Configurable `max_log_size` for file input plugin
### Changed
- Parsing the config fails on unknown fields (#127)
- Offset tracking is now opt-in by including the --database flag, which is used by default in the service installations (#126)
### Removed
- `google_cloud_output` plugin no longer accepts `labels_field` (#142) or `severity_field` (#144)
### Fixed
- Syslog plugin now uses current year rather than 1970 for rfc3164 parsing (#110)
