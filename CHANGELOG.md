# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [1.3.1] - 2024-02-04
### Fixed
- `main.go` fix month number at report generation.

## [1.3.0] - 2024-01-17
### Added
- `calendar.go` that has an embedded working hours calendar.

### Changed
- `-report` returns not only spent time but also number of working hours for the current month.

## [1.2.0] - 2024-01-16
### Added
- `-report` flag to generate daily, weekly and monthly recorded hours.
- `-logReport` to show **tired** log even when doing a report (`-report`).

## [1.1.0] - 2024-01-14
### Changed
- Split code onto files.

### Added
- Records validation.
- Records number.

## [1.0.2] - 2023-08-03
### Fixed
- Skip unfinished lines.

## [1.0.1] - 2023-08-02
### Fixed
- Don't split comment with `,` sign.

## [1.0.0] - 2023-07-28
### Added
- First release.
