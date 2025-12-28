package poltergeist

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// =============================================================================
// SSE CONFIGURATION
// =============================================================================

// SSEConfig holds SSE configuration options
type SSEConfig struct {
	RetryInterval     int           // Retry interval for client reconnection (ms)
	KeepAliveInterval time.Duration // Keep-alive interval
	BufferSize        int           // Buffer size for events
	WriteTimeout      time.Duration // Write timeout (default: 10s)
}

// DefaultSSEConfig returns default SSE configuration
func DefaultSSEConfig() *SSEConfig {
	return &SSEConfig{
		RetryInterval:     DefaultSSERetryInterval,
		KeepAliveInterval: DefaultSSEKeepAliveInterval,
		BufferSize:        DefaultBufferSize,
		WriteTimeout:      DefaultSSEWriteTimeout,
	}
}

// =============================================================================
// SSE EVENT
// =============================================================================

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	Event string // Event type
	Data  any    // Event data
	ID    string // Event ID
	Retry int    // Retry interval (ms)
}

// =============================================================================
// SSE WRITER
// =============================================================================

// SSEWriter handles Server-Sent Events streaming
type SSEWriter struct {
	w           http.ResponseWriter
	flusher     http.Flusher
	config      *SSEConfig
	closed      bool
	closeMu     sync.Mutex
	pipeline    *EventPipeline
	ctx         *Context
	id          string // Unique writer ID for room management
	lastEventID string // Last event ID for reconnection support
}

// newSSEWriter creates a new SSE writer
func newSSEWriter(w http.ResponseWriter, config *SSEConfig, pipeline *EventPipeline, ctx *Context) (*SSEWriter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming unsupported")
	}

	// Set SSE headers
	w.Header().Set(HeaderContentType, ContentTypeSSE)
	w.Header().Set(HeaderCacheControl, "no-cache")
	w.Header().Set(HeaderConnection, "keep-alive")
	w.Header().Set(HeaderAccessControlAllow, "*")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Send retry interval
	if config.RetryInterval > 0 {
		fmt.Fprintf(w, "retry: %d\n\n", config.RetryInterval)
		flusher.Flush()
	}

	// Get Last-Event-ID for reconnection support
	lastEventID := ""
	if ctx != nil && ctx.Request != nil {
		lastEventID = ctx.Request.Header.Get("Last-Event-ID")
	}

	return &SSEWriter{
		w:           w,
		flusher:     flusher,
		config:      config,
		pipeline:    pipeline,
		ctx:         ctx,
		id:          generateConnID(),
		lastEventID: lastEventID,
	}, nil
}

// LastEventID returns the Last-Event-ID sent by client on reconnection
// This allows resuming from where the client left off
func (s *SSEWriter) LastEventID() string {
	return s.lastEventID
}

// IsReconnect returns true if this is a reconnection (has Last-Event-ID)
func (s *SSEWriter) IsReconnect() bool {
	return s.lastEventID != ""
}

// --- Send Methods ---

// Send sends an SSE event
func (s *SSEWriter) Send(event *SSEEvent) error {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()

	if s.closed {
		return fmt.Errorf("SSE writer closed")
	}

	// Write event fields
	if event.Event != "" {
		if _, err := fmt.Fprintf(s.w, "event: %s\n", event.Event); err != nil {
			return err
		}
	}
	if event.ID != "" {
		if _, err := fmt.Fprintf(s.w, "id: %s\n", event.ID); err != nil {
			return err
		}
	}
	if event.Retry > 0 {
		if _, err := fmt.Fprintf(s.w, "retry: %d\n", event.Retry); err != nil {
			return err
		}
	}

	// Write data (serialize if needed)
	dataStr := s.serializeData(event.Data)
	if _, err := fmt.Fprintf(s.w, "data: %s\n\n", dataStr); err != nil {
		return err
	}

	s.flusher.Flush()
	return nil
}

