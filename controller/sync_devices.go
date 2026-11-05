// Copyright (C) 2025-2026 NEC Corporation.
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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	loggercommon "github.com/project-cdim/cdim-go-logger/common"
	yaml "gopkg.in/yaml.v2"
)

const (
	defaultTimeout int    = 600
	maxTimeout     int    = 36000
	minTimeout     int    = 1
	yamlFilePath   string = "configs/exporter.yaml"
)

const (
	incompleteDeviceList     string = "incompleteDeviceList"
	abnormalStatusDeviceList string = "abnormalStatusDeviceList"
)

type yamlContent struct {
	CollectConfigs yamlCollectConfig `yaml:"collect_configs"`
	ForwardConfigs yamlForwardConfig `yaml:"forward_configs"`
	AlertConfigs   yamlAlertConfig   `yaml:"alert_config"`
}

type yamlCollectConfig struct {
	TargetUrl string `yaml:"target_url"`
	TimeOut   *int   `yaml:"timeout"`
}

type yamlForwardConfig struct {
	TargetUrl string `yaml:"target_url"`
	TimeOut   *int   `yaml:"timeout"`
}

type yamlAlertConfig struct {
	TargetUrl     string           `yaml:"target_url"`
	TimeOut       *int             `yaml:"timeout"`
	StateSettings yamlStateSetting `yaml:"state_settings"`
}

type yamlStateSetting struct {
	NormalState  []string `yaml:"normal_state"`
	NormalHealth []string `yaml:"normal_health"`
}

type Output struct {
	Devices           []map[string]any `json:"deviceList"`
	IncompleteDevices []any            `json:"incompleteDeviceList"`
	TimeStamp         string           `json:"infoTimestamp"`
}

type alertContent struct {
	Status      string           `json:"status"`
	Labels      alertLabels      `json:"labels"`
	Annotations alertAnnotations `json:"annotations"`
}

type alertLabels struct {
	Alertname string `json:"alertname"`
	Instance  string `json:"instance"`
	Job       string `json:"job"`
	Severity  string `json:"severity"`
}

type alertAnnotations struct {
	Description string `json:"description"`
}

type alertContentList []alertContent

// create a new alertContentList
func NewAlertContentList(alertName, severity, description string) alertContentList {
	return alertContentList{
		{
			Status: "firing",
			Labels: alertLabels{
				Alertname: alertName,
				Instance:  "configuration-exporter",
				Job:       "configuration-exporter",
				Severity:  severity,
			},
			Annotations: alertAnnotations{
				Description: description,
			},
		},
	}
}

// SyncDevices handles the synchronization of device information.
// Execute bulk information retrieval of all HW control resources and
// edit the obtained data into the format of HW information synchronization input for configuration-manager.
// Forward the edited data to configuration-manager.
//
// Parameters:
//   - c: *gin.Context. The gin context for the HTTP request and response.
//
// Response Codes:
//   - 202 Accepted: Returned when the synchronization process is successfully initiated.
//   - 500 Internal Server Error: Returned when an error occurs during any step of the process.
func SyncDevices(c *gin.Context) {
	ctx := c.Request.Context()
	log.InfoWithContext(ctx, c.Request.URL.Path+"["+c.Request.Method+"] start.")

	settings := yamlContent{}

	err := loadConfig(ctx, yamlFilePath, &settings)
	if err != nil {
		log.ErrorWithContext(ctx, err.Error())
		c.JSON(GetStatusCode(err), ToJson(err))
		return
	}

	output := Output{}
	err = requestDevices(ctx, &settings, &output)
	if err != nil {
		log.ErrorWithContext(ctx, err.Error())
		c.JSON(GetStatusCode(err), ToJson(err))
		return
	}

	// If incompleteDeviceList exists, notify the alert of incompleteDeviceList
	if output.IncompleteDevices != nil {
		log.WarnWithContext(ctx, fmt.Sprintf("%s existed. Send an alert notification.", incompleteDeviceList))
		go postAlert(createAsyncContext(ctx), incompleteDeviceList, output.IncompleteDevices, &settings)
	} else {
		log.InfoWithContext(ctx, fmt.Sprintf("%s not existed. Not send an alert notification.", incompleteDeviceList))
	}

	// Edit the data obtained from bulk information retrieval of all HW control resources
	// into the format of HW information synchronization input for configuration information management
	resources := make([]any, 0)
	abnormalResources := make([]any, 0)
	for _, device := range output.Devices {
		resources = append(resources, device)
		if !isResourceStatus(ctx, device, settings.AlertConfigs.StateSettings) {
			abnormalResources = append(abnormalResources, device)
		}
	}

	// If there are resources with abnormal status, notify the alert of abnormalStatusDeviceList
	if len(abnormalResources) > 0 {
		log.WarnWithContext(ctx, fmt.Sprintf("%s existed. Send an alert notification.", abnormalStatusDeviceList))
		go postAlert(createAsyncContext(ctx), abnormalStatusDeviceList, abnormalResources, &settings)
	} else {
		log.InfoWithContext(ctx, fmt.Sprintf("%s not existed. Not send an alert notification.", abnormalStatusDeviceList))
	}

	// Forward the edited data to configuration-manager.
	go forwardData(createAsyncContext(ctx), &settings.ForwardConfigs, resources)

	log.InfoWithContext(ctx, c.Request.URL.Path+"["+c.Request.Method+"] completed successfully.")
	c.JSON(http.StatusAccepted, nil)
}

