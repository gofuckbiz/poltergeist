<h1 align="center">
  <br>
  <span style="font-size: 80px;">👻</span>
  <br>
  Poltergeist
  <br>
</h1>

<p align="center">
  <strong>High-performance Realtime & REST Go Framework</strong>
</p>

<p align="center">
  A lightweight, developer-first Go framework for building REST APIs and Realtime applications with built-in WebSocket, SSE, and auto-generated documentation.
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/gofuckbiz/poltergeist"><img src="https://pkg.go.dev/badge/github.com/gofuckbiz/poltergeist.svg" alt="Go Reference"></a>
  <a href="#-installation"><img src="https://img.shields.io/badge/go-%3E%3D1.21-00ADD8?style=flat-square&logo=go" alt="Go Version"></a>
  <a href="https://goreportcard.com/report/github.com/gofuckbiz/poltergeist"><img src="https://goreportcard.com/badge/github.com/gofuckbiz/poltergeist" alt="Go Report Card"></a>
  <img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/version-1.0.1-blue?style=flat-square" alt="Version">
</p>

<p align="center">
  <a href="#-features">Features</a> •
  <a href="#-installation">Installation</a> •
  <a href="#-getting-started">Getting Started</a> •
  <a href="#-documentation">Documentation</a> •
  <a href="#-examples">Examples</a> •
  <a href="#-contributing">Contributing</a>
</p>

---

## 🎯 What is Poltergeist?

**Poltergeist** is a high-performance, lightweight Go framework designed for developers who want to build REST APIs and Realtime applications without the hassle of configuring multiple libraries.

Unlike traditional Go web frameworks, Poltergeist comes with **WebSocket and SSE support out of the box**, an **event-driven pipeline** for request lifecycle hooks, and **automatic OpenAPI/Swagger documentation generation** — all with minimal boilerplate.

Whether you're building microservices, real-time chat applications, or desktop server apps, Poltergeist provides a production-ready foundation that compiles into a **single binary**.

---

## ✨ Features

| Feature | Description |
|---------|-------------|
| 🚀 **Zero-Config** | Minimal boilerplate, one method to start |
| 🔌 **Realtime Built-in** | Native WebSocket & SSE support |
| 📡 **Event Pipeline** | Before/after request hooks, error handling |
| 📚 **Auto Documentation** | OpenAPI/Swagger auto-generation |
| 🛡️ **Rich Middleware** | Logger, Recovery, CORS, RateLimit, Auth |
| 🎯 **Developer-First** | Intuitive API, powerful helpers |
| 📦 **Single Binary** | Compiles to one executable file |
| ⚡ **High Performance** | Built on Go's net/http with optimizations |

---

## 📦 Installation

### Requirements

- **Go 1.21** or higher

### Install

```bash
go get github.com/gofuckbiz/poltergeist
```

### Verify Installation

```bash
go mod tidy
```

---

## 🚀 Getting Started

Here's a complete example showing the core features of Poltergeist:

```go
package main

import (
    "log"
    "time"

    "github.com/gofuckbiz/poltergeist"
    "github.com/gofuckbiz/poltergeist/docs"
    "github.com/gofuckbiz/poltergeist/middleware"
)

// User model
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// Request body for creating users
type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func main() {
    // Create new Poltergeist server
    app := poltergeist.New()

    // Add global middleware
    app.Use(middleware.Logger())    // Request logging
    app.Use(middleware.Recovery())  // Panic recovery
    app.Use(middleware.CORS())      // CORS support

    // Setup event hooks
    app.Pipeline().BeforeRequest(func(c *poltergeist.Context) {
        c.Set("start", time.Now())
    })

    app.Pipeline().AfterRequest(func(c *poltergeist.Context) {
        start := c.MustGet("start").(time.Time)
        log.Printf("Request completed in %v", time.Since(start))
    })

    // Root endpoint
    app.GET("/", func(c *poltergeist.Context) error {
        return c.JSON(200, poltergeist.H{
            "message": "Welcome to Poltergeist! 👻",
            "docs":    "/swagger",
        })
    })

    // API group with versioning
    v1 := app.Group("/api/v1")
    {
        // User endpoints
        v1.GET("/users", listUsers).
            Name("List Users").
            Tag("Users")

        v1.GET("/users/:id", getUser).
            Name("Get User").
            Tag("Users")

        v1.POST("/users", createUser).
            Name("Create User").
            Tag("Users").
            Request(CreateUserRequest{}).
            Response(User{})
    }

    // WebSocket chat endpoint
    hub := poltergeist.NewWSHub()
    go hub.Run()

    app.WebSocketWithHub("/ws/chat", hub, func(conn *poltergeist.WSConn, _ int, msg []byte) {
        hub.BroadcastJSON(poltergeist.H{
            "message": string(msg),
            "time":    time.Now().Format("15:04:05"),
        })
    })

    // SSE events endpoint
    sseHub := poltergeist.NewSSEHub()
    go sseHub.Run()

    app.SSEWithHub("/sse/events", sseHub, func(c *poltergeist.Context, sse *poltergeist.SSEWriter) {
        sse.SendEvent("welcome", poltergeist.H{"message": "Connected!"})
    })

    // Enable Swagger documentation
    docs.Swagger(app, &docs.SwaggerConfig{
        Title:       "My API",
        Description: "API built with Poltergeist",
        Version:     "1.0.0",
    })

    // Start server
    log.Fatal(app.Run(":8080"))
}

// Handlers
func listUsers(c *poltergeist.Context) error {
    users := []User{
        {ID: 1, Name: "John", Email: "john@example.com"},
        {ID: 2, Name: "Jane", Email: "jane@example.com"},
    }
    return c.JSON(200, users)
}

func getUser(c *poltergeist.Context) error {
    id, err := c.ParamInt("id")
    if err != nil {
        return c.BadRequest("Invalid user ID")
    }
    return c.JSON(200, User{ID: id, Name: "John", Email: "john@example.com"})
}

func createUser(c *poltergeist.Context) error {
    var req CreateUserRequest
    if err := c.Bind(&req); err != nil {
        return c.BadRequest("Invalid request body")
    }
    user := User{ID: 1, Name: req.Name, Email: req.Email}
    return c.JSON(201, user)
}
```

