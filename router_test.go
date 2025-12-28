package poltergeist

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// =============================================================================
// ROUTER TESTS
// =============================================================================

func TestRouter_BasicRouting(t *testing.T) {
	router := NewRouter()

	router.GET("/", func(c *Context) error {
		return c.String(200, "home")
	})

	router.GET("/users", func(c *Context) error {
		return c.String(200, "users")
	})

	router.POST("/users", func(c *Context) error {
		return c.String(201, "created")
	})

	tests := []struct {
		method   string
		path     string
		wantCode int
		wantBody string
	}{
		{"GET", "/", 200, "home"},
		{"GET", "/users", 200, "users"},
		{"POST", "/users", 201, "created"},
		{"GET", "/notfound", 404, ""},
		{"POST", "/", 405, ""},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantCode)
			}
		})
	}
}

func TestRouter_PathParams(t *testing.T) {
	router := NewRouter()

	router.GET("/users/:id", func(c *Context) error {
		return c.String(200, "user:"+c.Param("id"))
	})

	router.GET("/users/:id/posts/:postId", func(c *Context) error {
		return c.String(200, c.Param("id")+":"+c.Param("postId"))
	})

	tests := []struct {
		path     string
		wantBody string
	}{
		{"/users/123", "user:123"},
		{"/users/abc", "user:abc"},
		{"/users/123/posts/456", "123:456"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Body.String() != tt.wantBody {
				t.Errorf("Body = %q, want %q", w.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestRouter_Groups(t *testing.T) {
	router := NewRouter()

	api := router.Group("/api")
	{
		v1 := api.Group("/v1")
		{
			v1.GET("/users", func(c *Context) error {
				return c.String(200, "v1-users")
			})
		}

		v2 := api.Group("/v2")
		{
			v2.GET("/users", func(c *Context) error {
				return c.String(200, "v2-users")
			})
		}
	}

	tests := []struct {
		path     string
		wantBody string
	}{
		{"/api/v1/users", "v1-users"},
		{"/api/v2/users", "v2-users"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != 200 {
				t.Errorf("Status = %d, want 200", w.Code)
			}
		})
	}
}

func TestRouter_Middleware(t *testing.T) {
	router := NewRouter()

	// Global middleware
	router.Use(func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			c.Set("global", true)
			return next(c)
		}
	})

	router.GET("/test", func(c *Context) error {
		if _, ok := c.Get("global"); !ok {
			return c.String(500, "middleware not applied")
		}
		return c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Status = %d, want 200", w.Code)
	}
}

func TestMatchPath(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		match   bool
		params  map[string]string
	}{
		{"/", "/", true, map[string]string{}},
		{"/users", "/users", true, map[string]string{}},
		{"/users/:id", "/users/123", true, map[string]string{"id": "123"}},
		{"/users/:id/posts/:postId", "/users/1/posts/2", true, map[string]string{"id": "1", "postId": "2"}},
		{"/static/*filepath", "/static/css/style.css", true, map[string]string{"filepath": "css/style.css"}},
		{"/users", "/posts", false, nil},
		{"/users/:id", "/users", false, nil},
		{"/users/:id", "/users/123/extra", false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"->"+tt.path, func(t *testing.T) {
			match, params := matchPath(tt.pattern, tt.path)

			if match != tt.match {
				t.Errorf("match = %v, want %v", match, tt.match)
			}

			if tt.match && tt.params != nil {
				for k, v := range tt.params {
					if params[k] != v {
						t.Errorf("params[%s] = %s, want %s", k, params[k], v)
					}
				}
			}
		})
	}
}

// =============================================================================
// ROUTER BENCHMARKS
// =============================================================================

func BenchmarkRouter_StaticRoute(b *testing.B) {
	router := NewRouter()
	router.GET("/users", func(c *Context) error {
		return c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/users", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_ParamRoute(b *testing.B) {
	router := NewRouter()
	router.GET("/users/:id", func(c *Context) error {
		return c.String(200, c.Param("id"))
	})

	req := httptest.NewRequest("GET", "/users/123", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_MultiParamRoute(b *testing.B) {
	router := NewRouter()
	router.GET("/users/:id/posts/:postId/comments/:commentId", func(c *Context) error {
		return c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/users/1/posts/2/comments/3", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_WithMiddleware(b *testing.B) {
	router := NewRouter()

	// Add 5 middlewares
	for i := 0; i < 5; i++ {
		router.Use(func(next HandlerFunc) HandlerFunc {
			return func(c *Context) error {
				return next(c)
			}
		})
	}

	router.GET("/users", func(c *Context) error {
		return c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/users", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkMatchPath_Static(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		matchPath("/users/list", "/users/list")
	}
}

func BenchmarkMatchPath_Params(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		matchPath("/users/:id/posts/:postId", "/users/123/posts/456")
	}
}

func BenchmarkMatchPath_Wildcard(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		matchPath("/static/*filepath", "/static/css/app/style.css")
	}
}

// =============================================================================
// FULL REQUEST BENCHMARKS
// =============================================================================

func BenchmarkFullRequest_JSON(b *testing.B) {
	router := NewRouter()
	router.GET("/api/users", func(c *Context) error {
		return c.JSON(200, H{
			"users": []H{
				{"id": 1, "name": "John"},
				{"id": 2, "name": "Jane"},
			},
		})
	})

	req := httptest.NewRequest("GET", "/api/users", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkFullRequest_String(b *testing.B) {
	router := NewRouter()
	router.GET("/ping", func(c *Context) error {
		return c.String(200, "pong")
	})

	req := httptest.NewRequest("GET", "/ping", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// =============================================================================
// COMPARATIVE BENCHMARK (simulate real-world API)
// =============================================================================

func BenchmarkRealWorld_RESTAPI(b *testing.B) {
	router := NewRouter()

	// Simulate middleware stack
	router.Use(func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			c.SetHeader("X-Request-ID", "123")
			return next(c)
		}
	})

	// Routes
	api := router.Group("/api/v1")
	api.GET("/users", func(c *Context) error {
		return c.JSON(200, H{"users": []H{{"id": 1, "name": "John"}}})
	})
	api.GET("/users/:id", func(c *Context) error {
		return c.JSON(200, H{"id": c.Param("id"), "name": "John"})
	})

	b.Run("ListUsers", func(b *testing.B) {
		req := httptest.NewRequest("GET", "/api/v1/users", nil)
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})

	b.Run("GetUser", func(b *testing.B) {
		req := httptest.NewRequest("GET", "/api/v1/users/123", nil)
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}

// =============================================================================
// HTTP SERVER INTEGRATION TEST
// =============================================================================

func TestServer_Integration(t *testing.T) {
	app := New()

	app.GET("/", func(c *Context) error {
		return c.JSON(200, H{"message": "hello"})
	})

	app.GET("/users/:id", func(c *Context) error {
		return c.JSON(200, H{"id": c.Param("id")})
	})

	// Create test server
	ts := httptest.NewServer(app.Router())
	defer ts.Close()

	// Test root
	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET / error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("GET / status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	// Test params
	resp, err = http.Get(ts.URL + "/users/123")
	if err != nil {
		t.Fatalf("GET /users/123 error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("GET /users/123 status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()
}