// Load settings from yaml file and store in struct
//
// Parameters:
//   - ctx: context.Context. The context for logging.
//   - filepath: string. The path to the YAML configuration file.
//   - settings: *yamlContent. The struct to be populated with the configuration.
//
// Returns:
//   - error: Returns an error if file reading, unmarshaling, or validation fails, nil otherwise.
func loadConfig(ctx context.Context, filepath string, settings *yamlContent) error {
	buf, err := os.ReadFile(filepath)
	if err != nil {
		return ExpErrorNew(http.StatusInternalServerError, "0001", "Failed to read file.")
	}

	err = yaml.Unmarshal(buf, &settings)
	if err != nil {
		return ExpErrorNew(http.StatusInternalServerError, "0002", "Failed to unmarshal yaml.")
	}

	// Check the required and format of URL (collect_configs/target_url)
	err = validConfigUrl("collect_configs/target_url", settings.CollectConfigs.TargetUrl)
	if err != nil {
		return err
	}

	// Check the required and format of URL (forward_configs/target_url)
	err = validConfigUrl("forward_configs/target_url", settings.ForwardConfigs.TargetUrl)
	if err != nil {
		return err
	}

	// Check the required and format of URL (alert_config/target_url)
	err = validConfigUrl("alert_config/target_url", settings.AlertConfigs.TargetUrl)
	if err != nil {
		return err
	}

	// Check the range of Timeout (collect_configs/timeout)
	settings.CollectConfigs.TimeOut, err = validConfigTime("collect_configs/timeout", settings.CollectConfigs.TimeOut)
	if err != nil {
		return err
	}

	// Check the range of Timeout (forward_configs/timeout)
	settings.ForwardConfigs.TimeOut, err = validConfigTime("forward_configs/timeout", settings.ForwardConfigs.TimeOut)
	if err != nil {
		return err
	}

	// Check the range of Timeout (alert_config/timeout)
	settings.AlertConfigs.TimeOut, err = validConfigTime("alert_config/timeout", settings.AlertConfigs.TimeOut)
	if err != nil {
		return err
	}

	// Check for nil or empty slice (alert_config/state_settings/normal_state)
	err = validConfigSliceRequired("alert_config/state_settings/normal_state", settings.AlertConfigs.StateSettings.NormalState)
	if err != nil {
		return err
	}

	// Check for nil or empty slice (alert_config/state_settings/normal_health)
	err = validConfigSliceRequired("alert_config/state_settings/normal_health", settings.AlertConfigs.StateSettings.NormalHealth)
	if err != nil {
		return err
	}

	return nil
}

// Check for required and format of URL
func validConfigUrl(targetName string, targetValue string) error {
	if targetValue == "" {
		return ExpErrorNew(http.StatusInternalServerError, "0010", fmt.Sprintf("%s setting is required.", targetName))
	}
	// url.Parse allows relative paths. url.ParseRequestURI only allows absolute URIs or absolute paths.
	// Since we want an absolute URI here, we parse with url.ParseRequestURI.
	_, err := url.ParseRequestURI(targetValue)
	if err != nil {
		return ExpErrorNew(http.StatusInternalServerError, "0011", fmt.Sprintf("%s Format of the url is invalid.", targetName))
	}

	return nil
}

