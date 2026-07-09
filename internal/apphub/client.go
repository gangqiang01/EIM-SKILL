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

// GetDevice returns the details of a single device.
func (c *Client) GetDevice(agentID string) (*Device, error) {
	var device Device
	path := "/api/devicemgr/" + url.PathEscape(agentID)
	if err := c.do(http.MethodGet, path, nil, &device); err != nil {
		return nil, err
	}
	return &device, nil
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
