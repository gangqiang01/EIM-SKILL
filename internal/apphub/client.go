// Package apphub is an HTTP client for the DeviceOn-EIM device
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

// Client talks to a DeviceOn-EIM server over HTTP.
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
		return fmt.Sprintf("DeviceOn-EIM: %s (http %d)", e.Message, e.StatusCode)
	}
	return fmt.Sprintf("DeviceOn-EIM: request failed (http %d)", e.StatusCode)
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
		return fmt.Errorf("DeviceOn-EIM: login request failed: %w", err)
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
		return fmt.Errorf("DeviceOn-EIM: decode login response: %w", err)
	}
	if loginResp.Token == "" {
		return fmt.Errorf("DeviceOn-EIM: login succeeded but no token was returned")
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
		return fmt.Errorf("DeviceOn-EIM: request %s %s failed: %w", method, path, err)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		if err := c.login(); err != nil {
			return err
		}
		resp, data, err = doOnce()
		if err != nil {
			return fmt.Errorf("DeviceOn-EIM: request %s %s failed: %w", method, path, err)
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
			return fmt.Errorf("DeviceOn-EIM: decode response for %s %s: %w", method, path, err)
		}
	}
	return nil
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

	var raw json.RawMessage
	if err := c.do(http.MethodGet, "/api/devicemgr/allDeviceByCondition?"+q.Encode(), nil, &raw); err != nil {
		return nil, err
	}
	return decodeDevicesByCondition(raw)
}

// decodeDevicesByCondition handles allDeviceByCondition's response, which
// has been observed as both a plain array (`[Device,...]`) and, per
// AppHub-v2.3.0.md's documented example, `{"count":<n>,"data":[Device,...]}`.
func decodeDevicesByCondition(raw json.RawMessage) ([]Device, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return []Device{}, nil
	}

	if trimmed[0] == '[' {
		var devices []Device
		if err := json.Unmarshal(trimmed, &devices); err != nil {
			return nil, fmt.Errorf("DeviceOn-EIM: decode devices: %w", err)
		}
		return devices, nil
	}

	var resp devicesByConditionResponse
	if err := json.Unmarshal(trimmed, &resp); err != nil {
		return nil, fmt.Errorf("DeviceOn-EIM: decode devices: %w", err)
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
		Details FlexibleJSON `json:"details"`
	}
	if err := c.do(http.MethodGet, "/api/devicemgr/details?"+q.Encode(), nil, &resp); err != nil {
		return nil, err
	}
	if len(resp.Details) == 0 {
		return nil, nil
	}
	return normalizeDeviceDetailsNumbers(json.RawMessage(resp.Details)), nil
}

// normalizeDeviceDetailsNumbers converts known numeric-string fields in the
// device details blob ("memorysize", and each storage entry's "total") into
// real JSON numbers - confirmed against a live server (both Linux and
// Windows devices) that these are always numeric strings (e.g.
// "memorysize":"7.64") rather than JSON numbers, unlike most other fields
// in this blob. The rest of the (loosely-typed, OS-varying) blob is passed
// through unchanged; if it doesn't parse as a JSON object for any reason,
// the original raw bytes are returned as-is rather than failing the whole
// response over a cosmetic normalization.
func normalizeDeviceDetailsNumbers(raw json.RawMessage) json.RawMessage {
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return raw
	}

	if s, ok := m["memorysize"].(string); ok {
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			m["memorysize"] = f
		}
	}
	if storage, ok := m["storage"].([]interface{}); ok {
		for _, entry := range storage {
			disk, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			if s, ok := disk["total"].(string); ok {
				if f, err := strconv.ParseFloat(s, 64); err == nil {
					disk["total"] = f
				}
			}
		}
	}

	out, err := json.Marshal(m)
	if err != nil {
		return raw
	}
	return out
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
		return nil, fmt.Errorf("DeviceOn-EIM: dataType must be \"cpu\" or \"memory\", got %q", dataType)
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

// GetDeviceCounts returns the number of online and offline devices.
func (c *Client) GetDeviceCounts() (*DeviceCounts, error) {
	var items []nameValueStat
	if err := c.do(http.MethodGet, "/api/devicemgr/alldevicestatus", nil, &items); err != nil {
		return nil, err
	}
	counts := &DeviceCounts{}
	for _, it := range items {
		switch it.Name {
		case "online":
			counts.Online = it.Value
		case "offline":
			counts.Offline = it.Value
		}
	}
	return counts, nil
}