// Check the range of Timeout
func validConfigTime(targetName string, targetValue *int) (*int, error) {
	if targetValue == nil {
		log.Warn(fmt.Sprintf("%s was not specified in the yaml configuration file. The default value has been set.", targetName), false)
		defTimeout := defaultTimeout
		return &defTimeout, nil
	}
	if *targetValue < minTimeout || *targetValue > maxTimeout {
		return nil, ExpErrorNew(http.StatusInternalServerError, "0012", fmt.Sprintf("%s value is out of range.", targetName))
	}

	return targetValue, nil
}

// Check for nil or empty slice
func validConfigSliceRequired(targetName string, targetValue []string) error {
	if targetValue == nil {
		return ExpErrorNew(http.StatusInternalServerError, "0013", fmt.Sprintf("%s value is nil.", targetName))
	}

	if len(targetValue) == 0 {
		return ExpErrorNew(http.StatusInternalServerError, "0014", fmt.Sprintf("%s value is blank.", targetName))
	}

	return nil
}

// Request bulk information retrieval of all resources for HW control
//
// Parameters:
//   - ctx: context.Context. The context for the HTTP request and logging.
//   - settings: *yamlContent. The configuration containing target URL and timeout.
//   - output: *Output. The struct to be populated with the response.
//
// Returns:
//   - error: Returns an error if the HTTP request fails or the response cannot be unmarshaled, nil otherwise.
func requestDevices(ctx context.Context, settings *yamlContent, output *Output) error {
	// Since http.Client does not have a timeout set by default, set it
	httpClient := http.Client{Timeout: time.Duration(*settings.CollectConfigs.TimeOut) * time.Second}

	// Create HTTP request with context
	req, newReqErr := http.NewRequestWithContext(ctx, http.MethodGet, settings.CollectConfigs.TargetUrl, nil)
	if newReqErr != nil {
		return ExpErrorNew(http.StatusInternalServerError, "0006", "Failed to create request.")
	}

	// Add traceparent header if present in context
	traceparent := loggercommon.TraceparentFromContext(ctx)
	if traceparent != "" {
		req.Header.Set("traceparent", traceparent)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return ExpErrorNew(http.StatusInternalServerError, "0006", "Get request failure.")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ExpErrorNew(http.StatusInternalServerError, "0007", "Collect target failure.")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ExpErrorNew(http.StatusInternalServerError, "0008", "Failed to read response.")
	}

	err = json.Unmarshal(body, &output)
	if err != nil {
		return ExpErrorNew(http.StatusInternalServerError, "0009", "Failed to unmarshal response.")
	}

	return nil
}

// Return true if the resource status is normal, false if abnormal
//
// Parameters:
//   - ctx: context.Context. The context for logging.
//   - resource: map[string]any. The resource data to check.
//   - stateSetting: yamlStateSetting. The configuration for normal status values.
//
// Returns:
//   - bool: Returns true if the resource status is normal, false if abnormal.
func isResourceStatus(ctx context.Context, resource map[string]any, stateSetting yamlStateSetting) bool {
	status, ok := resource["status"].(map[string]any)
	if !ok {
		log.WarnWithContext(ctx, "status does not exist or the value is not a Map.")
		return false
	}

	ok = isResourceStatusOne(ctx, "state", status, stateSetting.NormalState)
	if !ok {
		return false
	}

	ok = isResourceStatusOne(ctx, "health", status, stateSetting.NormalHealth)
	if !ok {
		return false
	}

	return true
}

// Return true if the value of the resource's state or health element is normal, false if abnormal
//
// Parameters:
//   - ctx: context.Context. The context for logging.
//   - key: string. The key to check in the status map ("state" or "health").
//   - statusMap: map[string]any. The status map containing state and health values.
//   - normalStatusList: []string. The list of normal status values.
//
// Returns:
//   - bool: Returns true if the value is normal, false if abnormal.
func isResourceStatusOne(ctx context.Context, key string, statusMap map[string]any, normalStatusList []string) bool {
	status, ok := statusMap[key].(string)
	if !ok {
		log.WarnWithContext(ctx, fmt.Sprintf("status.%s does not exist or the value is not a String.", key))
		return false
	}

	return slices.Contains(normalStatusList, status)
}

// createAsyncContext creates a new context for async processing, independent from the request context.
// This function extracts metadata (e.g., traceparent) from the request context and sets it in a new background context.
//
// Parameters:
//   - ctx: context.Context. The request context containing metadata to preserve.
//
// Returns:
//   - context.Context: A new background context with preserved metadata.
func createAsyncContext(ctx context.Context) context.Context {
	asyncCtx := context.Background()
	traceparent := loggercommon.TraceparentFromContext(ctx)
	if traceparent != "" {
		asyncCtx = context.WithValue(asyncCtx, loggercommon.TraceparentKey, traceparent)
	}
	return asyncCtx
}

