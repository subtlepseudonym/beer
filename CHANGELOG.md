# Changelog
## [0.3.1]
### Changed
- Use formatted output for Pour JSON fields

### Fixed
- Fixed pour event counting
- Panic during pour pruning

## [0.3.0] - 2023-02-19
### Added
- Multi-platform docker builds

### Changed
- Moved individual pour data to separate endpoint

### Fixed
- DHT measurement updates
- Pour volume metric calculation
- Remaining volume metric calculation

## [0.2.1] - 2023-02-17
### Added
- Remaining volume prometheus metric

## [0.2.0] - 2023-02-16
### Added
- Load state from file on start up
- Save state to file periodically
- Prevent autosaving to file with --no-autosave

### Changed
- Rename to "kegerator" from "beer"

## [0.1.0] - 2023-02-15
### Added
- Support for DHT sensors
- REST endpoint for sensor and keg state
- Prometheus metrics
- Ability to read state from JSON file

### Changed
- Overhauled POC into useable service

