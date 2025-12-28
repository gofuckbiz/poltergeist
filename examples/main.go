package main

import (
	"log"
	"time"

	"github.com/gofuckbiz/poltergeist"
	"github.com/gofuckbiz/poltergeist/docs"
	"github.com/gofuckbiz/poltergeist/middleware"
)

// User represents a user model
type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateUserRequest represents create user request
type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Message represents a chat message
type Message struct {
	User    string `json:"user"`
	Content string `json:"content"`
	Time    string `json:"time"`
}

// In-memory storage
var users = []User{
	{ID: 1, Name: "John Doe", Email: "john@example.com", CreatedAt: time.Now()},
	{ID: 2, Name: "Jane Smith", Email: "jane@example.com", CreatedAt: time.Now()},
}

var nextUserID = 3

func main() {
	// Create new Poltergeist server
	app := poltergeist.New()

	// Add global middleware
	app.Use(middleware.Logger())
	app.Use(middleware.Recovery())
	app.Use(middleware.CORS())

	// Setup event pipeline
	setupEventPipeline(app)

	// Setup routes
	setupRoutes(app)

	// Setup WebSocket
	setupWebSocket(app)

	// Setup SSE
	setupSSE(app)

	// Setup Swagger documentation
	docs.Swagger(app, &docs.SwaggerConfig{
		Title:       "Poltergeist Demo API",
		Description: "A demo API showcasing Poltergeist features",
		Version:     "1.0.0",
	})

	// Start server
	log.Println("Starting Poltergeist Demo Server...")
	if err := app.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

func setupEventPipeline(app *poltergeist.Server) {
	pipeline := app.Pipeline()

	// Before request hook
	pipeline.BeforeRequest(func(c *poltergeist.Context) {
		c.Set("request_start", time.Now())
	})

	// After request hook
	pipeline.AfterRequest(func(c *poltergeist.Context) {
		if start, ok := c.Get("request_start"); ok {
			duration := time.Since(start.(time.Time))
			log.Printf("Request completed in %v", duration)
		}
	})

	// Error handler
	pipeline.OnError(func(c *poltergeist.Context) {
		if err, ok := c.Get("error"); ok {
			log.Printf("Error occurred: %v", err)
		}
	})

	// Server lifecycle hooks
	pipeline.OnServerStart(func() {
		log.Println("👻 Server started successfully!")
	})

	pipeline.OnServerStop(func() {
		log.Println("👻 Server shutting down...")
	})
}

func setupRoutes(app *poltergeist.Server) {
	// Health check
	app.GET("/health", func(c *poltergeist.Context) error {
		return c.JSON(200, poltergeist.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	}).Name("Health Check").Desc("Returns server health status").Tag("System")

	// Root route
	app.GET("/", func(c *poltergeist.Context) error {
		return c.JSON(200, poltergeist.H{
			"message": "Welcome to Poltergeist! 👻",
			"version": poltergeist.Version,
			"docs":    "/swagger",
		})
	}).Name("Welcome").Desc("Welcome endpoint").Tag("General")

	// API v1 group
	v1 := app.Group("/api/v1")
	{
		// Users group
		users := v1.Group("/users")
		{
			users.GET("", listUsers).Name("List Users").Desc("Get all users").Tag("Users")
			users.GET("/:id", getUser).Name("Get User").Desc("Get user by ID").Tag("Users")
			users.POST("", createUser).Name("Create User").Desc("Create a new user").Tag("Users").
				Request(CreateUserRequest{}).Response(User{})
			users.PUT("/:id", updateUser).Name("Update User").Desc("Update an existing user").Tag("Users")
			users.DELETE("/:id", deleteUser).Name("Delete User").Desc("Delete a user").Tag("Users")
		}

		// Protected routes with rate limiting
		protected := v1.Group("/protected", middleware.RateLimitPerRoute(5, 10))
		{
			protected.GET("/data", func(c *poltergeist.Context) error {
				return c.JSON(200, poltergeist.H{
					"message": "This is rate-limited data",
					"limit":   "5 requests per second",
				})
			}).Name("Protected Data").Desc("Rate-limited endpoint").Tag("Protected")
		}
	}

	// Static files example (if you have a static folder)
	// app.Static("/static", "./static")
}

func listUsers(c *poltergeist.Context) error {
	// Query parameters
	limit := c.QueryIntDefault("limit", 10)
	offset := c.QueryIntDefault("offset", 0)

	// Paginate
	end := offset + limit
	if end > len(users) {
		end = len(users)
	}
	if offset > len(users) {
		offset = len(users)
	}

	return c.JSON(200, poltergeist.H{
		"users":  users[offset:end],
		"total":  len(users),
		"limit":  limit,
		"offset": offset,
	})
}

func getUser(c *poltergeist.Context) error {
	id, err := c.ParamInt("id")
	if err != nil {
		return c.BadRequest("Invalid user ID")
	}

	for _, user := range users {
		if user.ID == id {
			return c.JSON(200, user)
		}
	}

	return c.NotFound("User not found")
}

func createUser(c *poltergeist.Context) error {
	var req CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return c.BadRequest("Invalid request body")
	}

	if req.Name == "" || req.Email == "" {
		return c.BadRequest("Name and email are required")
	}

	user := User{
		ID:        nextUserID,
		Name:      req.Name,
		Email:     req.Email,
		CreatedAt: time.Now(),
	}
	nextUserID++
	users = append(users, user)

	return c.JSON(201, user)
}

func updateUser(c *poltergeist.Context) error {
	id, err := c.ParamInt("id")
	if err != nil {
		return c.BadRequest("Invalid user ID")
	}

	var req CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return c.BadRequest("Invalid request body")
	}

	for i, user := range users {
		if user.ID == id {
			if req.Name != "" {
				users[i].Name = req.Name
			}
			if req.Email != "" {
				users[i].Email = req.Email
			}
			return c.JSON(200, users[i])
		}
	}

	return c.NotFound("User not found")
}

