# Change Log
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).


## [Unreleased]

## [v0.1.0] - 2026-01-25
### Changed
- **Breaking**: Version management system now uses build-time injection
  - Version is now defined in VERSION file instead of hardcoded
  - Build process requires ldflags for version injection
  - Swagger documentation now uses dynamic versioning
- Refactored version handling into dedicated package (`pkg/version`)

### Added
- Makefile targets for version management (`make bump-version`, `make show-version`)
- Enhanced `/version` endpoint with build metadata

<!-- links -->
[v0.1.0]: https://github.com/boskuv/goreminder/releases/tag/v0.1.0