// GetOnlineDeviceHealth returns the status breakdown (normal/warning/error)
// of currently online devices.
func (c *Client) GetOnlineDeviceHealth() (*OnlineDeviceHealth, error) {
	var items []nameValueStat
	if err := c.do(http.MethodGet, "/api/devicemgr/onlinedeviceabnormalstatus", nil, &items); err != nil {
		return nil, err
	}
	health := &OnlineDeviceHealth{}
	for _, it := range items {
		switch it.Name {
		case "normal":
			health.Normal = it.Value
		case "warning":
			health.Warning = it.Value
		case "error":
			health.Error = it.Value
		}
	}
	return health, nil
}

// ListGroupStatuses returns every device group's online/alarm status breakdown.
func (c *Client) ListGroupStatuses() ([]GroupStatus, error) {
	var raw []rawGroupStatus
	if err := c.do(http.MethodGet, "/api/groupmgr/allStatusAndGroup", nil, &raw); err != nil {
		return nil, err
	}
	statuses := make([]GroupStatus, 0, len(raw))
	for _, r := range raw {
		statuses = append(statuses, GroupStatus{
			Name:    r.GName,
			Type:    r.Type,
			Total:   r.Total,
			Online:  r.Online,
			Warning: r.Warning,
			Error:   r.Error,
		})
	}
	return statuses, nil
}