Run the server:

```bash
go run main.go
```

Then visit:
- **API**: http://localhost:8080
- **Swagger UI**: http://localhost:8080/swagger
- **WebSocket**: Connect to `ws://localhost:8080/ws/chat`
- **SSE**: Connect to `http://localhost:8080/sse/events`

---

## 📖 Documentation

### Quick Reference

<details>
<summary><strong>🛣️ Routing</strong></summary>

```go
app.GET("/path", handler)
app.POST("/path", handler)
app.PUT("/path/:id", handler)
app.DELETE("/path/:id", handler)
app.PATCH("/path/:id", handler)

// Route groups
api := app.Group("/api", middleware.Logger())
api.GET("/users", listUsers)

// Static files
app.Static("/static", "./public")
```

</details>

<details>
<summary><strong>📝 Context Helpers</strong></summary>

```go
// Path & Query params
id := c.Param("id")
page := c.QueryIntDefault("page", 1)

// Request body
var data MyStruct
c.Bind(&data)

// Headers
auth := c.Header("Authorization")
c.SetHeader("X-Custom", "value")

// Responses
c.JSON(200, data)
c.String(200, "Hello")
c.HTML(200, "<h1>Hi</h1>")
c.NoContent()
c.Redirect(302, "/new")

// Errors
c.BadRequest("message")
c.Unauthorized("message")
c.NotFound("message")
```

</details>

<details>
<summary><strong>🛡️ Middleware</strong></summary>

```go
// Available middleware
middleware.Logger()          // Request logging
middleware.Recovery()        // Panic recovery
middleware.CORS()           // CORS headers
middleware.RateLimit()      // Rate limiting
middleware.BasicAuth(fn)    // Basic authentication
middleware.BearerAuth(fn)   // Bearer token auth
middleware.APIKeyAuth(fn)   // API key auth
middleware.Secure()         // Security headers
middleware.Gzip()           // Compression
middleware.Timeout(dur)     // Request timeout
middleware.RequestID()      // Unique request ID
```

</details>

<details>
<summary><strong>🔌 WebSocket</strong></summary>

```go
hub := poltergeist.NewWSHub()
go hub.Run()

app.WebSocketWithHub("/ws", hub, func(conn *poltergeist.WSConn, msgType int, msg []byte) {
    // Handle message
    conn.SendJSON(data)
    hub.Broadcast(message)
    hub.BroadcastToRoom("room", message)
})

// Rooms
hub.JoinRoom(conn, "room1")
hub.LeaveRoom(conn, "room1")
```

</details>

<details>
<summary><strong>📡 Server-Sent Events</strong></summary>

```go
sseHub := poltergeist.NewSSEHub()
go sseHub.Run()

app.SSEWithHub("/sse", sseHub, func(c *poltergeist.Context, sse *poltergeist.SSEWriter) {
    sse.SendEvent("init", data)
})

// Broadcast
sseHub.BroadcastEvent("update", data)
sseHub.BroadcastToRoom("room", event)
```

</details>

<details>
<summary><strong>📡 Event Pipeline</strong></summary>

