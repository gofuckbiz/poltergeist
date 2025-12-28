package poltergeist

import (
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// =============================================================================
// EVENT PIPELINE TESTS
// =============================================================================

func TestEventPipeline_On(t *testing.T) {
	pipeline := NewEventPipeline()
	called := false

	pipeline.On(EventBeforeRequest, func(c *Context) {
		called = true
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	pipeline.Emit(EventBeforeRequest, c)

	if !called {
		t.Error("Handler was not called")
	}
}

func TestEventPipeline_MultipleHandlers(t *testing.T) {
	pipeline := NewEventPipeline()
	var count int

	pipeline.On(EventBeforeRequest, func(c *Context) { count++ })
	pipeline.On(EventBeforeRequest, func(c *Context) { count++ })
	pipeline.On(EventBeforeRequest, func(c *Context) { count++ })

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	pipeline.Emit(EventBeforeRequest, c)

	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
}

func TestEventPipeline_Off(t *testing.T) {
	pipeline := NewEventPipeline()
	called := false

	pipeline.On(EventBeforeRequest, func(c *Context) {
		called = true
	})
	pipeline.Off(EventBeforeRequest)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	pipeline.Emit(EventBeforeRequest, c)

	if called {
		t.Error("Handler should not be called after Off()")
	}
}

func TestEventPipeline_Clear(t *testing.T) {
	pipeline := NewEventPipeline()

	pipeline.On(EventBeforeRequest, func(c *Context) {})
	pipeline.On(EventAfterRequest, func(c *Context) {})

	if !pipeline.HasHandlers(EventBeforeRequest) {
		t.Error("Should have BeforeRequest handlers")
	}

	pipeline.Clear()

	if pipeline.HasHandlers(EventBeforeRequest) {
		t.Error("Should not have handlers after Clear()")
	}
}

func TestEventPipeline_ConvenienceMethods(t *testing.T) {
	pipeline := NewEventPipeline()
	var beforeCalled, afterCalled, errorCalled bool

	pipeline.BeforeRequest(func(c *Context) { beforeCalled = true })
	pipeline.AfterRequest(func(c *Context) { afterCalled = true })
	pipeline.OnError(func(c *Context) { errorCalled = true })

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	pipeline.Emit(EventBeforeRequest, c)
	pipeline.Emit(EventAfterRequest, c)
	pipeline.Emit(EventError, c)

	if !beforeCalled || !afterCalled || !errorCalled {
		t.Error("Not all handlers were called")
	}
}

// =============================================================================
// EVENT PIPELINE BENCHMARKS
// =============================================================================

func BenchmarkEventPipeline_Emit(b *testing.B) {
	pipeline := NewEventPipeline()
	pipeline.On(EventBeforeRequest, func(c *Context) {})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		pipeline.Emit(EventBeforeRequest, c)
	}
}

func BenchmarkEventPipeline_EmitMultipleHandlers(b *testing.B) {
	pipeline := NewEventPipeline()

	// Add 10 handlers
	for i := 0; i < 10; i++ {
		pipeline.On(EventBeforeRequest, func(c *Context) {})
	}

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		pipeline.Emit(EventBeforeRequest, c)
	}
}

func BenchmarkEventPipeline_EmitAsync(b *testing.B) {
	pipeline := NewEventPipeline()
	var count int64

	pipeline.On(EventBeforeRequest, func(c *Context) {
		atomic.AddInt64(&count, 1)
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		pipeline.EmitAsync(EventBeforeRequest, c)
	}
}