// GetGroupStatus returns one device group's online/alarm status breakdown by name.
func (c *Client) GetGroupStatus(groupName string) (*GroupStatus, error) {
	statuses, err := c.ListGroupStatuses()
	if err != nil {
		return nil, err
	}
	for _, s := range statuses {
		if s.Name == groupName {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("DeviceOn-EIM: no group found with name %q", groupName)
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
// device group it already belongs to. Returns the resulting full group
// list. Fails if groupName does not exist - the underlying
// SetDeviceGroups/bygroupname endpoint accepts and silently records
// nonexistent group names otherwise (confirmed against a live server).
func (c *Client) AddDeviceToGroup(endpoint, groupName string) ([]string, error) {
	allGroups, err := c.ListGroups()
	if err != nil {
		return nil, err
	}
	exists := false
	for _, g := range allGroups {
		if g.Name == groupName {
			exists = true
			break
		}
	}
	if !exists {
		return nil, fmt.Errorf("DeviceOn-EIM: group %q does not exist", groupName)
	}

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
		return nil, fmt.Errorf("DeviceOn-EIM: decode installed linux packages: %w", err)
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

// --- Android APP OTA ---

const (
	androidAppInfoResourceID = 27310
	androidAppDeployResource = "40004/0/27308" // install and update share this resource
	androidAppUninstallResID = 27305
)

// GetInstalledAndroidApps returns the raw (undocumented double-JSON-encoded)
// list of apps installed on an Android device: {"code":200,"data":[...]}.
func (c *Client) GetInstalledAndroidApps(agentID string) (json.RawMessage, error) {
	path := fmt.Sprintf("/api/devicemgr/%s/40004/0/%d", url.PathEscape(agentID), androidAppInfoResourceID)
	var cr contentResponse
	if err := c.do(http.MethodGet, path, nil, &cr); err != nil {
		return nil, err
	}
	if cr.Content.Value == "" {
		return nil, nil
	}
	return json.RawMessage(cr.Content.Value), nil
}

// DeployAndroidApp installs or updates an Android app on a single device.
// downloadPath is typically "<repo package address>/<repo package filename>".
func (c *Client) DeployAndroidApp(agentID, taskname, filename, pkgName, versionCode, version, downloadPath string) error {
	path := fmt.Sprintf("/api/devicemgr/%s/%s", url.PathEscape(agentID), androidAppDeployResource)
	body := deployAndroidAppRequest{
		Taskname:     taskname,
		Filename:     filename,
		PkgName:      pkgName,
		VersionCode:  versionCode,
		Version:      version,
		DownloadPath: downloadPath,
	}
	var sr statusResponse
	return c.do(http.MethodPost, path, body, &sr)
}

// DeployAndroidAppToGroup installs or updates an Android app on every
// device in groupName (and/or matching labelName, which may be empty).
func (c *Client) DeployAndroidAppToGroup(groupName, taskname, filename, pkgName, versionCode, version, downloadPath, labelName string) error {
	path := "/api/devicemgr/batchdeploy/" + androidAppDeployResource
	body := deployAndroidAppGroupRequest{
		GroupName:    groupName,
		Taskname:     taskname,
		Filename:     filename,
		PkgName:      pkgName,
		VersionCode:  versionCode,
		Version:      version,
		DownloadPath: downloadPath,
		LabelName:    labelName,
	}
	var sr statusResponse
	return c.do(http.MethodPost, path, body, &sr)
}

// UninstallAndroidApp removes an Android app (by package name) from a
// single device. Unlike every other OTA action, this is a plain LwM2M
// write ({"id","value"}), not the {taskname,...} shape.
func (c *Client) UninstallAndroidApp(agentID, pkgName string) error {
	path := fmt.Sprintf("/api/devicemgr/%s/40004/0/%d", url.PathEscape(agentID), androidAppUninstallResID)
	body := execActionRequest{ID: androidAppUninstallResID, Value: pkgName}
	var sr statusResponse
	return c.do(http.MethodPut, path, body, &sr)
}

// --- Windows exe OTA ---

const (
	winExeInstalledResourceID = 27300
	winExeInstallResource     = "50001/0/27301"
	winExeUpgradeResource     = "50001/0/27303"
	winExeUninstallResource   = "50001/0/27302"
)

// GetInstalledWindowsExePackages returns the raw (undocumented) list of
// exe packages installed on a Windows device via AppHub.
func (c *Client) GetInstalledWindowsExePackages(agentID string) (json.RawMessage, error) {
	path := fmt.Sprintf("/api/devicemgr/%s/50001/0/%d", url.PathEscape(agentID), winExeInstalledResourceID)
	var cr contentResponse
	if err := c.do(http.MethodGet, path, nil, &cr); err != nil {
		return nil, err
	}
	if cr.Content.Value == "" {
		return nil, nil
	}
	return json.RawMessage(cr.Content.Value), nil
}

// InstallWindowsExe installs an exe package (built with tool "Inno Setup"
// or "WiX Toolset") on a single device.
func (c *Client) InstallWindowsExe(agentID, taskname, projectName, tool, version, downloadPath string) error {
	return c.deployWindowsExe(agentID, winExeInstallResource, taskname, projectName, tool, version, downloadPath)
}

// UpgradeWindowsExe upgrades an already-installed exe package on a single device.
func (c *Client) UpgradeWindowsExe(agentID, taskname, projectName, tool, version, downloadPath string) error {
	return c.deployWindowsExe(agentID, winExeUpgradeResource, taskname, projectName, tool, version, downloadPath)
}

func (c *Client) deployWindowsExe(agentID, resource, taskname, projectName, tool, version, downloadPath string) error {
	path := fmt.Sprintf("/api/devicemgr/%s/%s", url.PathEscape(agentID), resource)
	body := deployWindowsExeRequest{
		Taskname:     taskname,
		ProjectName:  projectName,
		Tool:         tool,
		Version:      version,
		DownloadPath: downloadPath,
	}
	var sr statusResponse
	return c.do(http.MethodPost, path, body, &sr)
}

// InstallWindowsExeToGroup installs an exe package on every device in groupName.
func (c *Client) InstallWindowsExeToGroup(groupName, taskname, projectName, tool, version, downloadPath string) error {
	return c.deployWindowsExeToGroup(winExeInstallResource, groupName, taskname, projectName, tool, version, downloadPath)
}

// UpgradeWindowsExeToGroup upgrades an exe package on every device in groupName.
func (c *Client) UpgradeWindowsExeToGroup(groupName, taskname, projectName, tool, version, downloadPath string) error {
	return c.deployWindowsExeToGroup(winExeUpgradeResource, groupName, taskname, projectName, tool, version, downloadPath)
}

func (c *Client) deployWindowsExeToGroup(resource, groupName, taskname, projectName, tool, version, downloadPath string) error {
	path := "/api/devicemgr/batchdeploy/" + resource
	body := deployWindowsExeGroupRequest{
		GroupName:    groupName,
		Taskname:     taskname,
		ProjectName:  projectName,
		Tool:         tool,
		Version:      version,
		DownloadPath: downloadPath,
	}
	var sr statusResponse
	return c.do(http.MethodPost, path, body, &sr)
}

// UninstallWindowsExe removes an exe package from a single device.
func (c *Client) UninstallWindowsExe(agentID, taskname, projectName, tool, version string) error {
	path := fmt.Sprintf("/api/devicemgr/%s/%s", url.PathEscape(agentID), winExeUninstallResource)
	body := struct {
		Taskname    string `json:"taskname"`
		ProjectName string `json:"projectname"`
		Tool        string `json:"tool"`
		Version     string `json:"version"`
	}{taskname, projectName, tool, version}
	var sr statusResponse
	return c.do(http.MethodPost, path, body, &sr)
}

// --- Windows zip OTA ---

const (
	winZipInstalledResourceID = 28300
	winZipInstallResource     = "50001/0/28301"
	winZipUpdateResource      = "50001/0/28303"
	winZipUninstallResource   = "50001/0/28302"
)

// GetInstalledWindowsZipPackages returns the raw (undocumented) list of
// zip/compressed packages installed on a Windows device via AppHub.
func (c *Client) GetInstalledWindowsZipPackages(agentID string) (json.RawMessage, error) {
	path := fmt.Sprintf("/api/devicemgr/%s/50001/0/%d", url.PathEscape(agentID), winZipInstalledResourceID)
	var cr contentResponse
	if err := c.do(http.MethodGet, path, nil, &cr); err != nil {
		return nil, err
	}
	if cr.Content.Value == "" {
		return nil, nil
	}
	return json.RawMessage(cr.Content.Value), nil
}

// InstallWindowsZip installs a zip/compressed package on a single device.
// parameter is optional (arguments passed to the extracted package).
func (c *Client) InstallWindowsZip(agentID, taskname, projectName, version, downloadPath, parameter string) error {
	return c.deployWindowsZip(agentID, winZipInstallResource, taskname, projectName, version, downloadPath, parameter)
}

// UpdateWindowsZip updates an already-installed zip/compressed package on a single device.
func (c *Client) UpdateWindowsZip(agentID, taskname, projectName, version, downloadPath, parameter string) error {
	return c.deployWindowsZip(agentID, winZipUpdateResource, taskname, projectName, version, downloadPath, parameter)
}

func (c *Client) deployWindowsZip(agentID, resource, taskname, projectName, version, downloadPath, parameter string) error {
	path := fmt.Sprintf("/api/devicemgr/%s/%s", url.PathEscape(agentID), resource)
	body := deployWindowsZipRequest{
		Taskname:     taskname,
		ProjectName:  projectName,
		Version:      version,
		DownloadPath: downloadPath,
		Parameter:    parameter,
	}
	var sr statusResponse
	return c.do(http.MethodPost, path, body, &sr)
}

// InstallWindowsZipToGroup installs a zip package on every device in groupName.
func (c *Client) InstallWindowsZipToGroup(groupName, taskname, projectName, version, downloadPath, parameter, labelName string) error {
	return c.deployWindowsZipToGroup(winZipInstallResource, groupName, taskname, projectName, version, downloadPath, parameter, labelName)
}

// UpdateWindowsZipToGroup updates a zip package on every device in groupName.
func (c *Client) UpdateWindowsZipToGroup(groupName, taskname, projectName, version, downloadPath, parameter, labelName string) error {
	return c.deployWindowsZipToGroup(winZipUpdateResource, groupName, taskname, projectName, version, downloadPath, parameter, labelName)
}

func (c *Client) deployWindowsZipToGroup(resource, groupName, taskname, projectName, version, downloadPath, parameter, labelName string) error {
	path := "/api/devicemgr/batchdeploy/" + resource
	body := deployWindowsZipGroupRequest{
		GroupName:    groupName,
		Taskname:     taskname,
		ProjectName:  projectName,
		Version:      version,
		DownloadPath: downloadPath,
		Parameter:    parameter,
		LabelName:    labelName,
	}
	var sr statusResponse
	return c.do(http.MethodPost, path, body, &sr)
}

// UninstallWindowsZip removes a zip package from a single device.
func (c *Client) UninstallWindowsZip(agentID, taskname, projectName, version string) error {
	path := fmt.Sprintf("/api/devicemgr/%s/%s", url.PathEscape(agentID), winZipUninstallResource)
	body := struct {
		Taskname    string `json:"taskname"`
		ProjectName string `json:"projectname"`
		Version     string `json:"version"`
	}{taskname, projectName, version}
	var sr statusResponse
	return c.do(http.MethodPost, path, body, &sr)
}

// --- Docker compose OTA ---

const (
	dockerComposeDeployResource    = "60001/0/27301" // install and update share this resource
	dockerComposeUninstallResource = "60001/0/27302"
)

// DeployDockerCompose installs or updates a Docker compose/swarm package
// on a single device.
func (c *Client) DeployDockerCompose(agentID, taskname, filename, projectName, version, downloadPath string) error {
	path := fmt.Sprintf("/api/devicemgr/%s/%s", url.PathEscape(agentID), dockerComposeDeployResource)
	body := deployDockerComposeRequest{
		Taskname:     taskname,
		Filename:     filename,
		ProjectName:  projectName,
		Version:      version,
		DownloadPath: downloadPath,
	}
	var sr statusResponse
	return c.do(http.MethodPost, path, body, &sr)
}

// DeployDockerComposeToGroup installs or updates a Docker compose/swarm
// package on every device in groupName.
func (c *Client) DeployDockerComposeToGroup(groupName, taskname, filename, projectName, version, downloadPath, labelName string) error {
	path := "/api/devicemgr/batchdeploy/" + dockerComposeDeployResource
	body := deployDockerComposeGroupRequest{
		GroupName:    groupName,
		Taskname:     taskname,
		Filename:     filename,
		ProjectName:  projectName,
		Version:      version,
		DownloadPath: downloadPath,
		LabelName:    labelName,
	}
	var sr statusResponse
	return c.do(http.MethodPost, path, body, &sr)
}

// UninstallDockerCompose removes a Docker compose/swarm package from a single device.
func (c *Client) UninstallDockerCompose(agentID, taskname, projectName, version string) error {
	path := fmt.Sprintf("/api/devicemgr/%s/%s", url.PathEscape(agentID), dockerComposeUninstallResource)
	body := struct {
		Taskname    string `json:"taskname"`
		ProjectName string `json:"projectname"`
		Version     string `json:"version"`
	}{taskname, projectName, version}
	var sr statusResponse
	return c.do(http.MethodPost, path, body, &sr)
}

// --- generic file deploy ---

const fileDeployResource = "40003/0/27206"

// DeployFile writes a file to savepath on a single device, optionally
// making it executable (isExec).
func (c *Client) DeployFile(agentID, taskname, savePath, filename, downloadPath string, isExec bool) error {
	path := fmt.Sprintf("/api/devicemgr/%s/%s", url.PathEscape(agentID), fileDeployResource)
	body := deployFileRequest{
		Taskname:     taskname,
		SavePath:     savePath,
		Filename:     filename,
		DownloadPath: downloadPath,
		IsExec:       isExec,
	}
	var sr statusResponse
	return c.do(http.MethodPost, path, body, &sr)
}

// DeployFileToGroup writes a file to savepath on every device in groupName.
func (c *Client) DeployFileToGroup(groupName, taskname, savePath, filename, downloadPath string, isExec bool, labelName string) error {
	path := "/api/devicemgr/batchdeploy/" + fileDeployResource
	body := deployFileGroupRequest{
		GroupName:    groupName,
		Taskname:     taskname,
		SavePath:     savePath,
		Filename:     filename,
		DownloadPath: downloadPath,
		IsExec:       isExec,
		LabelName:    labelName,
	}
	var sr statusResponse
	return c.do(http.MethodPost, path, body, &sr)
}

// --- OTA deploy status ------------------------------------------------------
//
// These endpoints are not documented in AppHub-v2.3.0.md; they were found
// via an apifox-exported OpenAPI spec (apphub-web's appStatusApi.js/
// fileDeployApi.js/updateBSPApi.js source files) and confirmed to respond
// (200, no error) against a live server, but this environment had zero
// deploy-task history recorded at the time, so the exact non-empty
// response shape and the "type" business-classifier's value per package
// type are NOT independently confirmed beyond Android (documented as the
// fixed literal "sg" for /api/appstatusmgr/bypageindevid). GetAppDeployStatus
// is used as-is (type="sg") for every appstatusmgr-backed package type
// (linux/android/windows-exe/windows-zip/docker) since "sg" is described
// in the docs as a *fixed* value rather than a per-type enum, suggesting
// it may be shared across the whole manager - but this is a best-effort
// assumption for every type except Android. Raw JSON is returned
// unparsed rather than guessing at a Go struct for an unconfirmed shape.

// GetAppDeployStatus returns the most recent app-deployment status
// record(s) for a device from the shared Android/Linux/Windows/Docker
// "appstatusmgr" manager. See the OTA deploy status section note above -
// only confirmed for Android.
func (c *Client) GetAppDeployStatus(agentID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("endpoint", agentID)
	q.Set("type", "sg")
	var raw json.RawMessage
	if err := c.do(http.MethodGet, "/api/appstatusmgr/bydevid?"+q.Encode(), nil, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// GetFileDeployStatus returns the most recent generic-file deployment
// status record(s) for a device and repo file --type category. See the
// OTA deploy status section note above.
func (c *Client) GetFileDeployStatus(agentID, fileType string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("endpoint", agentID)
	q.Set("type", fileType)
	var raw json.RawMessage
	if err := c.do(http.MethodGet, "/api/filestatusmgr/bydevid?"+q.Encode(), nil, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// GetBSPUpdateStatus returns BSP/system-update deployment status
// record(s) for a device. Unlike GetAppDeployStatus/GetFileDeployStatus,
// this endpoint takes no "type" parameter at all - BSP updates are their
// own single business domain, not shared across package types, so there's
// no type-guessing risk here.
func (c *Client) GetBSPUpdateStatus(agentID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("endpoint", agentID)
	var raw json.RawMessage
	if err := c.do(http.MethodGet, "/api/deploymgr/bydevid?"+q.Encode(), nil, &raw); err != nil {
		return nil, err
	}
	return raw, nil
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
// AppHub-v2.3.0.md. The action key is "set_audio" - confirmed against a
// live server after an earlier "audio" key was found (via a real bug
// report) to return {"status":"CHANGED"} without actually changing
// anything on the device.
func (c *Client) SetVolume(agentID string, volume int) error {
	return c.execAction(agentID, map[string]interface{}{"set_audio": volume})
}

// SetBacklight sets the device's screen backlight/brightness (0-100).
//
// NOTE: unlike Reboot/Shutdown, this action is not documented in
// AppHub-v2.3.0.md. The action key is "set_brightness" - confirmed against
// a live server after an earlier "brightness" key was found (via a real
// bug report) to return {"status":"CHANGED"} without actually changing
// anything on the device.
func (c *Client) SetBacklight(agentID string, brightness int) error {
	return c.execAction(agentID, map[string]interface{}{"set_brightness": brightness})
}

// GetVolume returns the device's current system audio volume (0-100), as
// last recorded by SetVolume - nil if unavailable.
//
// NOTE: not documented in AppHub-v2.3.0.md. Confirmed against a live
// server that this reflects whatever SetVolume last recorded, but not
// confirmed whether it reflects the actual physical hardware volume on
// every device model (some devices may accept and record the value
// without applying it to real hardware).
func (c *Client) GetVolume(agentID string) (*float64, error) {
	raw, err := c.getPluginAction(agentID, `{'get_audio':true}`)
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return nil, nil
	}
	// The response wraps the value under a "get_audio" key (echoing the
	// action name) - unlike get_brightness, which wraps it under
	// "brightness" instead of "get_brightness". Both confirmed against a
	// live server.
	var resp struct {
		Volume FlexibleNumber `json:"get_audio"`
	}
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("DeviceOn-EIM: decode volume: %w", err)
	}
	return resp.Volume.Value, nil
}

// GetBacklight returns the device's current screen backlight/brightness
// (0-100), as last recorded by SetBacklight - nil if unavailable.
//
// NOTE: not documented in AppHub-v2.3.0.md. Confirmed against a live
// server that this reflects whatever SetBacklight last recorded, but not
// confirmed whether it reflects the actual physical hardware brightness
// on every device model (some devices may accept and record the value
// without applying it to real hardware).
func (c *Client) GetBacklight(agentID string) (*float64, error) {
	raw, err := c.getPluginAction(agentID, `{'get_brightness':true}`)
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return nil, nil
	}
	var resp struct {
		Brightness FlexibleNumber `json:"brightness"`
	}
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("DeviceOn-EIM: decode brightness: %w", err)
	}
	return resp.Brightness.Value, nil
}

// execActionGroup sends a devicecontrol action to every device in groupName
// (optionally further filtered to devices matching labelName, which may be
// empty).
//
// This resolves groupName/labelName to a concrete list of device endpoints
// via ListDevicesByCondition, then executes the action against all of them
// in one call via /api/plugins/batchExec/40008/0/27601 (documented in
// AppHub-v2.3.0.md as "PUT 指定批量设备立即关机", body
// {"clientIds":[...],"value":...} - note the doc's own example body has a
// typo, "cliendIds"; the real parameter is "clientIds", confirmed against
// a live server).
//
// AppHub-v2.3.0.md also documents a simpler-looking
// PUT /api/plugins/batch/{groupname}/40008/0/27601 that takes a groupname
// directly with no client-side resolution step - that was this function's
// first implementation, but it was found (via a real bug report - it
// consistently returns an empty-bodied HTTP 500 for every payload tried,
// reproduced independently against a second server) to not actually work,
// so this now goes through batchExec instead.
func (c *Client) execActionGroup(groupName string, actions map[string]interface{}, labelName string) error {
	devices, err := c.ListDevicesByCondition(nil, groupName, "", labelName)
	if err != nil {
		return err
	}
	if len(devices) == 0 {
		if labelName != "" {
			return fmt.Errorf("DeviceOn-EIM: no devices found in group %q with label %q", groupName, labelName)
		}
		return fmt.Errorf("DeviceOn-EIM: no devices found in group %q", groupName)
	}
	clientIDs := make([]string, len(devices))
	for i, d := range devices {
		clientIDs[i] = d.Endpoint
	}

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

	body := execBatchRequest{ClientIDs: clientIDs, Value: string(valueBytes)}
	var sr statusResponse
	return c.do(http.MethodPut, "/api/plugins/batchExec/40008/0/27601", body, &sr)
}

// RebootGroup immediately reboots every device in groupName.
func (c *Client) RebootGroup(groupName, labelName string) error {
	return c.execActionGroup(groupName, map[string]interface{}{"reboot": true}, labelName)
}

// ShutdownGroup immediately shuts down every device in groupName.
func (c *Client) ShutdownGroup(groupName, labelName string) error {
	return c.execActionGroup(groupName, map[string]interface{}{"shutdown": true}, labelName)
}

// SetVolumeGroup sets the system audio volume (0-100) on every device in
// groupName. See the NOTE on SetVolume - undocumented, best effort.
func (c *Client) SetVolumeGroup(groupName string, volume int, labelName string) error {
	return c.execActionGroup(groupName, map[string]interface{}{"set_audio": volume}, labelName)
}

// SetBacklightGroup sets the screen backlight/brightness (0-100) on every
// device in groupName. See the NOTE on SetBacklight - undocumented, best effort.
func (c *Client) SetBacklightGroup(groupName string, brightness int, labelName string) error {
	return c.execActionGroup(groupName, map[string]interface{}{"set_brightness": brightness}, labelName)
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

// GetBoardInfo returns a device's board-level monitoring snapshot
// (CPU/memory/storage/battery/network), normalized into consistent
// float64 metrics regardless of how the agent encoded them - see
// BoardInfo. Not documented in AppHub-v2.3.0.md.
func (c *Client) GetBoardInfo(agentID string) (*BoardInfo, error) {
	q := url.Values{}
	q.Set("pluginname", "devicecontrol")
	q.Set("actions", `{'get_board_info':true}`)
	path := fmt.Sprintf("/api/plugins/%s/40008/0/27600?%s", url.PathEscape(agentID), q.Encode())

	var cr contentFlexibleResponse
	if err := c.do(http.MethodGet, path, nil, &cr); err != nil {
		return nil, err
	}
	if len(cr.Content.Value) == 0 {
		return &BoardInfo{NetIO: []BoardNetIOStat{}}, nil
	}

	var raw rawBoardInfo
	if err := json.Unmarshal(cr.Content.Value, &raw); err != nil {
		return nil, fmt.Errorf("DeviceOn-EIM: decode board info: %w", err)
	}
	return raw.normalize(), nil
}

// GetThresholds returns the device's current monitoring threshold
// configuration for CPU/memory/storage/network/battery metrics.
// GetMonitoredProcesses returns the processes currently selected for
// application/process monitoring, normalized from the agent's
// string-typed fields into proper numbers/bools. Not documented in
// AppHub-v2.3.0.md; see MonitoredProcess for details.
func (c *Client) GetMonitoredProcesses(agentID string) ([]MonitoredProcess, error) {
	raw, err := c.getPluginAction(agentID, `{'get_monitor_app':true}`)
	if err != nil {
		return nil, err
	}

	processes := make([]MonitoredProcess, 0)
	if raw == "" {
		return processes, nil
	}

	var rawProcesses []rawMonitoredProcess
	if err := json.Unmarshal([]byte(raw), &rawProcesses); err != nil {
		return nil, fmt.Errorf("DeviceOn-EIM: decode monitored processes: %w", err)
	}
	for _, p := range rawProcesses {
		processes = append(processes, p.normalize())
	}
	return processes, nil
}

func (c *Client) GetThresholds(agentID string) ([]ThresholdSetting, error) {
	raw, err := c.getPluginAction(agentID, `{'get_threshold':true}`)
	if err != nil {
		return nil, err
	}

	settings := make([]ThresholdSetting, 0, len(thresholdMetrics))
	if raw == "" {
		return settings, nil
	}

	var m map[string]rawThresholdSetting
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, fmt.Errorf("DeviceOn-EIM: decode thresholds: %w", err)
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
		return fmt.Errorf("DeviceOn-EIM: unsupported threshold metric %q", metric)
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

// GetDockerInfo returns the Docker containers on a device that are
// currently selected for monitoring (IsMonitor == "true"). The
// devicecontrol "get_docker_info" action itself returns every container
// regardless of monitor status; this filters to the monitored subset.
func (c *Client) GetDockerInfo(agentID string) ([]DockerContainer, error) {
	raw, err := c.getPluginAction(agentID, `{'get_docker_info':true}`)
	if err != nil {
		return nil, err
	}
	containers := make([]DockerContainer, 0)
	if raw == "" {
		return containers, nil
	}
	if err := json.Unmarshal([]byte(raw), &containers); err != nil {
		return nil, fmt.Errorf("DeviceOn-EIM: decode docker info: %w", err)
	}

	monitored := make([]DockerContainer, 0, len(containers))
	for _, container := range containers {
		if container.IsMonitor == "true" {
			monitored = append(monitored, container)
		}
	}
	return monitored, nil
}

// GetUSBMonitorEnabled reports whether USB peripheral change monitoring
// (alerting on USB connect/disconnect) is enabled. Confirmed against a
// live server via a real packet capture, and cross-verified by toggling
// SetUSBMonitorEnabled true/false and observing this correctly reflects
// each change.
func (c *Client) GetUSBMonitorEnabled(agentID string) (bool, error) {
	raw, err := c.getPluginAction(agentID, `{'get_usb_change':true}`)
	if err != nil {
		return false, err
	}
	if raw == "" {
		return false, nil
	}

	var m map[string]string
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return false, fmt.Errorf("DeviceOn-EIM: decode usb monitor status: %w", err)
	}
	return m["get_usb_change"] == "true", nil
}

// SetUSBMonitorEnabled turns USB peripheral change monitoring on or off.
// Confirmed against a live server via a real packet capture, and
// cross-verified against GetUSBMonitorEnabled: toggling this true/false
// and re-reading GetUSBMonitorEnabled correctly reflects each change.
//
// NOTE: an earlier version of this client briefly used
// "set_usb_storage_status" instead (a different, separate feature - USB
// mass-storage access - based on a packet capture that turned out to be
// for that other feature, not this one). That action's SET call
// succeeded and was logged under a friendly audit-log event name
// ("enableUsb"/"disableUsb"), which seemed like strong evidence at the
// time, but its GET side ("get_usb_storage_status") never reflected the
// change (always returned an empty result) - unlike get_usb_change here,
// which reliably reflects set_peripheral_monitor. A GET that actually
// reflects the SET is stronger evidence of a real, working toggle than
// an audit-log label, so this was reverted back to
// set_peripheral_monitor/get_usb_change.
func (c *Client) SetUSBMonitorEnabled(agentID string, enabled bool) error {
	return c.execAction(agentID, map[string]interface{}{
		"set_peripheral_monitor": []map[string]interface{}{{"usb_change": enabled}},
	})
}

// SetSystemLogMonitorEnabled turns systemd log collection/reporting on or
// off for a device. This feeds ListSystemLogs's data - without it, system
// log queries for that device may return nothing.
//
// NOTE: not documented in AppHub-v2.3.0.md; inferred from a captured
// devicecontrol request using the "set_monitor_data_report" action.
func (c *Client) SetSystemLogMonitorEnabled(agentID string, enabled bool) error {
	return c.execAction(agentID, map[string]interface{}{
		"set_monitor_data_report": []map[string]interface{}{{"systemd_log_monitor": enabled}},
	})
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
	return page.Data, nil
}

// DeleteAllOperationRecords clears every operation record.
func (c *Client) DeleteAllOperationRecords() error {
	return c.do(http.MethodDelete, "/api/records/all", nil, nil)
}

// --- system log --------------------------------------------------------------

// ListSystemLogs returns a page of device system log entries (fed by
// SetSystemLogMonitorEnabled turning collection on for that device). Any
// of deviceID, os, keywords, mode, level may be empty to leave that filter
// unset; beginTs/endTs are Unix milliseconds, pass 0 to omit either bound.
//
// NOTE: not documented in AppHub-v2.3.0.md. The path is /api/systemLog/
// byPage (capital P) - confirmed against a live server via a real packet
// capture; an earlier version of this client used the lowercase
// /api/systemLog/bypage, which the server rejects with an HTTP 400
// "Invalid path" error.
func (c *Client) ListSystemLogs(currentPage, limit int, deviceID, os, keywords, mode, level string, beginTs, endTs int64) ([]SystemLogEntry, error) {
	q := url.Values{}
	q.Set("currentpage", fmt.Sprintf("%d", currentPage))
	q.Set("limit", fmt.Sprintf("%d", limit))
	q.Set("keywords", keywords)
	q.Set("agentId", deviceID)
	q.Set("os", os)
	q.Set("mode", mode)
	q.Set("level", level)
	if beginTs > 0 {
		q.Set("beginTs", fmt.Sprintf("%d", beginTs))
	} else {
		q.Set("beginTs", "")
	}
	if endTs > 0 {
		q.Set("endTs", fmt.Sprintf("%d", endTs))
	} else {
		q.Set("endTs", "")
	}

	var page systemLogsPageResponse
	if err := c.do(http.MethodGet, "/api/systemLog/byPage?"+q.Encode(), nil, &page); err != nil {
		return nil, err
	}
	return page.Data, nil
}

// DeleteSystemLogsByDevice deletes every system log entry for a device.
func (c *Client) DeleteSystemLogsByDevice(deviceID string) error {
	return c.do(http.MethodPost, "/api/systemLog/deleteByDeviceId", deleteByDeviceIDRequest{DeviceID: deviceID}, nil)
}
