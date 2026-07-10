// Package apphub is an HTTP client for the DeviceLink/AppHub device
// management REST API (see /opt/AppHub-v2.3.0.md), used to back the
// EIM-SKILL command-line tools.
package apphub

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client talks to a DeviceLink/AppHub server over HTTP.
type Client struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
	token      string
}

// NewClient creates a Client for the AppHub server at baseURL (e.g.
// "http://127.0.0.1:30010"). Login happens lazily on first request.
func NewClient(baseURL, username, password string) *Client {
	return &Client{
		baseURL:  baseURL,
		username: username,
		password: password,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// APIError is returned when the server responds with a non-2xx status.
type APIError struct {
	StatusCode int
	Status     string
	Message    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("apphub: %s (http %d)", e.Message, e.StatusCode)
	}
	return fmt.Sprintf("apphub: request failed (http %d)", e.StatusCode)
}

func (c *Client) login() error {
	reqBody, err := json.Marshal(loginRequest{Username: c.username, Password: c.password})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/login", bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("apphub: login request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{StatusCode: resp.StatusCode, Message: string(data)}
	}

	var loginResp loginResponse
	if err := json.Unmarshal(data, &loginResp); err != nil {
		return fmt.Errorf("apphub: decode login response: %w", err)
	}
	if loginResp.Token == "" {
		return fmt.Errorf("apphub: login succeeded but no token was returned")
	}

	c.token = loginResp.Token
	return nil
}

// Token returns a valid accesstoken for this server, logging in first if
// there is no cached token yet. It exists so other clients (namely the repo
// server client, which accepts the same AppHub token) can reuse this
// client's session instead of logging in separately.
func (c *Client) Token() (string, error) {
	if c.token == "" {
		if err := c.login(); err != nil {
			return "", err
		}
	}
	return c.token, nil
}

// Relogin discards any cached token and logs in again.
func (c *Client) Relogin() error {
	return c.login()
}

// do issues an HTTP request against path (e.g. "/api/devicemgr"), decoding
// the JSON response body into out (if non-nil). It logs in first if there
// is no cached token yet, and retries once after a fresh login on 401/403.
func (c *Client) do(method, path string, body interface{}, out interface{}) error {
	if c.token == "" {
		if err := c.login(); err != nil {
			return err
		}
	}

	doOnce := func() (*http.Response, []byte, error) {
		var reader io.Reader
		if body != nil {
			data, err := json.Marshal(body)
			if err != nil {
				return nil, nil, err
			}
			reader = bytes.NewReader(data)
		}

		req, err := http.NewRequest(method, c.baseURL+path, reader)
		if err != nil {
			return nil, nil, err
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("accesstoken", c.token)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, nil, err
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, nil, err
		}
		return resp, data, nil
	}

	resp, data, err := doOnce()
	if err != nil {
		return fmt.Errorf("apphub: request %s %s failed: %w", method, path, err)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		if err := c.login(); err != nil {
			return err
		}
		resp, data, err = doOnce()
		if err != nil {
			return fmt.Errorf("apphub: request %s %s failed: %w", method, path, err)
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{StatusCode: resp.StatusCode, Message: string(data)}
		var sr statusResponse
		if json.Unmarshal(data, &sr) == nil && sr.ErrMsg != "" {
			apiErr.Status = sr.Status
			apiErr.Message = sr.ErrMsg
		}
		return apiErr
	}

	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("apphub: decode response for %s %s: %w", method, path, err)
		}
	}
	return nil
}

// downloadFile issues a GET request against path and returns the raw
// response body, for endpoints that return a binary file rather than the
// usual JSON envelope. It logs in first if there is no cached token yet,
// and retries once after a fresh login on 401/403.
func (c *Client) downloadFile(path string) ([]byte, error) {
	if c.token == "" {
		if err := c.login(); err != nil {
			return nil, err
		}
	}

	doOnce := func() (*http.Response, []byte, error) {
		req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
		if err != nil {
			return nil, nil, err
		}
		req.Header.Set("accesstoken", c.token)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, nil, err
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, nil, err
		}
		return resp, data, nil
	}

	resp, data, err := doOnce()
	if err != nil {
		return nil, fmt.Errorf("apphub: download %s failed: %w", path, err)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		if err := c.login(); err != nil {
			return nil, err
		}
		resp, data, err = doOnce()
		if err != nil {
			return nil, fmt.Errorf("apphub: download %s failed: %w", path, err)
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{StatusCode: resp.StatusCode, Message: string(data)}
	}
	return data, nil
}

