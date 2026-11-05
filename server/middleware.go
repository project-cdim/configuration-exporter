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

	"github.com/gin-gonic/gin"
	loggercommon "github.com/project-cdim/cdim-go-logger/common"
)

// TracingMiddleware extracts the traceparent header from an HTTP request and injects it into the request context.
//
// This middleware should be registered early in the middleware chain to ensure
// traceparent is available throughout the request lifecycle for distributed tracing.
// The traceparent value follows the W3C Trace Context specification and is used
// to correlate logs and operations across service boundaries.
//
// Returns:
//   - gin.HandlerFunc: A Gin middleware function that can be registered with gin.Engine
func TracingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceparent := c.GetHeader("traceparent")
		if traceparent != "" {
			ctx := context.WithValue(c.Request.Context(), loggercommon.TraceparentKey, traceparent)
			c.Request = c.Request.WithContext(ctx)
		}
		c.Next()
	}
}
