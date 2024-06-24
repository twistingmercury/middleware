# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.1] - 2024-06-24

### Fixed
- Corrected issue where values for middleware metric labels were not properly applied.

### Removed
- `_example` dir

## [1.0.0] - 2024-06-21

### Fixed
- Corrected issue where the trace was not being propagated through the request context.

### Removed
- `_example` dir

## [0.9.0] - 2024-06-03

### Added
- Initial release of the project.
- Provides telemetry data for RESTful APIs build using [gin-gonic/gin](https://github.com/gin-gonic/gin).
- Telemetry data is generated using the packages in [twistingmercury/telemetry](https://github.com/twistingmercury/telemetry)

