# Changelog
## [0.4.1] -
### Fixed
- Initialize gpio memory in sensor-test
- Export human-readable state fields

## [0.4.0] - 2023-04-12
### Added
- Endpoint for calibrating flow meters
- Endpoint for refilling kegs
- Binary for testing sensors

### Changed
- Return list of pours rather than map in /pours
- Reset RemainingVolume prometheus metric on file reload
- Organized directory structure to enable multiple binaries

### Fixed
- Prevent DHT from exceeding temperature limit on startup
- Avoid over-counting volume dispensed on new pour
- Setting initial temperature metric
- Using incorrect edge type for flow meter interrupts

## [0.3.2] - 2023-02-28
### Added
- Erroneous temperature detection
- Version flag

## [0.3.1] - 2023-02-20
### Changed
- Use formatted output for Pour JSON fields
- Truncate state file on save

### Fixed
- Pour event counting
- Panic during pour pruning
- Panic during state reload

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

