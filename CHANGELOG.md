# Changelog

## [3.3.0] - 2026-01-16
### Added
- Galera Cluster topology support for high-availability multi-master replication
- Example configuration and documentation for Galera deployments

## [3.2.0] - 2025-10-23
### Added
- Multi-organization support.

## [3.1.5] - 2025-10-23
### Fixed
- prevent start/stop operations on serverless-standalone services.

## [3.1.4] - 2025-07-17
### Features
- service `tags` support added.

## [3.1.3] - 2024-07-25
### Package Updates
- upgraded go 1.19 => 1.21
- added toolchain go1.22.3
- upgraded github.com/golang/protobuf v1.5.2 => v1.5.4
- upgraded github.com/google/go-cmp v0.5.9 => v0.6.0
- upgraded github.com/google/uuid v1.3.0 => v1.6.0
- upgraded google.golang.org/grpc v1.54.0 => v1.65.0

## [3.1.2] - 2024-07-24
### Package Vulnerability Updates
- upgraded google.golang.org/protobuf v1.30.0 => v1.34.2
- upgraded golang.org/x/crypto v0.0.0-20220622213112-05595931fe9d => v0.25.0
- upgraded golang.org/x/net v0.8.0 => v0.27.0
- upgraded golang.org/x/sys v0.6.0 => v0.22.0
- upgraded golang.org/x/text v0.8.0 => v0.16.0

## [3.1.1] - 2024-07-16
### Fixed
- `gp3` volume type support for AWS.
- `volume_throughput` support for AWS GP3 storage volume type.

## [3.1.0] - 2024-07-15
### Features
- `azure` provider is now supported.

## [3.0.0] - 2024-06-05
### Breaking Change
- New API Key Access, `TF_SKYSQL_API_KEY` [API Access](https://app.skysql.com/user-profile/api-keys)

## [2.0.1] - 2024-02-24
**OBSOLETE - DO NOT USE**
### Fixed
- `volume_type` imports work as expected.
