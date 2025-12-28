package poltergeist

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"
)

// =============================================================================
// CONTEXT TESTS
// =============================================================================

func TestContext_Query(t *testing.T) {
	req := httptest.NewRequest("GET", "/?name=john&age=25&active=true", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	// Test Query
	if got := c.Query("name"); got != "john" {
		t.Errorf("Query('name') = %q, want %q", got, "john")
	}

	// Test QueryDefault
	if got := c.QueryDefault("missing", "default"); got != "default" {
		t.Errorf("QueryDefault('missing', 'default') = %q, want %q", got, "default")
	}

	// Test QueryInt
	if got, err := c.QueryInt("age"); err != nil || got != 25 {
		t.Errorf("QueryInt('age') = %d, %v, want 25, nil", got, err)
	}

	// Test QueryIntDefault
	if got := c.QueryIntDefault("missing", 10); got != 10 {
		t.Errorf("QueryIntDefault('missing', 10) = %d, want 10", got)
	}

	// Test QueryBool
	if got := c.QueryBool("active"); !got {
		t.Errorf("QueryBool('active') = %v, want true", got)
	}
}

func TestContext_Param(t *testing.T) {
	req := httptest.NewRequest("GET", "/users/123", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)
	c.Params = map[string]string{"id": "123"}

	// Test Param
	if got := c.Param("id"); got != "123" {
		t.Errorf("Param('id') = %q, want %q", got, "123")
	}

	// Test ParamInt
	if got, err := c.ParamInt("id"); err != nil || got != 123 {
		t.Errorf("ParamInt('id') = %d, %v, want 123, nil", got, err)
	}
}

func TestContext_JSON(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	data := H{"message": "hello"}
	if err := c.JSON(200, data); err != nil {
		t.Fatalf("JSON() error = %v", err)
	}

	if w.Code != 200 {
		t.Errorf("Status code = %d, want 200", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}

	if !strings.Contains(w.Body.String(), `"message":"hello"`) {
		t.Errorf("Body = %q, want to contain message:hello", w.Body.String())
	}
}

func TestContext_Bind(t *testing.T) {
	body := `{"name":"John","email":"john@example.com"}`
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	var data struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := c.Bind(&data); err != nil {
		t.Fatalf("Bind() error = %v", err)
	}

	if data.Name != "John" {
		t.Errorf("Name = %q, want %q", data.Name, "John")
	}
	if data.Email != "john@example.com" {
		t.Errorf("Email = %q, want %q", data.Email, "john@example.com")
	}
}

func TestContext_SetGet(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	// Test Set and Get
	c.Set("key", "value")
	if got, exists := c.Get("key"); !exists || got != "value" {
		t.Errorf("Get('key') = %v, %v, want 'value', true", got, exists)
	}

	// Test GetString
	c.Set("str", "hello")
	if got := c.GetString("str"); got != "hello" {
		t.Errorf("GetString('str') = %q, want %q", got, "hello")
	}

	// Test GetInt
	c.Set("num", 42)
	if got := c.GetInt("num"); got != 42 {
		t.Errorf("GetInt('num') = %d, want 42", got)
	}
}

func TestContext_ErrorResponses(t *testing.T) {
	tests := []struct {
		name     string
		fn       func(*Context) error
		wantCode int
	}{
		{"BadRequest", func(c *Context) error { return c.BadRequest("bad") }, 400},
		{"Unauthorized", func(c *Context) error { return c.Unauthorized("unauth") }, 401},
		{"Forbidden", func(c *Context) error { return c.Forbidden("forbidden") }, 403},
		{"NotFound", func(c *Context) error { return c.NotFound("not found") }, 404},
		{"InternalServerError", func(c *Context) error { return c.InternalServerError("error") }, 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()
			c := NewContext(w, req)

			tt.fn(c)

			if w.Code != tt.wantCode {
				t.Errorf("Status code = %d, want %d", w.Code, tt.wantCode)
			}
		})
	}
}

// =============================================================================
// CONTEXT BENCHMARKS
// =============================================================================

func BenchmarkContext_Query(b *testing.B) {
	req := httptest.NewRequest("GET", "/?name=john&age=25", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = c.Query("name")
	}
}

func BenchmarkContext_QueryInt(b *testing.B) {
	req := httptest.NewRequest("GET", "/?age=25", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = c.QueryInt("age")
	}
}

func BenchmarkContext_Param(b *testing.B) {
	req := httptest.NewRequest("GET", "/users/123", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)
	c.Params = map[string]string{"id": "123"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = c.Param("id")
	}
}

func BenchmarkContext_JSON(b *testing.B) {
	data := H{"message": "hello", "status": "ok"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		c := NewContext(w, req)
		_ = c.JSON(200, data)
	}
}

func BenchmarkContext_Bind(b *testing.B) {
	body := `{"name":"John","email":"john@example.com"}`

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		c := NewContext(w, req)

		var data struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		_ = c.Bind(&data)
	}
}

func BenchmarkContext_SetGet(b *testing.B) {
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		c.Set("key", "value")
		_, _ = c.Get("key")
	}
}
