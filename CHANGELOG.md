# Changelog

## [v0.4.2] - 2025-11-19

Changed:
- throwing hands with staticcheck

## [v0.4.1] - 2025-11-19

Changed:
- Improved comment clarity and removed false positive lint warning.

## [v0.4.0] - 2025-08-27

Adds a new package, `xnet`, that provides a function to wait until the network is likely usable.

The `Wait` function blocks until a non-loopback, UP interface has a global IP address and at least one probe succeeds.
It includes default probes (Cloudflare DNS and example.com) and allows users to specify custom probes such as TCP addresses or domain names.
It uses exponential backoff with jitter for retries.

## [v0.3.0] - 2025-08-05

- Added Print / Printf to xlog for further compatibility.

## [v0.2.0] - 2025-07-26

Added:

- Shutdown func to server in httpx.

## [v0.1.0] - 2025-06-30

Project is now at version 0.1.0, marking a stable initial release. API still subject to change, but the core functionality is solid and ready for production use.

Added:

- Contributing guidelines.

Changed:

- Workflow name.
- Improved readme.

## [v0.0.9] - 2025-06-28

Added:

- Workflow to test and automatically tag releases based on `CHANGELOG.md` entries.