```go
pipeline := app.Pipeline()

pipeline.BeforeRequest(func(c *poltergeist.Context) {
    // Before each request
})

pipeline.AfterRequest(func(c *poltergeist.Context) {
    // After each request
})

pipeline.OnError(func(c *poltergeist.Context) {
    // On error
})

pipeline.OnServerStart(func() { /* Server started */ })
pipeline.OnServerStop(func() { /* Server stopping */ })
```

</details>

<details>
<summary><strong>⚙️ Configuration</strong></summary>

```go
config := &poltergeist.Config{
    Addr:             ":8080",
    ReadTimeout:      30 * time.Second,
    WriteTimeout:     30 * time.Second,
    GracefulShutdown: true,
    ShutdownTimeout:  30 * time.Second,
    DevMode:          true,
}

app := poltergeist.NewWithConfig(config)

// TLS
app.RunTLS(":443", "cert.pem", "key.pem")
```

</details>

---

## 📚 Examples

### Full Example Application

The `examples/` directory contains a complete demo application showcasing all features:

```bash
cd examples
go run main.go
```

### HTML Clients

- **WebSocket Chat**: Open `examples/websocket_client.html` in your browser
- **SSE Events Viewer**: Open `examples/sse_client.html` in your browser

### More Examples

| Example | Description |
|---------|-------------|
| [Basic REST API](examples/main.go) | CRUD operations with JSON |
| [WebSocket Chat](examples/main.go#L130) | Real-time chat with rooms |
| [SSE Events](examples/main.go#L150) | Server-sent events streaming |
| [Swagger Docs](examples/main.go#L160) | Auto-generated API documentation |

---

## ⚡ Benchmarks

Performance on AMD Ryzen 5 3600 (Windows, Go 1.22):

| Benchmark | Operations/sec | Time/op | Allocs/op |
|-----------|---------------|---------|-----------|
| **Static Route** | 1,318,196 | 961 ns | 13 |
| **Param Route** | 854,396 | 1,209 ns | 16 |
| **Full JSON Response** | 430,201 | 2,928 ns | 33 |
| **String Response** | 1,270,788 | 884 ns | 13 |
| **Path Matching (static)** | 28,118,124 | 44 ns | 1 |
| **Event Pipeline Emit** | 100,000,000 | 11.5 ns | 0 |
| **Context Get/Set** | 34,153,198 | 35 ns | 0 |

Run benchmarks yourself:

```bash
go test -run=^$ -bench=Benchmark -benchmem
```

---

## 🔄 Comparison

| Feature | Poltergeist | Gin / Fiber |
|---------|-------------|-------------|
| **Realtime Support** | ✅ Built-in WebSocket + SSE | ❌ Requires external libs |
| **Event Pipeline** | ✅ Before/after hooks | ❌ Not available |
| **Auto Documentation** | ✅ OpenAPI generation | ❌ Requires external libs |
| **Zero Config** | ✅ Minimal setup | ⚠️ More configuration |
| **Single Binary** | ✅ Yes | ✅ Yes |

---

## 📁 Project Structure

```
poltergeist/
├── poltergeist.go      # Main entry & shortcuts
├── constants.go        # Shared constants (DRY)
├── interfaces.go       # Interface definitions (SOLID)
├── context.go          # Request context & helpers
├── router.go           # Router & route groups
├── server.go           # HTTP server
├── events.go           # Event pipeline
├── hub.go              # Base hub for realtime (DRY)
├── websocket.go        # WebSocket support
├── sse.go              # SSE support
├── middleware/         # Built-in middleware
│   ├── logging.go
│   ├── recovery.go
│   ├── ratelimit.go
│   ├── cors.go
│   ├── auth.go
│   └── middleware.go   # Utility middleware
├── docs/               # Documentation
│   └── swagger.go      # OpenAPI generation
└── examples/           # Example applications
    ├── main.go
    ├── websocket_client.html
    └── sse_client.html
```

---

## 🤝 Contributing

We welcome contributions! Here's how you can help:

1. **Fork** the repository
2. **Create** a feature branch: `git checkout -b feature/amazing-feature`
3. **Commit** your changes: `git commit -m 'Add amazing feature'`
4. **Push** to the branch: `git push origin feature/amazing-feature`
5. **Open** a Pull Request

### Guidelines

- Write tests for new features
- Follow Go conventions and formatting (`go fmt`)
- Update documentation as needed
- Keep commits atomic and descriptive

### Reporting Issues

Found a bug? Have a suggestion? [Open an issue](https://github.com/gofuckbiz/poltergeist/issues/new) with details.

---

## 📝 License

This project is licensed under the **MIT License** - see the [LICENSE](LICENSE) file for details.

```
MIT License

Copyright (c) 2025 Poltergeist Framework

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.
```

---

<p align="center">
  <sub>Built with 👻 and ❤️ by the Poltergeist Team</sub>
</p>

<p align="center">
  <a href="#-poltergeist">Back to top ⬆️</a>
</p>
