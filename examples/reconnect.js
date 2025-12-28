/**
 * Poltergeist Reconnection Helpers
 * Client-side JavaScript for automatic WebSocket and SSE reconnection
 */

// =============================================================================
// WEBSOCKET WITH AUTO-RECONNECT
// =============================================================================

/**
 * Creates a WebSocket connection with automatic reconnection
 * @param {string} url - WebSocket URL
 * @param {Object} options - Configuration options
 * @returns {Object} - Connection wrapper with send/close methods
 */
function createReconnectingWebSocket(url, options = {}) {
    const config = {
        reconnectInterval: options.reconnectInterval || 1000,
        maxReconnectInterval: options.maxReconnectInterval || 30000,
        reconnectDecay: options.reconnectDecay || 1.5,
        maxRetries: options.maxRetries || Infinity,
        onOpen: options.onOpen || (() => {}),
        onMessage: options.onMessage || (() => {}),
        onClose: options.onClose || (() => {}),
        onError: options.onError || (() => {}),
        onReconnect: options.onReconnect || (() => {}),
        protocols: options.protocols || [],
    };

    let ws = null;
    let retries = 0;
    let currentInterval = config.reconnectInterval;
    let forceClosed = false;
    let messageQueue = [];

    function connect() {
        ws = new WebSocket(url, config.protocols);

        ws.onopen = (event) => {
            console.log('[WS] Connected to', url);
            retries = 0;
            currentInterval = config.reconnectInterval;
            
            // Send queued messages
            while (messageQueue.length > 0) {
                ws.send(messageQueue.shift());
            }
            
            config.onOpen(event);
        };

        ws.onmessage = (event) => {
            config.onMessage(event);
        };

        ws.onclose = (event) => {
            console.log('[WS] Connection closed', event.code, event.reason);
            config.onClose(event);
            
            if (!forceClosed && retries < config.maxRetries) {
                scheduleReconnect();
            }
        };

        ws.onerror = (error) => {
            console.error('[WS] Error:', error);
            config.onError(error);
        };
    }

    function scheduleReconnect() {
        retries++;
        console.log(`[WS] Reconnecting in ${currentInterval}ms (attempt ${retries})`);
        
        setTimeout(() => {
            config.onReconnect(retries);
            connect();
        }, currentInterval);

        // Exponential backoff
        currentInterval = Math.min(
            currentInterval * config.reconnectDecay,
            config.maxReconnectInterval
        );
    }

    // Initial connection
    connect();

    // Return wrapper object
    return {
        send(data) {
            if (ws && ws.readyState === WebSocket.OPEN) {
                ws.send(typeof data === 'string' ? data : JSON.stringify(data));
            } else {
                // Queue message for when connection is restored
                messageQueue.push(typeof data === 'string' ? data : JSON.stringify(data));
            }
        },

        sendJSON(data) {
            this.send(JSON.stringify(data));
        },

        close() {
            forceClosed = true;
            if (ws) {
                ws.close();
            }
        },

        get readyState() {
            return ws ? ws.readyState : WebSocket.CLOSED;
        },

        get isConnected() {
            return ws && ws.readyState === WebSocket.OPEN;
        },

        reconnect() {
            if (ws) {
                ws.close();
            }
            forceClosed = false;
            connect();
        }
    };
}

// =============================================================================
// SSE WITH AUTO-RECONNECT
// =============================================================================

/**
 * Creates an SSE connection with automatic reconnection
 * Note: SSE has built-in reconnection, but this provides more control
 * @param {string} url - SSE endpoint URL
 * @param {Object} options - Configuration options
 * @returns {Object} - Connection wrapper with close method
 */
function createReconnectingEventSource(url, options = {}) {
    const config = {
        withCredentials: options.withCredentials || false,
        reconnectInterval: options.reconnectInterval || 3000,
        maxRetries: options.maxRetries || Infinity,
        onOpen: options.onOpen || (() => {}),
        onMessage: options.onMessage || (() => {}),
        onError: options.onError || (() => {}),
        onReconnect: options.onReconnect || (() => {}),
        eventHandlers: options.eventHandlers || {},
    };

    let eventSource = null;
    let retries = 0;
    let forceClosed = false;
    let lastEventId = null;

    function connect() {
        // Add Last-Event-ID to URL for resumption
        let connectUrl = url;
        if (lastEventId) {
            const separator = url.includes('?') ? '&' : '?';
            connectUrl = `${url}${separator}lastEventId=${encodeURIComponent(lastEventId)}`;
        }

        eventSource = new EventSource(connectUrl, {
            withCredentials: config.withCredentials
        });

        eventSource.onopen = (event) => {
            console.log('[SSE] Connected to', url);
            retries = 0;
            config.onOpen(event);
        };

        eventSource.onmessage = (event) => {
            lastEventId = event.lastEventId || lastEventId;
            config.onMessage(event);
        };

        eventSource.onerror = (error) => {
            console.error('[SSE] Error:', error);
            config.onError(error);
            
            if (eventSource.readyState === EventSource.CLOSED && !forceClosed) {
                if (retries < config.maxRetries) {
                    scheduleReconnect();
                }
            }
        };

        // Register custom event handlers
        for (const [eventType, handler] of Object.entries(config.eventHandlers)) {
            eventSource.addEventListener(eventType, (event) => {
                lastEventId = event.lastEventId || lastEventId;
                handler(event);
            });
        }

        // Handle shutdown event from server
        eventSource.addEventListener('shutdown', (event) => {
            console.log('[SSE] Server shutting down:', event.data);
            // Don't reconnect on graceful shutdown
            forceClosed = true;
            eventSource.close();
        });
    }

    function scheduleReconnect() {
        retries++;
        console.log(`[SSE] Reconnecting in ${config.reconnectInterval}ms (attempt ${retries})`);
        
        setTimeout(() => {
            config.onReconnect(retries);
            connect();
        }, config.reconnectInterval);
    }

    // Initial connection
    connect();

    // Return wrapper object
    return {
        close() {
            forceClosed = true;
            if (eventSource) {
                eventSource.close();
            }
        },

        get readyState() {
            return eventSource ? eventSource.readyState : EventSource.CLOSED;
        },

        get isConnected() {
            return eventSource && eventSource.readyState === EventSource.OPEN;
        },

        reconnect() {
            if (eventSource) {
                eventSource.close();
            }
            forceClosed = false;
            connect();
        },

        addEventListener(type, handler) {
            if (eventSource) {
                eventSource.addEventListener(type, handler);
            }
        }
    };
}

// =============================================================================
// USAGE EXAMPLES
// =============================================================================

/*
// WebSocket Example:
const ws = createReconnectingWebSocket('ws://localhost:8080/ws', {
    reconnectInterval: 1000,
    maxReconnectInterval: 30000,
    maxRetries: 10,
    onOpen: () => console.log('Connected!'),
    onMessage: (event) => console.log('Message:', event.data),
    onClose: () => console.log('Disconnected'),
    onReconnect: (attempt) => console.log('Reconnecting, attempt:', attempt),
});

ws.send('Hello, server!');
ws.sendJSON({ type: 'message', content: 'Hello!' });

// SSE Example:
const sse = createReconnectingEventSource('http://localhost:8080/events', {
    reconnectInterval: 3000,
    maxRetries: 10,
    onOpen: () => console.log('SSE Connected!'),
    onMessage: (event) => console.log('SSE Message:', event.data),
    eventHandlers: {
        'notification': (event) => console.log('Notification:', event.data),
        'update': (event) => console.log('Update:', event.data),
    },
});
*/

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
        createReconnectingWebSocket,
        createReconnectingEventSource
    };
}

