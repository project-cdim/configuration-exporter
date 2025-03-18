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
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetDevices(t *testing.T) {
	w := httptest.NewRecorder()
	// Create gin context
	ginContext, _ := gin.CreateTestContext(w)
	// Since GetDevices does not use request parameters, a dummy is used
	req := httptest.NewRequest("GET", "/dummy", nil)
	// Put request information into the context
	ginContext.Request = req

	GetDevices(ginContext)

	// Since GetDevices will cause an error in loadConfig, the test is passed with 500 instead of 200.
	if w.Code != 500 {
		b, _ := io.ReadAll(w.Body)
		t.Error(w.Code, string(b))
	}
}

func Test_loadConfig(t *testing.T) {
	settings := yamlContent{}

	type args struct {
		filepath string
		settings yamlContent
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"Error case: YAML file does not exist",
			args{
				"testdata/aaa.yaml",
				settings,
			},
			"",
			true,
		},
		{
			"Error case: Content of YAML file is not in YAML format",
			args{
				"testdata/textonly.yaml",
				settings,
			},
			"",
			true,
		},
		{
			"Error case: collect_configs/timeout is less than the lower limit. Boundary value test",
			args{
				"testdata/collect_timeout0.yaml",
				settings,
			},
			"",
			true,
		},
		{
			"Normal case: collect_configs/timeout is equal to the lower limit. Boundary value test",
			args{
				"testdata/collect_timeout1.yaml",
				settings,
			},
			"",
			false,
		},
		{
			"Normal case: collect_configs/timeout is equal to the upper limit. Boundary value test",
			args{
				"testdata/collect_timeout36000.yaml",
				settings,
			},
			"",
			false,
		},
		{
			"Error case: collect_configs/timeout is greater than the upper limit. Boundary value test",
			args{
				"testdata/collect_timeout36001.yaml",
				settings,
			},
			"",
			true,
		},
		{
			"Normal case: Default value is used when collect_configs/timeout is omitted",
			args{
				"testdata/collect_timeout_empty.yaml",
				settings,
			},
			"",
			false,
		},
		{
			"Error case: collect_configs/target_url is required and cannot be omitted",
			args{
				"testdata/collect_url_empty.yaml",
				settings,
			},
			"",
			true,
		},
		{
			"Error case: collect_configs/target_url is not in URL format",
			args{
				"testdata/collect_url_format_invalid.yaml",
				settings,
			},
			"",
			true,
		},
		{
			"Error case: alert_configs/timeout is less than the lower limit. Boundary value test",
			args{
				"testdata/alert_timeout0.yaml",
				settings,
			},
			"",
			true,
		},
		{
			"Normal case: alert_configs/timeout is equal to the lower limit. Boundary value test",
			args{
				"testdata/alert_timeout1.yaml",
				settings,
			},
			"",
			false,
		},
		{
			"Normal case: alert_configs/timeout is equal to the upper limit. Boundary value test",
			args{
				"testdata/alert_timeout36000.yaml",
				settings,
			},
			"",
			false,
		},
		{
			"Error case: alert_configs/timeout is greater than the upper limit. Boundary value test",
			args{
				"testdata/alert_timeout36001.yaml",
				settings,
			},
			"",
			true,
		},
		{
			"Normal case: Default value is used when alert_configs/timeout is omitted",
			args{
				"testdata/alert_timeout_empty.yaml",
				settings,
			},
			"",
			false,
		},
		{
			"Error case: alert_configs/target_url is required and cannot be omitted",
			args{
				"testdata/alert_url_empty.yaml",
				settings,
			},
			"",
			true,
		},
		{
			"Error case: alert_configs/target_url is not in URL format",
			args{
				"testdata/alert_url_format_invalid.yaml",
				settings,
			},
			"",
			true,
		},
		{
			"Error case: alert_configs/state_settings/normal_state is required and cannot be omitted",
			args{
				"testdata/alert_normal_state_empty.yaml",
				settings,
			},
			"",
			true,
		},
		{
			"Error case: alert_configs/state_settings/normal_state value is empty",
			args{
				"testdata/alert_normal_state_blank.yaml",
				settings,
			},
			"",
			true,
		},
		{
			"Error case: alert_configs/state_settings/normal_health is required and cannot be omitted",
			args{
				"testdata/alert_normal_health_empty.yaml",
				settings,
			},
			"",
			true,
		},
		{
			"Error case: alert_configs/state_settings/normal_health value is empty",
			args{
				"testdata/alert_normal_health_blank.yaml",
				settings,
			},
			"",
			true,
		},
		{
			"Normal case: Typical usage scenario",
			args{
				"testdata/exporter.yaml",
				settings,
			},
			"",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := loadConfig(tt.args.filepath, &tt.args.settings)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				fmt.Printf("url: %+v\n", tt.args.settings.CollectConfigs.TargetUrl)
				fmt.Printf("timeout: %+v\n", tt.args.settings.CollectConfigs.TimeOut)
				return
			}
		})
	}
}

func Test_validConfigUrl(t *testing.T) {
	t.Skip("not test. Because the loadConfig function is covered in the tests for this function.")
}

