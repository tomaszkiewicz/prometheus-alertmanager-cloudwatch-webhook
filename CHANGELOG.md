# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Prometheus metrics are exported on the `/metrics` endpoint.
- Alerts' labels can be exported as Cloudwatch metrics' dimensions.
- Configuration can be provided via a yml file.

### Changed

- Activated `CredentialsChainVerboseErrors` to get more meaningful messages when AWS session can't be created.
- Labels are parsed as a `map[string]string` as they are variable based on the alerts received from alert manager.
- Moved from viper to koanf for managing configuration. (No fix for this issue <https://github.com/spf13/viper/issues/373>)
- Upgraded go to 1.17
- Upgraded gin to v1.8.1
- Upgraded aws-sdk-go to v1.44.135

### Fixed

- Concurrent update of the global `webhookData` when multiple alerts triggerred are calling the webhook at the same time.