// ListDevices returns every device registered on the server.
func (c *Client) ListDevices() ([]Device, error) {
	var devices []Device
	if err := c.do(http.MethodGet, "/api/devicemgr", nil, &devices); err != nil {
		return nil, err
	}
	return devices, nil
}

// ListDevicesByGroup returns the devices that belong to the named device group.
func (c *Client) ListDevicesByGroup(groupName string) ([]Device, error) {
	path := "/api/devicemgr/bygroupname?name=" + url.QueryEscape(groupName)
	var devices []Device
	if err := c.do(http.MethodGet, path, nil, &devices); err != nil {
		return nil, err
	}
	return devices, nil
}

// ListDevicesByCondition returns devices filtered dynamically by online
// status and/or group. online is optional: pass nil for both online and
// offline devices, a pointer to true for online-only, or a pointer to
// false for offline-only. groupName, os, and labelName are optional
// exact-match filters; pass "" to leave any of them unset.
func (c *Client) ListDevicesByCondition(online *bool, groupName, os, labelName string) ([]Device, error) {
	q := url.Values{}
	if online != nil {
		q.Set("status", strconv.FormatBool(*online))
	}
	if groupName != "" {
		q.Set("groupname", groupName)
	}
	if os != "" {
		q.Set("os", os)
	}
	if labelName != "" {
		q.Set("labelName", labelName)
	}

	var resp devicesByConditionResponse
	if err := c.do(http.MethodGet, "/api/devicemgr/allDeviceByCondition?"+q.Encode(), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// ListAlarmDevices returns devices currently in an alarm state (software or
// hardware status not "normal" - deviceStatus is the server's precomputed
// combined status: 0=normal, 1=error, 2=warning), optionally scoped to one
// device group.
func (c *Client) ListAlarmDevices(groupName string) ([]Device, error) {
	devices, err := c.ListDevicesByCondition(nil, groupName, "", "")
	if err != nil {
		return nil, err
	}

	alarms := make([]Device, 0)
	for _, d := range devices {
		if d.DeviceStatus != 0 {
			alarms = append(alarms, d)
		}
	}
	return alarms, nil
}

// GetDevice returns the details of a single device.
func (c *Client) GetDevice(agentID string) (*Device, error) {
	var device Device
	path := "/api/devicemgr/" + url.PathEscape(agentID)
	if err := c.do(http.MethodGet, path, nil, &device); err != nil {
		return nil, err
	}
	return &device, nil
}

// GetDeviceDetails returns a device's detailed system info (cpu, memory,
// storage, network interfaces, etc), as reported by its agent. This is a
// lighter-weight alternative to GetDevice for when only that blob is
// needed - not documented in AppHub-v2.3.0.md, see ClientServlet's
// get_1_details.
func (c *Client) GetDeviceDetails(deviceID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("deviceId", deviceID)

	var resp struct {
		Details string `json:"details"`
	}
	if err := c.do(http.MethodGet, "/api/devicemgr/details?"+q.Encode(), nil, &resp); err != nil {
		return nil, err
	}
	if resp.Details == "" {
		return nil, nil
	}
	return json.RawMessage(resp.Details), nil
}

// DeleteDevice removes a device from the server.
func (c *Client) DeleteDevice(agentID string) error {
	path := "/api/devicemgr/" + url.PathEscape(agentID)
	var sr statusResponse
	return c.do(http.MethodDelete, path, nil, &sr)
}

// MonitorData returns CPU or memory monitoring samples for a device.
// dataType must be "cpu" or "memory". beginTs/endTs are Unix milliseconds;
// pass 0 to omit either bound.
func (c *Client) MonitorData(agentID, dataType string, beginTs, endTs int64) ([]MonitorDataPoint, error) {
	if dataType != "cpu" && dataType != "memory" {
		return nil, fmt.Errorf("apphub: dataType must be \"cpu\" or \"memory\", got %q", dataType)
	}

	q := url.Values{}
	q.Set("type", dataType)
	q.Set("agentId", agentID)
	if beginTs > 0 {
		q.Set("beginTs", fmt.Sprintf("%d", beginTs))
	}
	if endTs > 0 {
		q.Set("endTs", fmt.Sprintf("%d", endTs))
	}

	var points []MonitorDataPoint
	if err := c.do(http.MethodGet, "/api/monitorData/byType?"+q.Encode(), nil, &points); err != nil {
		return nil, err
	}
	return points, nil
}

// CreateGroup adds a new device group. groupType is typically one of
// "linux", "windows", "android", "mix" (the server's built-in defaults),
// though this isn't a strictly enforced enum. username is recorded as the
// group's creator.
func (c *Client) CreateGroup(groupName, desc, groupType, username string) error {
	body := createGroupRequest{GroupName: groupName, Desc: desc, Type: groupType, Username: username}
	var sr statusResponse
	return c.do(http.MethodPost, "/api/groupmgr", body, &sr)
}

// ListGroups returns every device group defined on the server.
func (c *Client) ListGroups() ([]DeviceGroup, error) {
	var groups []DeviceGroup
	if err := c.do(http.MethodGet, "/api/groupmgr/all", nil, &groups); err != nil {
		return nil, err
	}
	return groups, nil
}

// GroupsOfDevice returns the device groups that the given device endpoint
// currently belongs to.
func (c *Client) GroupsOfDevice(endpoint string) ([]DeviceGroup, error) {
	path := "/api/groupmgr/getGroupByDevId?endpoint=" + url.QueryEscape(endpoint)
	var groups []DeviceGroup
	if err := c.do(http.MethodGet, path, nil, &groups); err != nil {
		return nil, err
	}
	return groups, nil
}

// SetDeviceGroups replaces the full set of device groups that endpoint
// belongs to with groupNames.
func (c *Client) SetDeviceGroups(endpoint string, groupNames []string) error {
	body := setGroupsRequest{Endpoint: endpoint, GroupNames: groupNames}
	var sr statusResponse
	return c.do(http.MethodPut, "/api/devicemgr/bygroupname", body, &sr)
}

// AddDeviceToGroup adds endpoint to groupName without removing it from any
// device group it already belongs to. Returns the resulting full group list.
func (c *Client) AddDeviceToGroup(endpoint, groupName string) ([]string, error) {
	current, err := c.GroupsOfDevice(endpoint)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(current)+1)
	found := false
	for _, g := range current {
		names = append(names, g.Name)
		if g.Name == groupName {
			found = true
		}
	}
	if !found {
		names = append(names, groupName)
	}

	if err := c.SetDeviceGroups(endpoint, names); err != nil {
		return nil, err
	}
	return names, nil
}

// linuxPkgResourcePath is the LwM2M devicecontrol object/resource that
// reports (GET) and drives (POST) Linux software package management. See
// AppHub-v2.3.0.md "Linux-获取单个设备的所有安装包信息"/"Linux-单个设备更新安装包"
// and DeviceLink's lwm2mMap.js linuxPkgControl (getPkgInfo/install/upgrade).
const (
	linuxPkgInfoResourceID = 28300
	linuxPkgDeployResource = "60001/0/28301"

	linuxSingleDeployType = "debsg" // single-device install/update/uninstall
	linuxGroupDeployType  = "debbt" // group/batch install/update/uninstall
)

// GetInstalledLinuxPackages returns the software packages currently
// installed on a Linux device.
func (c *Client) GetInstalledLinuxPackages(agentID string) ([]InstalledLinuxPackage, error) {
	path := fmt.Sprintf("/api/devicemgr/%s/60001/0/%d", url.PathEscape(agentID), linuxPkgInfoResourceID)
	var cr contentResponse
	if err := c.do(http.MethodGet, path, nil, &cr); err != nil {
		return nil, err
	}
	if cr.Content.Value == "" {
		return []InstalledLinuxPackage{}, nil
	}

	var pkgs []InstalledLinuxPackage
	if err := json.Unmarshal([]byte(cr.Content.Value), &pkgs); err != nil {
		return nil, fmt.Errorf("apphub: decode installed linux packages: %w", err)
	}
	return pkgs, nil
}

// DeployLinuxPackage installs or updates a software package on a single
// Linux device (agentID). taskname should be unique per deployment.
// downloadPath is the package's full download path, typically
// "<repo package address>/<repo package filename>".
func (c *Client) DeployLinuxPackage(agentID, taskname, productName, version, downloadPath string) error {
	path := fmt.Sprintf("/api/devicemgr/%s/%s", url.PathEscape(agentID), linuxPkgDeployResource)
	body := deployLinuxPackageRequest{
		Taskname:     taskname,
		ProductName:  productName,
		Version:      version,
		DownloadPath: downloadPath,
		Type:         linuxSingleDeployType,
	}
	var sr statusResponse
	return c.do(http.MethodPost, path, body, &sr)
}

// DeployLinuxPackageToGroup installs or updates a software package on every
// Linux device in groupName (and/or matching labelName, which may be empty).
func (c *Client) DeployLinuxPackageToGroup(groupName, taskname, productName, version, downloadPath, labelName string) error {
	path := "/api/devicemgr/batchdeploy/" + linuxPkgDeployResource
	body := deployLinuxPackageGroupRequest{
		GroupName:    groupName,
		Taskname:     taskname,
		ProductName:  productName,
		Version:      version,
		DownloadPath: downloadPath,
		Type:         linuxGroupDeployType,
		LabelName:    labelName,
	}
	var sr statusResponse
	return c.do(http.MethodPost, path, body, &sr)
}

// devicecontrol object/resource path used for reboot, shutdown, and (best
// effort, undocumented in AppHub-v2.3.0.md) volume/backlight control. See
// DeviceLink's PluginsServlet#put_4_writevalue and the "send command"/
// "reboot" endpoints documented under /api/plugins/{agentId}/40008/0/27601.
const devicecontrolResourceID = 27601

func devicecontrolPath(agentID string) string {
	return fmt.Sprintf("/api/plugins/%s/40008/0/27601", agentID)
}

// execAction sends a devicecontrol action to the given device endpoint.
func (c *Client) execAction(agentID string, actions map[string]interface{}) error {
	actionsBytes, err := json.Marshal(actions)
	if err != nil {
		return err
	}

	valueBytes, err := json.Marshal(actionValue{
		PluginName: "devicecontrol",
		Actions:    string(actionsBytes),
	})
	if err != nil {
		return err
	}

	body := execActionRequest{ID: devicecontrolResourceID, Value: string(valueBytes)}
	var sr statusResponse
	return c.do(http.MethodPut, devicecontrolPath(agentID), body, &sr)
}

// Reboot immediately reboots the device identified by agentID (endpoint).
func (c *Client) Reboot(agentID string) error {
	return c.execAction(agentID, map[string]interface{}{"reboot": true})
}

// Shutdown immediately shuts down the device identified by agentID (endpoint).
func (c *Client) Shutdown(agentID string) error {
	return c.execAction(agentID, map[string]interface{}{"shutdown": true})
}

// SetVolume sets the device's system audio volume (0-100).
//
// NOTE: unlike Reboot/Shutdown, this action is not documented in
// AppHub-v2.3.0.md. It is inferred from the legacy "audio" action handled
// by DeviceLink's PluginsServlet and sent over the same devicecontrol
// resource. Verify against your device firmware before relying on it.
func (c *Client) SetVolume(agentID string, volume int) error {
	return c.execAction(agentID, map[string]interface{}{"audio": volume})
}

// SetBacklight sets the device's screen backlight/brightness (0-100).
//
// NOTE: unlike Reboot/Shutdown, this action is not documented in
// AppHub-v2.3.0.md. It is inferred from the legacy "brightness" action
// handled by DeviceLink's PluginsServlet and sent over the same
// devicecontrol resource. Verify against your device firmware before
// relying on it.
func (c *Client) SetBacklight(agentID string, brightness int) error {
	return c.execAction(agentID, map[string]interface{}{"brightness": brightness})
}

// getPluginAction issues a devicecontrol "read" action against resource
// 40008/0/27600 (the GET counterpart of execAction's 27601), returning the
// raw content.value string for the caller to decode. See DeviceLink's
// restfulapi/pluginsApi.js getDeviceStatus and AppHub-v2.3.0.md "GET 获取USB状态".
func (c *Client) getPluginAction(agentID string, actionsJSON string) (string, error) {
	q := url.Values{}
	q.Set("pluginname", "devicecontrol")
	q.Set("actions", actionsJSON)
	path := fmt.Sprintf("/api/plugins/%s/40008/0/27600?%s", url.PathEscape(agentID), q.Encode())

	var cr contentResponse
	if err := c.do(http.MethodGet, path, nil, &cr); err != nil {
		return "", err
	}
	return cr.Content.Value, nil
}

// GetThresholds returns the device's current monitoring threshold
// configuration for CPU/memory/storage/network/battery metrics.
func (c *Client) GetThresholds(agentID string) ([]ThresholdSetting, error) {
	raw, err := c.getPluginAction(agentID, `{"get_threshold":true}`)
	if err != nil {
		return nil, err
	}

	settings := make([]ThresholdSetting, 0, len(thresholdMetrics))
	if raw == "" {
		return settings, nil
	}

	var m map[string]rawThresholdSetting
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, fmt.Errorf("apphub: decode thresholds: %w", err)
	}

	for _, metric := range thresholdMetrics {
		v, ok := m[metric]
		if !ok {
			continue
		}
		warn, _ := v.Warn.Float64()
		errVal, _ := v.Error.Float64()
		settings = append(settings, ThresholdSetting{
			Metric:  metric,
			Enabled: v.Status == "true",
			Warn:    warn,
			Error:   errVal,
		})
	}
	return settings, nil
}

