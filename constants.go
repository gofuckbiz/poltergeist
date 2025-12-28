package poltergeist

import (
	"net/http"
	"time"
)

// =============================================================================
// CONSTANTS - DRY principle: Define once, use everywhere
// =============================================================================

// Content types
const (
	ContentTypeJSON      = "application/json; charset=utf-8"
	ContentTypeText      = "text/plain; charset=utf-8"
	ContentTypeHTML      = "text/html; charset=utf-8"
	ContentTypeSSE       = "text/event-stream"
	ContentTypeForm      = "application/x-www-form-urlencoded"
	ContentTypeMultipart = "multipart/form-data"
)

// Header names
const (
	HeaderContentType        = "Content-Type"
	HeaderAuthorization      = "Authorization"
	HeaderAccept             = "Accept"
	HeaderAcceptEncoding     = "Accept-Encoding"
	HeaderContentEncoding    = "Content-Encoding"
	HeaderCacheControl       = "Cache-Control"
	HeaderConnection         = "Connection"
	HeaderXForwardedFor      = "X-Forwarded-For"
	HeaderXRealIP            = "X-Real-IP"
	HeaderXRequestID         = "X-Request-ID"
	HeaderAccessControlAllow = "Access-Control-Allow-Origin"
)

// AllHTTPMethods contains all standard HTTP methods
// DRY: Single source of truth for HTTP methods
var AllHTTPMethods = []string{
	http.MethodGet,
	http.MethodPost,
	http.MethodPut,
	http.MethodDelete,
	http.MethodPatch,
	http.MethodOptions,
	http.MethodHead,
}

// Default timeouts
const (
	DefaultReadTimeout     = 30 * time.Second
	DefaultWriteTimeout    = 30 * time.Second
	DefaultIdleTimeout     = 120 * time.Second
	DefaultShutdownTimeout = 30 * time.Second
)

// Default sizes
const (
	DefaultMaxHeaderBytes = 1 << 20 // 1MB
	DefaultBufferSize     = 256
	DefaultMaxMessageSize = 512 * 1024 // 512KB
)

// WebSocket defaults
const (
	DefaultWSReadBufferSize   = 1024
	DefaultWSWriteBufferSize  = 1024
	DefaultWSPingInterval     = 30 * time.Second
	DefaultWSPongTimeout      = 60 * time.Second
	DefaultWSWriteTimeout     = 10 * time.Second
	DefaultWSReadTimeout      = 60 * time.Second
	DefaultWSHandshakeTimeout = 10 * time.Second
)

// SSE defaults
const (
	DefaultSSERetryInterval     = 3000 // milliseconds
	DefaultSSEKeepAliveInterval = 30 * time.Second
	DefaultSSEWriteTimeout      = 10 * time.Second
)

// Hub shutdown defaults
const (
	DefaultHubShutdownTimeout = 30 * time.Second
)