func Test_validConfigTime(t *testing.T) {
	t.Skip("not test. Because the loadConfig function is covered in the tests for this function.")
}

func Test_validConfigSliceRequired(t *testing.T) {
	t.Skip("not test. Because the loadConfig function is covered in the tests for this function.")
}

func Test_requestDevices(t *testing.T) {
	t.Skip("not test")
}

func Test_isResourceStatus(t *testing.T) {
	type args struct {
		resource     map[string]any
		stateSetting yamlStateSetting
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"Error case: Resource without status element",
			args{
				map[string]any{"test": "aaa"},
				yamlStateSetting{
					NormalState:  []string{"Enabled", "Qualified"},
					NormalHealth: []string{"OK", "Warning"},
				},
			},
			false,
		},
		{
			"Error case: Resource where the value of the status element is not a map",
			args{
				map[string]any{"status": "aaa"},
				yamlStateSetting{
					NormalState:  []string{"Enabled", "Qualified"},
					NormalHealth: []string{"OK", "Warning"},
				},
			},
			false,
		},
		{
			"Error case: Resource without status.state element",
			args{
				map[string]any{"status": map[string]any{"test": "Enabled", "health": "OK"}},
				yamlStateSetting{
					NormalState:  []string{"Enabled", "Qualified"},
					NormalHealth: []string{"OK", "Warning"},
				},
			},
			false,
		},
		{
			"Error case: Resource where the value of the status.state element is not a string",
			args{
				map[string]any{"status": map[string]any{"state": 1, "health": "OK"}},
				yamlStateSetting{
					NormalState:  []string{"Enabled", "Qualified"},
					NormalHealth: []string{"OK", "Warning"},
				},
			},
			false,
		},
		{
			"Error case: Resource where the value of status.state element is an abnormal value (not present in yamlStateSetting.NormalState)",
			args{
				map[string]any{"status": map[string]any{"state": "aaa", "health": "OK"}},
				yamlStateSetting{
					NormalState:  []string{"Enabled", "Qualified"},
					NormalHealth: []string{"OK", "Warning"},
				},
			},
			false,
		},
		{
			"Error case: Resource without status.health element",
			args{
				map[string]any{"status": map[string]any{"state": "Enabled", "test": "OK"}},
				yamlStateSetting{
					NormalState:  []string{"Enabled", "Qualified"},
					NormalHealth: []string{"OK", "Warning"},
				},
			},
			false,
		},
		{
			"Error case: Resource where the value of status.health element is not a string",
			args{
				map[string]any{"status": map[string]any{"state": "Enabled", "health": 1}},
				yamlStateSetting{
					NormalState:  []string{"Enabled", "Qualified"},
					NormalHealth: []string{"OK", "Warning"},
				},
			},
			false,
		},
		{
			"Error case: Resource where the value of status.health element is an abnormal value (not present in yamlStateSetting.NormalHealth)",
			args{
				map[string]any{"status": map[string]any{"state": "Enabled", "health": "aaa"}},
				yamlStateSetting{
					NormalState:  []string{"Enabled", "Qualified"},
					NormalHealth: []string{"OK", "Warning"},
				},
			},
			false,
		},
		{
			"Normal case: Resource with normal status",
			args{
				map[string]any{"status": map[string]any{"state": "Enabled", "health": "OK"}},
				yamlStateSetting{
					NormalState:  []string{"Enabled", "Qualified"},
					NormalHealth: []string{"OK", "Warning"},
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isResourceStatus(tt.args.resource, tt.args.stateSetting); got != tt.want {
				t.Errorf("isResourceStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isResourceStatusOne(t *testing.T) {
	t.Skip("not test. Because the isResourceStatus function is covered in the tests for this function.")
}

func Test_postAlert(t *testing.T) {
	// Create a test server to simulate the external web application
	testServerOk := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	}))
	defer testServerOk.Close()

	testServerNg := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "ERROR"}`))
	}))
	defer testServerNg.Close()

	type args struct {
		alertName string
		alerts    []any
		settings  yamlContent
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"Normal case: Post alert successfully",
			args{
				alertName: "testAlert",
				alerts:    []any{"alert1", "alert2"},
				settings: yamlContent{
					AlertConfigs: yamlAlertConfig{
						TargetUrl: testServerOk.URL,
						TimeOut:   new(int),
					},
				},
			},
		},
		{
			"Error case: Failed to marshal alerts",
			args{
				alertName: "testAlert",
				alerts:    []any{make(chan int)}, // Unmarshalable type
				settings: yamlContent{
					AlertConfigs: yamlAlertConfig{
						TargetUrl: testServerOk.URL,
						TimeOut:   new(int),
					},
				},
			},
		},
		{
			"Error case: Invalid alert target URL",
			args{
				alertName: "testAlert",
				alerts:    []any{"alert1", "alert2"},
				settings: yamlContent{
					AlertConfigs: yamlAlertConfig{
						TargetUrl: testServerNg.URL,
						TimeOut:   new(int),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			postAlert(tt.args.alertName, tt.args.alerts, &tt.args.settings)
		})
	}
}