// POST an alert to the alert notification destination
//
// Parameters:
//   - ctx: context.Context. The context for the HTTP request and logging (independent from request context).
//   - alertName: string. The name of the alert.
//   - alerts: []any. The alert data to send.
//   - settings: *yamlContent. The configuration containing alert target URL and timeout.
func postAlert(ctx context.Context, alertName string, alerts []any, settings *yamlContent) {
	// Create a new context with timeout for async processing
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(*settings.AlertConfigs.TimeOut)*time.Second)
	defer cancel()

	log.InfoWithContext(timeoutCtx, "Starting the post.")

	// Marshal the alerts to set the result as a string in "annotations"
	annotationsJson, err := json.Marshal(alerts)
	if err != nil {
		log.ErrorWithContext(timeoutCtx, "Failed to marshal for 'annotations'.")
		log.ErrorWithContext(timeoutCtx, fmt.Sprintf("Unmarshalable: %#v", alerts))
		log.ErrorWithContext(timeoutCtx, err.Error())
		return
	}

	alertBody := NewAlertContentList(alertName, "critical", string(annotationsJson))

	alertJsonBody, err := json.Marshal(alertBody)
	if err != nil {
		log.ErrorWithContext(timeoutCtx, "Failed to marshal.")
		log.ErrorWithContext(timeoutCtx, fmt.Sprintf("Unmarshalable: %#v", alertBody))
		log.ErrorWithContext(timeoutCtx, err.Error())
		return
	}

	// Create HTTP request with context
	req, newReqErr := http.NewRequestWithContext(timeoutCtx, http.MethodPost, settings.AlertConfigs.TargetUrl, bytes.NewBuffer(alertJsonBody))
	if newReqErr != nil {
		log.ErrorWithContext(timeoutCtx, "Failed to create request.")
		log.ErrorWithContext(timeoutCtx, newReqErr.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")

	// Add traceparent header if present in context
	traceparent := loggercommon.TraceparentFromContext(timeoutCtx)
	if traceparent != "" {
		req.Header.Set("traceparent", traceparent)
	}

	httpClient := http.Client{Timeout: time.Duration(*settings.AlertConfigs.TimeOut) * time.Second}
	_, err = httpClient.Do(req)
	if err != nil {
		log.ErrorWithContext(timeoutCtx, "post has failed.")
		log.ErrorWithContext(timeoutCtx, string(alertJsonBody))
		log.ErrorWithContext(timeoutCtx, err.Error())
		return
	}

	log.InfoWithContext(timeoutCtx, "post has been completed.")
	log.InfoWithContext(timeoutCtx, string(alertJsonBody))

	return
}

// forwardData sends the provided data to a specified target URL using an HTTP POST request.
// This function is called asynchronously, so it does not return an error.
//
// Parameters:
//   - ctx: context.Context. The context for the HTTP request and logging (independent from request context).
//   - settings: A pointer to a yamlForwardConfig struct.
//   - data: The resource data to be sent, which can be of any type.
func forwardData(ctx context.Context, settings *yamlForwardConfig, data any) {
	// Create a new context with timeout for async processing
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(*settings.TimeOut)*time.Second)
	defer cancel()

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.ErrorWithContext(timeoutCtx, err.Error())
		return
	}

	// Create HTTP request with context
	req, newReqErr := http.NewRequestWithContext(timeoutCtx, http.MethodPost, settings.TargetUrl, bytes.NewBuffer(jsonData))
	if newReqErr != nil {
		log.ErrorWithContext(timeoutCtx, newReqErr.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")

	// Add traceparent header if present in context
	traceparent := loggercommon.TraceparentFromContext(timeoutCtx)
	if traceparent != "" {
		req.Header.Set("traceparent", traceparent)
	}

	httpClient := http.Client{Timeout: time.Duration(*settings.TimeOut) * time.Second}
	res, err := httpClient.Do(req)
	if err != nil {
		log.ErrorWithContext(timeoutCtx, err.Error())
		return
	}

	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		statusCode := strconv.Itoa(res.StatusCode)
		log.ErrorWithContext(timeoutCtx, "status code = "+statusCode)
		return
	}

	return
}
