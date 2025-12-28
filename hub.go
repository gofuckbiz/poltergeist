package poltergeist

import (
	"context"
	"sync"
	"time"
)

// =============================================================================
// BASE HUB - Common functionality for WebSocket and SSE hubs (DRY)
// =============================================================================

// BaseHub provides common hub functionality for managing connections and rooms
// This implements the DRY principle by extracting shared code
type BaseHub struct {
	mu       sync.RWMutex
	rooms    map[string]map[string]bool // room -> set of client IDs
	running  bool
	shutdown chan struct{} // Graceful shutdown signal
	done     chan struct{} // Shutdown complete signal
}

// newBaseHub creates a new BaseHub
func newBaseHub() *BaseHub {
	return &BaseHub{
		rooms:    make(map[string]map[string]bool),
		shutdown: make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Shutdown gracefully shuts down the hub
func (h *BaseHub) Shutdown(ctx context.Context) error {
	h.setRunning(false)
	close(h.shutdown)

	// Wait for done or context timeout
	select {
	case <-h.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ShutdownWithTimeout gracefully shuts down the hub with timeout
func (h *BaseHub) ShutdownWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return h.Shutdown(ctx)
}

// shutdownChan returns the shutdown channel for select statements
func (h *BaseHub) shutdownChan() <-chan struct{} {
	return h.shutdown
}

// markDone signals that shutdown is complete
func (h *BaseHub) markDone() {
	close(h.done)
}

// addToRoom adds a client to a room
func (h *BaseHub) addToRoom(clientID, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.rooms[room] == nil {
		h.rooms[room] = make(map[string]bool)
	}
	h.rooms[room][clientID] = true
}

// removeFromRoom removes a client from a room
func (h *BaseHub) removeFromRoom(clientID, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.rooms[room]; ok {
		delete(clients, clientID)
		if len(clients) == 0 {
			delete(h.rooms, room)
		}
	}
}

// removeFromAllRooms removes a client from all rooms
func (h *BaseHub) removeFromAllRooms(clientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for room, clients := range h.rooms {
		delete(clients, clientID)
		if len(clients) == 0 {
			delete(h.rooms, room)
		}
	}
}

// getRoomClientIDs returns all client IDs in a room
func (h *BaseHub) getRoomClientIDs(room string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.rooms[room]
	if !ok {
		return nil
	}

	ids := make([]string, 0, len(clients))
	for id := range clients {
		ids = append(ids, id)
	}
	return ids
}

// roomCount returns the number of clients in a room
func (h *BaseHub) roomCount(room string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.rooms[room]; ok {
		return len(clients)
	}
	return 0
}

// setRunning sets the running state
func (h *BaseHub) setRunning(running bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.running = running
}