// SetThreshold enables/disables alerting for metric and sets its warn/error
// levels. metric must be one of: cpu_temp, cpu_usage, memory_usage,
// storage_usage, battery_available, network_upload_speed,
// network_download_speed.
func (c *Client) SetThreshold(agentID, metric string, enabled bool, warn, errorLevel float64) error {
	if !settableThresholdMetrics[metric] {
		return fmt.Errorf("apphub: unsupported threshold metric %q", metric)
	}
	entry := map[string]interface{}{
		metric:  enabled,
		"warn":  warn,
		"error": errorLevel,
	}
	return c.execAction(agentID, map[string]interface{}{
		"set_threshold": []map[string]interface{}{entry},
	})
}

// SetMonitorEnabled turns the device's overall hardware monitoring
// (CPU/memory/storage/network/battery threshold alerting) on or off.
func (c *Client) SetMonitorEnabled(agentID string, enabled bool) error {
	return c.execAction(agentID, map[string]interface{}{"set_isMonitor": enabled})
}

// ListProcesses returns the running processes/threads on a Windows or Linux
// device, including whether each is currently being monitored.
func (c *Client) ListProcesses(agentID string) ([]ProcessInfo, error) {
	raw, err := c.getPluginAction(agentID, `{"get_software_info":true}`)
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return []ProcessInfo{}, nil
	}

	var procs []ProcessInfo
	if err := json.Unmarshal([]byte(raw), &procs); err != nil {
		return nil, fmt.Errorf("apphub: decode process list: %w", err)
	}
	return procs, nil
}

