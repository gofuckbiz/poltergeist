package poltergeist

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// =============================================================================
// WEBSOCKET CONFIGURATION
// =============================================================================

// WSConfig holds WebSocket configuration options
type WSConfig struct {
	ReadBufferSize    int                        // Read buffer size (default: 1024)
	WriteBufferSize   int                        // Write buffer size (default: 1024)
	EnableCompression bool                       // Enable compression
	CheckOrigin       func(r *http.Request) bool // Origin check function
	PingInterval      time.Duration              // Ping interval (default: 30s)
	PongTimeout       time.Duration              // Pong timeout (default: 60s)
	WriteTimeout      time.Duration              // Write timeout (default: 10s)
	ReadTimeout       time.Duration              // Read timeout (default: 60s)
	MaxMessageSize    int64                      // Max message size (default: 512KB)
	HandshakeTimeout  time.Duration              // Handshake timeout (default: 10s)
}

// DefaultWSConfig returns default WebSocket configuration
func DefaultWSConfig() *WSConfig {
	return &WSConfig{
		ReadBufferSize:    DefaultWSReadBufferSize,
		WriteBufferSize:   DefaultWSWriteBufferSize,
		EnableCompression: true,
		CheckOrigin:       func(r *http.Request) bool { return true },
		PingInterval:      DefaultWSPingInterval,
		PongTimeout:       DefaultWSPongTimeout,
		WriteTimeout:      DefaultWSWriteTimeout,
		ReadTimeout:       DefaultWSReadTimeout,
		MaxMessageSize:    DefaultMaxMessageSize,
		HandshakeTimeout:  DefaultWSHandshakeTimeout,
	}
}

// =============================================================================
// WEBSOCKET CONNECTION
// =============================================================================

// WSConn represents a WebSocket connection
type WSConn struct {
	conn     *websocket.Conn
	config   *WSConfig
	send     chan []byte
	closed   bool
	closeMu  sync.Mutex
	pipeline *EventPipeline
	ctx      *Context
	id       string // Unique connection ID for room management
}

// newWSConn creates a new WebSocket connection wrapper
func newWSConn(conn *websocket.Conn, config *WSConfig, pipeline *EventPipeline, ctx *Context) *WSConn {
	return &WSConn{
		conn:     conn,
		config:   config,
		send:     make(chan []byte, DefaultBufferSize),
		pipeline: pipeline,
		ctx:      ctx,
		id:       generateConnID(),
	}
}

// generateConnID generates a unique connection ID
func generateConnID() string {
	return time.Now().Format("20060102150405.000000000")
}

// --- Send Methods ---

// Send sends a raw message to the connection
func (c *WSConn) Send(message []byte) error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	if c.closed {
		return websocket.ErrCloseSent
	}

	select {
	case c.send <- message:
		return nil
	default:
		return websocket.ErrCloseSent
	}
}

// SendJSON sends a JSON message
func (c *WSConn) SendJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.Send(data)
}

// SendText sends a text message
func (c *WSConn) SendText(text string) error {
	return c.Send([]byte(text))
}

// --- Lifecycle ---

// Close closes the connection
func (c *WSConn) Close() error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	close(c.send)
	return c.conn.Close()
}

// readPump reads messages from the connection
func (c *WSConn) readPump(handler WSMessageHandler) {
	defer func() {
		if c.pipeline != nil && c.ctx != nil {
			c.pipeline.Emit(EventWSDisconnect, c.ctx)
		}
		c.Close()
	}()

	c.conn.SetReadLimit(c.config.MaxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(c.config.ReadTimeout))
	c.conn.SetPongHandler(func(string) error {
		// Reset read deadline on pong received
		c.conn.SetReadDeadline(time.Now().Add(c.config.ReadTimeout))
		return nil
	})

	for {
		messageType, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Reset read deadline after each message
		c.conn.SetReadDeadline(time.Now().Add(c.config.ReadTimeout))

		if handler != nil {
			handler(c, messageType, message)
		}
	}
}

// writePump writes messages to the connection
func (c *WSConn) writePump() {
	ticker := time.NewTicker(c.config.PingInterval)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// =============================================================================
// WEBSOCKET HUB - Manages multiple connections
// =============================================================================

// WSHub manages multiple WebSocket connections
type WSHub struct {
	*BaseHub                       // Embed common hub functionality (DRY)
	connections map[*WSConn]bool   // Active connections
	connMu      sync.RWMutex       // Connection mutex
	broadcast   chan []byte        // Broadcast channel
	register    chan *WSConn       // Register channel
	unregister  chan *WSConn       // Unregister channel
	connIndex   map[string]*WSConn // ID -> connection mapping for rooms
}

// NewWSHub creates a new WebSocket hub
func NewWSHub() *WSHub {
	return &WSHub{
		BaseHub:     newBaseHub(),
		connections: make(map[*WSConn]bool),
		broadcast:   make(chan []byte, DefaultBufferSize),
		register:    make(chan *WSConn),
		unregister:  make(chan *WSConn),
		connIndex:   make(map[string]*WSConn),
	}
}

// Run starts the hub's main event loop
func (h *WSHub) Run() {
	h.setRunning(true)
	defer h.markDone()

	for {
		select {
		case <-h.shutdownChan():
			h.closeAllConnections()
			return
		case conn := <-h.register:
			h.registerConn(conn)
		case conn := <-h.unregister:
			h.unregisterConn(conn)
		case message := <-h.broadcast:
			h.broadcastToAll(message)
		}
	}
}

// Stop stops the hub (deprecated, use Shutdown for graceful shutdown)
func (h *WSHub) Stop() {
	h.setRunning(false)
}

// closeAllConnections closes all WebSocket connections gracefully
func (h *WSHub) closeAllConnections() {
	h.connMu.Lock()
	defer h.connMu.Unlock()

	for conn := range h.connections {
		// Send close message before closing
		conn.conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseGoingAway, "server shutdown"),
		)
		conn.Close()
		delete(h.connections, conn)
		delete(h.connIndex, conn.id)
	}
}

