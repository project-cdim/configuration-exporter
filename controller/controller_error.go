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
        
package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// Error object for Exporter
type ExpError struct {
	StatusCode int
	Code       string
	Message    string
}

// Implementation of the Error method in the error interface
func (gce *ExpError) Error() string {
	return fmt.Sprintf("http status code = %d, code = %s message = %s", gce.StatusCode, gce.Code, gce.Message)
}

// Create a new ExpError
func ExpErrorNew(statusCode int, code string, message string) error {
	return &ExpError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
	}
}

// Return the StatusCode value from ExpError
func GetStatusCode(err error) int {
	res := 0
	switch castErr := err.(type) {
	case *ExpError:
		res = castErr.StatusCode
	}
	return res
}

// Convert Code and Message of ExpError to gin.H (map format) and return
func ToJson(err error) gin.H {
	var res gin.H
	switch castErr := err.(type) {
	case *ExpError:
		res = gin.H{
			"code":    castErr.Code,
			"message": castErr.Message,
		}
	}
	return res
}