// SetProcessMonitorEnabled turns application/process monitoring on or off.
func (c *Client) SetProcessMonitorEnabled(agentID string, enabled bool) error {
	return c.execAction(agentID, map[string]interface{}{"set_isProcessMonitor": enabled})
}

// GetDockerInfo returns the Docker compose/swarm/image groups deployed on a
// device, including each container's running status.
func (c *Client) GetDockerInfo(agentID string) ([]DockerGroup, error) {
	path := fmt.Sprintf("/api/devicemgr/%s/60001/0/27300", url.PathEscape(agentID))
	var cr contentResponse
	if err := c.do(http.MethodGet, path, nil, &cr); err != nil {
		return nil, err
	}
	if cr.Content.Value == "" {
		return []DockerGroup{}, nil
	}

	var resp dockerInfoResponse
	if err := json.Unmarshal([]byte(cr.Content.Value), &resp); err != nil {
		return nil, fmt.Errorf("apphub: decode docker info: %w", err)
	}
	return resp.Data, nil
}

// SetDockerMonitorEnabled turns Docker container monitoring on or off.
func (c *Client) SetDockerMonitorEnabled(agentID string, enabled bool) error {
	return c.execAction(agentID, map[string]interface{}{"set_isDockerMonitor": enabled})
}

// GetUSBMonitorEnabled reports whether USB peripheral change monitoring
// (alerting on USB connect/disconnect) is enabled.
func (c *Client) GetUSBMonitorEnabled(agentID string) (bool, error) {
	raw, err := c.getPluginAction(agentID, `{"get_usb_change":true}`)
	if err != nil {
		return false, err
	}
	if raw == "" {
		return false, nil
	}

	var m map[string]string
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return false, fmt.Errorf("apphub: decode usb monitor status: %w", err)
	}
	return m["get_usb_change"] == "true", nil
}

