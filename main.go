// Copyright (C) 2025 NEC Corporation.
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

package main

import (
	"github.com/project-cdim/configuration-exporter/controller"

	logger "github.com/project-cdim/cdim-go-logger"
	logger_common "github.com/project-cdim/cdim-go-logger/common"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// v1 route base url
const URL_BASE_V1 = "/cdim/api/v1"

// Audit Trail Logger
var log, _ = logger.New(logger_common.Option{Tag: logger_common.TAG_TRAIL})

func main() {
	// Create an instance of gin Engine
	router := gin.Default()
	// Add custom middleware to gin Engine for logging
	router.Use(logMiddleware())

	router.Use(cors.New(cors.Config{
		// Allowed HTTP methods
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS"},
		// Allowed origins
		AllowOrigins: []string{
			"*",
		},
		// Allowed HTTP request headers
		AllowHeaders: []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
	}))

	// v1 route group
	v1 := router.Group(URL_BASE_V1)
	// API to get devices data and to forward that data
	v1.POST("/devices/sync", controller.SyncDevices)

	router.Run(":8080") // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

// custom middleware for gin
// * Process Description
//   - Before API execution : logging the start of API
//   - After API execution  : logging the end of API
func logMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		log.TrailReq(ctx.Request.Method, ctx.Request.URL.Path, "-", "request start.")
		ctx.Next()
		log.TrailRes(ctx.Writer.Status(), "response end.")
	}
}
