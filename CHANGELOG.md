# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0]  - 2024-08-09
### Added
- unit tests for private func `middleware.normalize`
- unit tests for private func `middleware.containsPath`
- unit tests for private func `middleware.fromMap`

### Updated
- package dependencies

### Removed
- deprecated func `middleware.Initialize`
- deprecated func `middleware.Tracing`
- type `middleware.TracingOptions`
- type `middleware.TracingOption`
- func `middleware.NewTracingOptions`
- type `middleware.MetricsOptions`
- type `middleware.MetricsOption`
- type `middleware.LoggingOptions`
- type `middleware.LoggingOption`

## [1.2.0]  - 2024-07-10
### Added
- Added `OtelTracing` func that returns the Otel tracing middleware
- Added `PrometheusMetrics` func that returns the Prometheus metrics middleware
- Added `Logging` func that returns the logging middleware
- updated README.md to correct `go get` cmd
- updated packages to latest version

### Updated
- Marked `Telemetry` and `Initialize` functions as deprecated

## [1.1.0]  - 2024-07-08
### Updated
- Retracted all previous versions.
- Updated README.md to correct `go get` cmd.
- Updated packages to latest version.

## [1.0.2]  - 2024-06-25
### Fixed
- Corrected issue where the guage tracking concurrent requests being served was not being incremented, only decremented.

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