// SetUSBMonitorEnabled turns USB peripheral change monitoring on or off.
func (c *Client) SetUSBMonitorEnabled(agentID string, enabled bool) error {
	return c.execAction(agentID, map[string]interface{}{
		"set_peripheral_monitor": []map[string]interface{}{{"usb_change": enabled}},
	})
}

// GetUSBStorageEnabled reports whether USB mass-storage access is
// currently enabled on the device (distinct from GetUSBMonitorEnabled,
// which is about change-alerting, not the storage toggle itself).
func (c *Client) GetUSBStorageEnabled(agentID string) (bool, error) {
	q := url.Values{}
	q.Set("pluginname", "devicecontrol")
	// Any actions value other than the special-cased "get_usb_status" /
	// "get_usb_change" / "get_threshold" / "get_software_info" /
	// "get_docker_info" strings makes the server fall through to a plain
	// LwM2M read of resource 27600, which is the USB-storage-enabled flag.
	q.Set("actions", `{"get_usb_storage_status":true}`)
	path := fmt.Sprintf("/api/plugins/%s/40008/0/27600?%s", url.PathEscape(agentID), q.Encode())

	var cr contentBoolResponse
	if err := c.do(http.MethodGet, path, nil, &cr); err != nil {
		return false, err
	}
	return cr.Content.Value, nil
}

