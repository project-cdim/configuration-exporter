// Copyright (C) 2026 NEC Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	loggercommon "github.com/project-cdim/cdim-go-logger/common"
)

func TestTracingMiddleware(t *testing.T) {
	tests := []struct {
		name              string
		traceparentHeader string
		wantInContext     bool
	}{
		{
			name:              "traceparent header exists",
			traceparentHeader: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			wantInContext:     true,
		},
		{
			name:              "traceparent header is empty",
			traceparentHeader: "",
			wantInContext:     false,
		},
		{
			name:              "traceparent header with different format",
			traceparentHeader: "test-trace-id",
			wantInContext:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest("GET", "/test", nil)
			if tt.traceparentHeader != "" {
				req.Header.Set("traceparent", tt.traceparentHeader)
			}
			c.Request = req

			middleware := TracingMiddleware()
			var capturedContext context.Context
			middleware(c)
			c.Next()
			capturedContext = c.Request.Context()

			traceparent := loggercommon.TraceparentFromContext(capturedContext)
			if tt.wantInContext {
				if traceparent != tt.traceparentHeader {
					t.Errorf("loggercommon.TraceparentFromContext() = %v, want %v", traceparent, tt.traceparentHeader)
				}
			} else {
				if traceparent != "" {
					t.Errorf("loggercommon.TraceparentFromContext() = %v, want empty string", traceparent)
				}
			}
		})
	}
}

func TestTracingMiddleware_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TracingMiddleware())

	var capturedTraceparent string
	router.GET("/test", func(c *gin.Context) {
		capturedTraceparent = loggercommon.TraceparentFromContext(c.Request.Context())
		c.String(http.StatusOK, "ok")
	})

	t.Run("integration test with traceparent", func(t *testing.T) {
		expectedTraceparent := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("traceparent", expectedTraceparent)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
		}
		if capturedTraceparent != expectedTraceparent {
			t.Errorf("Expected traceparent %s, got %s", expectedTraceparent, capturedTraceparent)
		}
	})

	t.Run("integration test without traceparent", func(t *testing.T) {
		capturedTraceparent = "should-be-cleared"
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
		}
		if capturedTraceparent != "" {
			t.Errorf("Expected empty traceparent, got %s", capturedTraceparent)
		}
	})
}