// --- Internal helpers (KISS: small focused functions) ---

func (h *WSHub) registerConn(conn *WSConn) {
	h.connMu.Lock()
	defer h.connMu.Unlock()
	h.connections[conn] = true
	h.connIndex[conn.id] = conn
}

func (h *WSHub) unregisterConn(conn *WSConn) {
	h.connMu.Lock()
	defer h.connMu.Unlock()

	if _, ok := h.connections[conn]; ok {
		delete(h.connections, conn)
		delete(h.connIndex, conn.id)
		h.removeFromAllRooms(conn.id)
	}
}

func (h *WSHub) broadcastToAll(message []byte) {
	h.connMu.RLock()
	defer h.connMu.RUnlock()

	for conn := range h.connections {
		select {
		case conn.send <- message:
		default:
			go conn.Close()
		}
	}
}

// --- Public API ---

// Broadcast sends a message to all connections
func (h *WSHub) Broadcast(message []byte) {
	h.broadcast <- message
}

// BroadcastJSON sends a JSON message to all connections
func (h *WSHub) BroadcastJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	h.Broadcast(data)
	return nil
}

// BroadcastToRoom sends a message to all connections in a room
func (h *WSHub) BroadcastToRoom(room string, message []byte) {
	h.connMu.RLock()
	defer h.connMu.RUnlock()

	for _, clientID := range h.getRoomClientIDs(room) {
		if conn, ok := h.connIndex[clientID]; ok {
			select {
			case conn.send <- message:
			default:
				go conn.Close()
			}
		}
	}
}

// BroadcastJSONToRoom sends a JSON message to all connections in a room
func (h *WSHub) BroadcastJSONToRoom(room string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	h.BroadcastToRoom(room, data)
	return nil
}

// JoinRoom adds a connection to a room
func (h *WSHub) JoinRoom(conn *WSConn, room string) {
	h.addToRoom(conn.id, room)
}

// LeaveRoom removes a connection from a room
func (h *WSHub) LeaveRoom(conn *WSConn, room string) {
	h.removeFromRoom(conn.id, room)
}

// ConnectionCount returns the number of active connections
func (h *WSHub) ConnectionCount() int {
	h.connMu.RLock()
	defer h.connMu.RUnlock()
	return len(h.connections)
}

// RoomCount returns the number of connections in a room
func (h *WSHub) RoomCount(room string) int {
	return h.roomCount(room)
}

// =============================================================================
// WEBSOCKET HANDLERS - Server integration
// =============================================================================

// WSMessageHandler is the function type for handling WebSocket messages
type WSMessageHandler func(conn *WSConn, messageType int, message []byte)

// WebSocket creates a WebSocket handler
func (s *Server) WebSocket(path string, handler WSMessageHandler, config ...*WSConfig) *Route {
	cfg := getWSConfig(config)
	upgrader := createUpgrader(cfg)

	return s.GET(path, func(c *Context) error {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return err
		}

		wsConn := newWSConn(conn, cfg, s.Pipeline(), c)
		c.WS = wsConn

		s.Pipeline().Emit(EventWSConnect, c)

		go wsConn.writePump()
		wsConn.readPump(handler)

		return nil
	})
}

// WebSocketWithHub creates a WebSocket handler with hub support
func (s *Server) WebSocketWithHub(path string, hub *WSHub, handler WSMessageHandler, config ...*WSConfig) *Route {
	cfg := getWSConfig(config)
	upgrader := createUpgrader(cfg)

	return s.GET(path, func(c *Context) error {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return err
		}

		wsConn := newWSConn(conn, cfg, s.Pipeline(), c)
		c.WS = wsConn

		hub.register <- wsConn
		defer func() { hub.unregister <- wsConn }()

		s.Pipeline().Emit(EventWSConnect, c)

		go wsConn.writePump()
		wsConn.readPump(handler)

		return nil
	})
}

// --- Helpers (DRY) ---

func getWSConfig(config []*WSConfig) *WSConfig {
	if len(config) > 0 && config[0] != nil {
		return config[0]
	}
	return DefaultWSConfig()
}

func createUpgrader(cfg *WSConfig) websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:    cfg.ReadBufferSize,
		WriteBufferSize:   cfg.WriteBufferSize,
		EnableCompression: cfg.EnableCompression,
		CheckOrigin:       cfg.CheckOrigin,
		HandshakeTimeout:  cfg.HandshakeTimeout,
	}
}
