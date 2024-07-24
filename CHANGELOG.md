# Changelog

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