func deleteUser(c *poltergeist.Context) error {
	id, err := c.ParamInt("id")
	if err != nil {
		return c.BadRequest("Invalid user ID")
	}

	for i, user := range users {
		if user.ID == id {
			users = append(users[:i], users[i+1:]...)
			return c.NoContent()
		}
	}

	return c.NotFound("User not found")
}

// WebSocket chat hub
var wsHub = poltergeist.NewWSHub()

func setupWebSocket(app *poltergeist.Server) {
	// Start the hub
	go wsHub.Run()

	// WebSocket endpoint
	app.WebSocketWithHub("/ws/chat", wsHub, func(conn *poltergeist.WSConn, messageType int, message []byte) {
		// Broadcast message to all connected clients
		msg := Message{
			User:    "Anonymous",
			Content: string(message),
			Time:    time.Now().Format("15:04:05"),
		}
		wsHub.BroadcastJSON(msg)
	})

	// WebSocket info endpoint
	app.GET("/ws/info", func(c *poltergeist.Context) error {
		return c.JSON(200, poltergeist.H{
			"connected_clients": wsHub.ConnectionCount(),
			"endpoint":          "ws://localhost:8080/ws/chat",
		})
	}).Name("WebSocket Info").Desc("Get WebSocket connection info").Tag("WebSocket")
}

// SSE hub
var sseHub = poltergeist.NewSSEHub()

func setupSSE(app *poltergeist.Server) {
	// Start the hub
	go sseHub.Run()

	// Start a goroutine to send time updates
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for range ticker.C {
			sseHub.BroadcastEvent("time", poltergeist.H{
				"time": time.Now().Format(time.RFC3339),
			})
		}
	}()

	// SSE endpoint
	app.SSEWithHub("/sse/events", sseHub, func(c *poltergeist.Context, sse *poltergeist.SSEWriter) {
		// Send welcome message
		sse.SendEvent("welcome", poltergeist.H{
			"message": "Connected to SSE stream",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// SSE info endpoint
	app.GET("/sse/info", func(c *poltergeist.Context) error {
		return c.JSON(200, poltergeist.H{
			"connected_clients": sseHub.ClientCount(),
			"endpoint":          "http://localhost:8080/sse/events",
		})
	}).Name("SSE Info").Desc("Get SSE connection info").Tag("SSE")

	// Trigger SSE event manually
	app.POST("/sse/trigger", func(c *poltergeist.Context) error {
		var data map[string]interface{}
		if err := c.Bind(&data); err != nil {
			return c.BadRequest("Invalid request body")
		}

		event := c.QueryDefault("event", "message")
		sseHub.BroadcastEvent(event, data)

		return c.JSON(200, poltergeist.H{
			"message":   "Event sent",
			"event":     event,
			"data":      data,
			"receivers": sseHub.ClientCount(),
		})
	}).Name("Trigger SSE Event").Desc("Send an event to all SSE clients").Tag("SSE")
}
