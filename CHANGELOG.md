# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.3] - 2020-07-02
### Added
- Kubernetes metadata decorator plugin `k8s_metadata_decorator`

### Changed
- Custom plugin metadata format enhanced to allow title, description, and richer parameter descriptions
- The file input plugin's `start_at` parameter now defaults to `end`
- Various performance improvements

### Fixed
- Builtin plugins writing to multiple outputs can be included in custom plugins


## [0.2.0] - 2020-06-26
### Added
- Builtin plugins can now write to multiple outputs without the copy plugin
- Severity parsing and a top level severity field
- Releases now include version-pinned install scripts
- More complete timestamp parsing
- File input plugin `start_at` parameter to configure starting from the end of the file
- Metadata plugin and top-level labels and tags
- Embedded expression syntax, currently only for metadata plugin
- Configurable `max_log_size` for file input plugin
### Changed
- Parsing the config fails on unknown fields
- Offset tracking is now opt-in by including the --database flag, which is used by default in the service installations
### Removed
- `google_cloud_output` plugin no longer accepts `labels_field` or `severity_field`
### Fixed
- Syslog plugin now uses current year rather than 1970 for rfc3164 parsing
