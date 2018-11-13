package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPanic(t *testing.T) {
	crashedHandler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			panic("hello world")
	})

	middleware := NewPanic(crashedHandler, t.Log)

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)

	middleware.ServeHTTP(resp, req)
}