// SetUSBStorageEnabled enables or disables USB mass-storage access on the device.
func (c *Client) SetUSBStorageEnabled(agentID string, enabled bool) error {
	return c.execAction(agentID, map[string]interface{}{"usb_storage_status": enabled})
}

// SyncTime triggers an immediate clock sync on the device.
//
// NOTE: not documented in AppHub-v2.3.0.md; inferred from the "syncTime"
// action key handled by DeviceLink's PluginsServlet for batch operations.
// It is not specially handled in the single-device write path either, so
// it should pass through to the device as-is, but hasn't been verified
// against real firmware.
func (c *Client) SyncTime(agentID string) error {
	return c.execAction(agentID, map[string]interface{}{"syncTime": true})
}

// --- system log ------------------------------------------------------------

// ListSystemLogs returns a page of device system log entries. Any of
// deviceID, os, keywords may be empty to leave that filter unset; beginTs/
// endTs are Unix milliseconds, pass 0 to omit either bound.
func (c *Client) ListSystemLogs(currentPage, limit int, deviceID, os, keywords string, beginTs, endTs int64) ([]SystemLogEntry, error) {
	q := url.Values{}
	q.Set("currentpage", fmt.Sprintf("%d", currentPage))
	q.Set("limit", fmt.Sprintf("%d", limit))
	if deviceID != "" {
		q.Set("agentId", deviceID)
	}
	if os != "" {
		q.Set("os", os)
	}
	if keywords != "" {
		q.Set("keywords", keywords)
	}
	if beginTs > 0 {
		q.Set("beginTs", fmt.Sprintf("%d", beginTs))
	}
	if endTs > 0 {
		q.Set("endTs", fmt.Sprintf("%d", endTs))
	}

	var logs []SystemLogEntry
	if err := c.do(http.MethodGet, "/api/systemLog/bypage?"+q.Encode(), nil, &logs); err != nil {
		return nil, err
	}
	return logs, nil
}

// DeleteSystemLog deletes a single system log entry by id.
func (c *Client) DeleteSystemLog(id int64) error {
	return c.do(http.MethodDelete, fmt.Sprintf("/api/systemLog/%d", id), nil, nil)
}

