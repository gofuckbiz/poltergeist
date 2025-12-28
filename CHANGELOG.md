# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.1] - 2025-12-28

### Fixed

- 🔧 Fixed module path in all imports to match `go.mod`
- 📦 Synced all internal imports with `github.com/gofuckbiz/poltergeist`
- 📝 Updated README badges and documentation links

---

## [1.0.0] - 2025-12-28

### Added

- 🚀 **Core Framework**
  - HTTP routing with path parameters and wildcards
  - Route groups with shared prefix and middleware
  - Context helpers for request/response handling
  - JSON, String, HTML, and Bytes response methods
  - Query and path parameter parsing

- 🔌 **Realtime Support**
  - WebSocket connections with hub management
  - Server-Sent Events (SSE) streaming
  - Room-based broadcasting for both WS and SSE
  - Automatic connection lifecycle management

- 📡 **Event Pipeline**
  - BeforeRequest / AfterRequest hooks
  - Error handling hooks
  - Server lifecycle events (start/stop)
  - WebSocket and SSE connection events

- 🛡️ **Built-in Middleware**
  - Logger: Request logging with colors
  - Recovery: Panic recovery with stack traces
  - CORS: Cross-Origin Resource Sharing
  - RateLimit: Token bucket rate limiting
  - Auth: Basic, Bearer, and API Key authentication
  - Secure: Security headers
  - Gzip: Response compression
  - Timeout: Request timeout
  - RequestID: Unique request identification

- 📚 **Documentation**
  - Automatic OpenAPI/Swagger generation
  - Swagger UI integration
  - Route metadata (name, description, tags)
  - Request/Response body documentation

- ⚙️ **Configuration**
  - Zero-config with sensible defaults
  - Custom configuration support
  - TLS/HTTPS support
  - Graceful shutdown
  - Development mode

### Performance

- Static route: ~1.3M requests/second
- Param route: ~850K requests/second
- Event pipeline emit: ~100M operations/second
- Context get/set: ~34M operations/second

---

## [Unreleased]

### Planned

- Template rendering support
- Session management
- Database helpers
- GraphQL support
- gRPC integration