// serializeData converts data to string (DRY helper)
func (s *SSEWriter) serializeData(data any) string {
	switch v := data.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		if jsonData, err := json.Marshal(v); err == nil {
			return string(jsonData)
		}
		return fmt.Sprintf("%v", v)
	}
}

// SendData sends data without event type
func (s *SSEWriter) SendData(data any) error {
	return s.Send(&SSEEvent{Data: data})
}

// SendEvent sends an event with type and data
func (s *SSEWriter) SendEvent(eventType string, data any) error {
	return s.Send(&SSEEvent{Event: eventType, Data: data})
}

// SendJSON sends JSON data
func (s *SSEWriter) SendJSON(data any) error {
	return s.Send(&SSEEvent{Data: data})
}

// SendComment sends a comment (for keep-alive)
func (s *SSEWriter) SendComment(comment string) error {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()

	if s.closed {
		return fmt.Errorf("SSE writer closed")
	}

	if _, err := fmt.Fprintf(s.w, ": %s\n\n", comment); err != nil {
		return err
	}
	s.flusher.Flush()
	return nil
}

// --- Lifecycle ---

// Close closes the SSE writer
func (s *SSEWriter) Close() {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()

	if s.closed {
		return
	}

	s.closed = true
	if s.pipeline != nil && s.ctx != nil {
		s.pipeline.Emit(EventSSEDisconnect, s.ctx)
	}
}

// IsClosed returns whether the writer is closed
func (s *SSEWriter) IsClosed() bool {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()
	return s.closed
}

// =============================================================================
// SSE HUB - Manages multiple SSE connections
// =============================================================================

// SSEHub manages multiple SSE connections
type SSEHub struct {
	*BaseHub                          // Embed common hub functionality (DRY)
	clients     map[*SSEWriter]bool   // Active clients
	clientMu    sync.RWMutex          // Client mutex
	register    chan *SSEWriter       // Register channel
	unregister  chan *SSEWriter       // Unregister channel
	broadcast   chan *SSEEvent        // Broadcast channel
	clientIndex map[string]*SSEWriter // ID -> client mapping for rooms
}

// NewSSEHub creates a new SSE hub
func NewSSEHub() *SSEHub {
	return &SSEHub{
		BaseHub:     newBaseHub(),
		clients:     make(map[*SSEWriter]bool),
		register:    make(chan *SSEWriter),
		unregister:  make(chan *SSEWriter),
		broadcast:   make(chan *SSEEvent, DefaultBufferSize),
		clientIndex: make(map[string]*SSEWriter),
	}
}

// Run starts the hub's main event loop
func (h *SSEHub) Run() {
	h.setRunning(true)
	defer h.markDone()

	for {
		select {
		case <-h.shutdownChan():
			h.closeAllClients()
			return
		case client := <-h.register:
			h.registerClient(client)
		case client := <-h.unregister:
			h.unregisterClient(client)
		case event := <-h.broadcast:
			h.broadcastToAll(event)
		}
	}
}

// Stop stops the hub (deprecated, use Shutdown for graceful shutdown)
func (h *SSEHub) Stop() {
	h.setRunning(false)
}

// closeAllClients closes all SSE clients gracefully
func (h *SSEHub) closeAllClients() {
	h.clientMu.Lock()
	defer h.clientMu.Unlock()

	for client := range h.clients {
		// Send goodbye event before closing
		client.Send(&SSEEvent{
			Event: "shutdown",
			Data:  "server shutting down",
		})
		client.Close()
		delete(h.clients, client)
		delete(h.clientIndex, client.id)
	}
}

// --- Internal helpers (KISS) ---

func (h *SSEHub) registerClient(client *SSEWriter) {
	h.clientMu.Lock()
	defer h.clientMu.Unlock()
	h.clients[client] = true
	h.clientIndex[client.id] = client
}

func (h *SSEHub) unregisterClient(client *SSEWriter) {
	h.clientMu.Lock()
	defer h.clientMu.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		delete(h.clientIndex, client.id)
		h.removeFromAllRooms(client.id)
		client.Close()
	}
}