// DeleteSystemLogsByDevice deletes every system log entry for a device.
func (c *Client) DeleteSystemLogsByDevice(deviceID string) error {
	return c.do(http.MethodPost, "/api/systemLog/deleteByDeviceId", deleteByDeviceIDRequest{DeviceID: deviceID}, nil)
}

// BatchDeleteSystemLogs deletes multiple system log entries by id.
func (c *Client) BatchDeleteSystemLogs(ids []int64) error {
	return c.do(http.MethodPost, "/api/systemLog/batchDelete", batchIDsRequest{IDs: ids}, nil)
}

// --- operation records ------------------------------------------------------

// ListOperationRecords returns a page of admin operation-record (audit
// log) entries. target is an optional "contains" filter; beginTs/endTs are
// Unix milliseconds, pass 0 to omit either bound. This endpoint is not
// documented in AppHub-v2.3.0.md - see OperationRecordServlet.java.
func (c *Client) ListOperationRecords(currentPage, limit int, target string, beginTs, endTs int64) ([]OperationRecord, error) {
	q := url.Values{}
	q.Set("currentPage", fmt.Sprintf("%d", currentPage))
	q.Set("limit", fmt.Sprintf("%d", limit))
	if target != "" {
		q.Set("target", target)
	}
	if beginTs > 0 {
		q.Set("beginTs", fmt.Sprintf("%d", beginTs))
	}
	if endTs > 0 {
		q.Set("endTs", fmt.Sprintf("%d", endTs))
	}

	var page operationRecordsPageResponse
	if err := c.do(http.MethodGet, "/api/records/byPage?"+q.Encode(), nil, &page); err != nil {
		return nil, err
	}

	records := make([]OperationRecord, 0)
	if page.Data != "" {
		if err := json.Unmarshal([]byte(page.Data), &records); err != nil {
			return nil, fmt.Errorf("apphub: decode operation records: %w", err)
		}
	}
	return records, nil
}

// DeleteOperationRecord deletes a single operation record by id.
func (c *Client) DeleteOperationRecord(id int64) error {
	return c.do(http.MethodDelete, fmt.Sprintf("/api/records/%d", id), nil, nil)
}

// DeleteAllOperationRecords clears every operation record.
func (c *Client) DeleteAllOperationRecords() error {
	return c.do(http.MethodDelete, "/api/records/all", nil, nil)
}

// BatchDeleteOperationRecords deletes multiple operation records by id.
func (c *Client) BatchDeleteOperationRecords(ids []int64) error {
	return c.do(http.MethodPost, "/api/records/batchDelete", batchIDsRequest{IDs: ids}, nil)
}

// --- agent (client) log ------------------------------------------------------

// RequestAgentLogUpload asks a device's agent to upload its client log to
// callbackURL (must be an address the device can reach, running an HTTP
// server that accepts the agent's upload - this CLI does not provide one).
// actionID should be a fresh unique string; poll GetAgentLogStatus with the
// same value, then call DownloadAgentLog once it reports "success".
func (c *Client) RequestAgentLogUpload(agentID, callbackURL, actionID string) error {
	valueBytes, err := json.Marshal(agentLogRequestValue{Address: callbackURL, ActionID: actionID})
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/api/devicemgr/%s/40001/0/27012", url.PathEscape(agentID))
	body := execActionRequest{ID: 27012, Value: string(valueBytes)}
	var sr statusResponse
	return c.do(http.MethodPut, path, body, &sr)
}

// GetAgentLogStatus polls the status of a previously requested agent log
// upload (see RequestAgentLogUpload).
func (c *Client) GetAgentLogStatus(actionID string) (*logStatusResponse, error) {
	q := url.Values{}
	q.Set("actionId", actionID)
	var status logStatusResponse
	if err := c.do(http.MethodGet, "/api/devicemgr/logStatus?"+q.Encode(), nil, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// DownloadAgentLog downloads a device's uploaded client log (see
// RequestAgentLogUpload/GetAgentLogStatus) as a zip/log file. The server
// deletes the file after serving it, so it can only be downloaded once.
func (c *Client) DownloadAgentLog(agentID string) ([]byte, error) {
	path := fmt.Sprintf("/api/devicemgr/agentLog/%s", url.PathEscape(agentID))
	return c.downloadFile(path)
}
