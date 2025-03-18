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
	logger "github.com/project-cdim/cdim-go-logger"
	logger_common "github.com/project-cdim/cdim-go-logger/common"
)

// Application Logger
var log, _ = logger.New(logger_common.Option{Tag: logger_common.TAG_APP_EXPORTER /*, LoggingLevel: logger_common.DEBUG*/})