func (h *SSEHub) broadcastToAll(event *SSEEvent) {
	h.clientMu.RLock()
	defer h.clientMu.RUnlock()

	for client := range h.clients {
		if err := client.Send(event); err != nil {
			go func(c *SSEWriter) { h.unregister <- c }(client)
		}
	}
}

// --- Public API ---

// Broadcast sends an event to all clients
func (h *SSEHub) Broadcast(event *SSEEvent) {
	h.broadcast <- event
}

// BroadcastData sends data to all clients
func (h *SSEHub) BroadcastData(data any) {
	h.Broadcast(&SSEEvent{Data: data})
}

// BroadcastEvent sends an event with type and data to all clients
func (h *SSEHub) BroadcastEvent(eventType string, data any) {
	h.Broadcast(&SSEEvent{Event: eventType, Data: data})
}

// BroadcastToRoom sends an event to all clients in a room
func (h *SSEHub) BroadcastToRoom(room string, event *SSEEvent) {
	h.clientMu.RLock()
	defer h.clientMu.RUnlock()

	for _, clientID := range h.getRoomClientIDs(room) {
		if client, ok := h.clientIndex[clientID]; ok {
			if err := client.Send(event); err != nil {
				go func(c *SSEWriter) { h.unregister <- c }(client)
			}
		}
	}
}

// JoinRoom adds a client to a room
func (h *SSEHub) JoinRoom(client *SSEWriter, room string) {
	h.addToRoom(client.id, room)
}

// LeaveRoom removes a client from a room
func (h *SSEHub) LeaveRoom(client *SSEWriter, room string) {
	h.removeFromRoom(client.id, room)
}

// ClientCount returns the number of connected clients
func (h *SSEHub) ClientCount() int {
	h.clientMu.RLock()
	defer h.clientMu.RUnlock()
	return len(h.clients)
}

// RoomCount returns the number of clients in a room
func (h *SSEHub) RoomCount(room string) int {
	return h.roomCount(room)
}

// =============================================================================
// SSE HANDLERS - Server integration
// =============================================================================

// SSEHandler is the function type for handling SSE connections
type SSEHandler func(ctx *Context, sse *SSEWriter)

// SSE creates an SSE handler
func (s *Server) SSE(path string, handler SSEHandler, config ...*SSEConfig) *Route {
	cfg := getSSEConfig(config)

	return s.GET(path, func(c *Context) error {
		sse, err := newSSEWriter(c.Writer, cfg, s.Pipeline(), c)
		if err != nil {
			return c.Error(http.StatusInternalServerError, err.Error())
		}
		c.SSE = sse

		s.Pipeline().Emit(EventSSEConnect, c)

		// Wait for disconnect
		done := make(chan struct{})
		go func() {
			<-c.Request.Context().Done()
			sse.Close()
			close(done)
		}()

		handler(c, sse)
		<-done
		return nil
	})
}

// SSEWithHub creates an SSE handler with hub support
func (s *Server) SSEWithHub(path string, hub *SSEHub, handler SSEHandler, config ...*SSEConfig) *Route {
	cfg := getSSEConfig(config)

	return s.GET(path, func(c *Context) error {
		sse, err := newSSEWriter(c.Writer, cfg, s.Pipeline(), c)
		if err != nil {
			return c.Error(http.StatusInternalServerError, err.Error())
		}
		c.SSE = sse

		hub.register <- sse

		s.Pipeline().Emit(EventSSEConnect, c)

		// Wait for disconnect
		done := make(chan struct{})
		go func() {
			<-c.Request.Context().Done()
			hub.unregister <- sse
			close(done)
		}()

		handler(c, sse)
		<-done
		return nil
	})
}

// --- Helpers (DRY) ---

func getSSEConfig(config []*SSEConfig) *SSEConfig {
	if len(config) > 0 && config[0] != nil {
		return config[0]
	}
	return DefaultSSEConfig()
}
