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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
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
	AlertConfigs   yamlAlertConfig   `yaml:"alert_config"`
}

type yamlCollectConfig struct {
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
	TimeStanp         string           `json:"infoTimestamp"`
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

// Execute bulk information retrieval of all HW control resources and
// edit the obtained data into the format of HW information synchronization input for configuration information management
func GetDevices(c *gin.Context) {
	log.Info(c.Request.URL.Path + "[" + c.Request.Method + "] start.")

	settings := yamlContent{}

	err := loadConfig(yamlFilePath, &settings)
	if err != nil {
		log.Error(err.Error())
		c.JSON(GetStatusCode(err), ToJson(err))
		return
	}

	output := Output{}
	err = requestDevices(&settings, &output)
	if err != nil {
		log.Error(err.Error())
		c.JSON(GetStatusCode(err), ToJson(err))
		return
	}

	// If incompleteDeviceList exists, notify the alert of incompleteDeviceList
	if output.IncompleteDevices != nil {
		log.Warn(fmt.Sprintf("%s existed. Send an alert notification.", incompleteDeviceList))
		go postAlert(incompleteDeviceList, output.IncompleteDevices, &settings)
	} else {
		log.Info(fmt.Sprintf("%s not existed. Not send an alert notification.", incompleteDeviceList))
	}

	// Edit the data obtained from bulk information retrieval of all HW control resources
	// into the format of HW information synchronization input for configuration information management
	resources := make([]any, 0)
	abnormalResources := make([]any, 0)
	for _, device := range output.Devices {
		resources = append(resources, device)
		if !isResourceStatus(device, settings.AlertConfigs.StateSettings) {
			abnormalResources = append(abnormalResources, device)
		}
	}

	// If there are resources with abnormal status, notify the alert of abnormalStatusDeviceList
	if len(abnormalResources) > 0 {
		log.Warn(fmt.Sprintf("%s existed. Send an alert notification.", abnormalStatusDeviceList))
		go postAlert(abnormalStatusDeviceList, abnormalResources, &settings)
	} else {
		log.Info(fmt.Sprintf("%s not existed. Not send an alert notification.", abnormalStatusDeviceList))
	}

	log.Info(c.Request.URL.Path + "[" + c.Request.Method + "] completed successfully.")
	c.JSON(http.StatusOK, resources)
}

// Load settings from yaml file and store in struct
func loadConfig(filepath string, settings *yamlContent) error {
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
		log.Warn(fmt.Sprintf("%s was not specified in the yaml configuration file. The default value has been set.", targetName))
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
func requestDevices(settings *yamlContent, output *Output) error {
	// Since http.Client does not have a timeout set by default, set it
	httpClient := http.Client{Timeout: time.Duration(*settings.CollectConfigs.TimeOut) * time.Second}

	resp, err := httpClient.Get(settings.CollectConfigs.TargetUrl)
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
func isResourceStatus(resource map[string]any, stateSetting yamlStateSetting) bool {
	status, ok := resource["status"].(map[string]any)
	if !ok {
		log.Warn("status does not exist or the value is not a Map.")
		return false
	}

	ok = isResourceStatusOne("state", status, stateSetting.NormalState)
	if !ok {
		return false
	}

	ok = isResourceStatusOne("health", status, stateSetting.NormalHealth)
	if !ok {
		return false
	}

	return true
}

// Return true if the value of the resource's state or health element is normal, false if abnormal
func isResourceStatusOne(key string, statusMap map[string]any, normalStatusList []string) bool {
	status, ok := statusMap[key].(string)
	if !ok {
		log.Warn(fmt.Sprintf("status.%s does not exist or the value is not a String.", key))
		return false
	}

	return slices.Contains(normalStatusList, status)
}

// POST an alert to the alert notification destination
func postAlert(alertName string, alerts []any, settings *yamlContent) {
	log.Info("Starting the post.")

	// Marshal the alerts to set the result as a string in "annotations"
	annotationsJson, err := json.Marshal(alerts)
	if err != nil {
		log.Error("Failed to marshal for 'annotations'.")
		log.Error(fmt.Sprintf("Unmarshalable: %#v", alerts), false)
		log.Error(err.Error(), false)
		return
	}

	alertBody := NewAlertContentList(alertName, "critical", string(annotationsJson))

	alertJsonBody, err := json.Marshal(alertBody)
	if err != nil {
		log.Error("Failed to marshal.")
		log.Error(fmt.Sprintf("Unmarshalable: %#v", alertBody), false)
		log.Error(err.Error(), false)
		return
	}

	httpClient := http.Client{Timeout: time.Duration(*settings.AlertConfigs.TimeOut) * time.Second}
	_, err = httpClient.Post(settings.AlertConfigs.TargetUrl, "application/json", bytes.NewBuffer(alertJsonBody))
	if err != nil {
		log.Error("post has failed.")
		log.Error(string(alertJsonBody), false)
		log.Error(err.Error(), false)
		return
	}

	log.Info("post has been completed.")
	log.Info(string(alertJsonBody))

	return
}
