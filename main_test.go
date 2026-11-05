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

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_main(t *testing.T) {
	t.Skip("not test")
}

func Test_setupEngine(t *testing.T) {
	t.Skip("not test")
}

func Test_logMiddleware(t *testing.T) {
	t.Skip("not test")
}

func TestMustNewTrailLogger_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("mustNewTrailLogger() panicked unexpectedly: %v", r)
		}
	}()

	if got := mustNewTrailLogger(); got == nil {
		t.Fatalf("mustNewTrailLogger() returned nil logger")
	}
}

func TestMustNewTrailLogger_PanicWhenConfigFileNotFound(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	configFile := filepath.Clean(filepath.Join(wd, "configs", "logger_settings_trail.yaml"))

	if _, err := os.Stat(configFile); err != nil {
		t.Skipf("config file %s does not exist, skipping test", configFile)
		return
	}

	tempBackup := configFile + ".bak"
	if err := os.Rename(configFile, tempBackup); err != nil {
		t.Fatalf("failed to backup config file: %v", err)
	}
	defer func() {
		if err := os.Rename(tempBackup, configFile); err != nil {
			t.Fatalf("failed to restore config file: %v", err)
		}
	}()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("mustNewTrailLogger() did not panic when config file is missing")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("panic type = %T, want string", r)
		}
		if !strings.Contains(msg, "failed to initialize trail logger from") {
			t.Fatalf("panic message should contain 'failed to initialize trail logger from', got: %s", msg)
		}
	}()

	mustNewTrailLogger()
}
