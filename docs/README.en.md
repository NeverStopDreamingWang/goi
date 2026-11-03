# Goi Developer Documentation

[简体中文](README.md) | **English** · [Project Home](../README.en.md)

These guides describe configuration, usage, extension APIs, and current implementation boundaries by feature. Examples import `github.com/NeverStopDreamingWang/goi/v3`; the checked-out source is authoritative for exact APIs.

## Where to Start

1. Begin with [Quick Start](../README.en.md#quick-start) or the [CLI Guide](../cmd/goi/README.en.md).
2. Configure [logging](logging.en.md) and [server settings](configuration.en.md), then register [routes](routing.en.md) and [middleware](middleware.en.md).
3. Build handlers with [HTTP Requests and Responses](http.en.md) and [Parameter Validation](validation.en.md), passing the request context into database operations.
4. Register background work and cleanup through the [Lifecycle Guide](lifecycle.en.md), then call `RunServer`.

## Feature Guides

| Guide | Contents |
| --- | --- |
| [Routing, Converters, and Static Resources](routing.en.md) | Groups, method mapping, 404/405 handling, converters, local files/directories, and embedded resources |
| [Middleware](middleware.en.md) | Three hooks, ordering, short circuits, exception boundaries, and built-in CORS/security-header implementations |
| [HTTP Requests and Responses](http.en.md) | Path/query/body parameters, MIME and Parser selection, uploads, context, Response, Data, Result, and RawHandler |
| [Parameter Validation and Type Conversion](validation.en.md) | Field tags, built-in validators, custom Parse methods, error responses, and reflection assignment |
| [Startup, Background Tasks, and Shutdown](lifecycle.en.md) | Startup, ShutdownCallback, signals/manual stopping, errors, resource order, and waiting boundaries |
| [In-memory Cache](cache.en.md) | Reads/writes, TTL, cleanup modes, capacity/eviction settings, and current limits |
| [Logging](logging.en.md) | Default logger setup, levels, custom output, access logs, rotation, and caller-managed closing |
| [Configuration, HTTPS, Language, and Time](configuration.en.md) | Shared settings, certificates, self-signed helpers, global language, and time representations |
| [Password Hashing and Cryptographic Utilities](security.en.md) | PBKDF2-SHA256, random data, AES/RSA formats, and usage boundaries |
| [Databases and Custom ORM (CN)](../db/README.md) | Connection registration, configuration precedence, models, identifier/placeholder/column strategies, and result scanning |
| [CLI Guide](../cmd/goi/README.en.md) | Installation, optional builds, project/application path rules, registration, migration, and running |

## Examples and Verification Boundaries

For a complete application, see the separate [goi_example](https://github.com/NeverStopDreamingWang/goi_example) project. Its configuration, database, and initialization behavior differs from standalone framework examples.

Complete Go examples can be compiled as standalone files. Programs containing `RunServer` block and listen on a port; logging and certificate examples write files. Do not bulk-run every example simply to validate documentation. Embedded-resource examples also need their resource files; fragments belong in the documented surrounding context.

These guides describe existing behavior rather than treating field names or comments as additional guarantees. Read the limitations for scheduled cleanup, eviction, proxy headers, logging concurrency, XML, and cryptographic inputs in the relevant guides. This documentation work does not change production implementations.

## Related Links

[Project Home](../README.en.md) · [Releases](https://github.com/NeverStopDreamingWang/goi/releases) · [Source Module](../go.mod)
