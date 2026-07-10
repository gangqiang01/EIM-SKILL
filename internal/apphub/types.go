package apphub

import "encoding/json"

// Device is a device as returned by the AppHub/DeviceLink device manager API.
type Device struct {
	Endpoint         string `json:"endpoint"`
	IsOnline         bool   `json:"isonline"`
	RegistrationDate string `json:"registrationDate"`
	UpdateDate       string `json:"updateDate"`
	IPAddress        string `json:"ipaddress"`
	DevName          string `json:"devName"`
	BoardName        string `json:"boardName"`
	OS               string `json:"os"`
	SoftwareStatus   int    `json:"softwarestatus"`
	HardwareStatus   int    `json:"hardwarestatus"`
	SubDeviceStatus  string `json:"subDeviceStatus"`
	IsBind           bool   `json:"isBind"`
	IthingsEnable    string `json:"ithingsEnable"`
	DeviceStatus     int    `json:"deviceStatus"`
	Details          string `json:"details"`
	Lng              string `json:"lng"`
	Lat              string `json:"lat"`
}

// devicesByConditionResponse wraps GET /api/devicemgr/allDeviceByCondition's
// response: {"count":<n>,"data":[Device,...]}.
type devicesByConditionResponse struct {
	Count int64    `json:"count"`
	Data  []Device `json:"data"`
}

type createGroupRequest struct {
	GroupName string `json:"groupname"`
	Desc      string `json:"desc"`
	Type      string `json:"type"`
	Username  string `json:"username"`
}

// DeviceGroup is a device group as returned by the group manager API.
type DeviceGroup struct {
	GID    int    `json:"gid"`
	Name   string `json:"name"`
	Desc   string `json:"desc"`
	Type   string `json:"type"`
	Total  int    `json:"total"`
	Online int    `json:"online"`
	Ts     int64  `json:"ts"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Status       string `json:"status"`
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
}

type statusResponse struct {
	Status string `json:"status"`
	ErrMsg string `json:"errmsg"`
}

// MonitorDataPoint is a single CPU or memory monitoring sample for a device.
type MonitorDataPoint struct {
	ID    int64   `json:"id"`
	Value float64 `json:"value"`
	Ts    int64   `json:"ts"`
}

type setGroupsRequest struct {
	Endpoint   string   `json:"endpoint"`
	GroupNames []string `json:"groupnames"`
}

type execActionRequest struct {
	ID    int    `json:"id"`
	Value string `json:"value"`
}

type actionValue struct {
	PluginName string `json:"pluginname"`
	Actions    string `json:"actions"`
}

// contentResponse wraps LwM2M "read"-style responses, where the actual
// payload is a JSON string nested in content.value.
type contentResponse struct {
	Status  string `json:"status"`
	Content struct {
		ID    int    `json:"id"`
		Value string `json:"value"`
	} `json:"content"`
}

// InstalledLinuxPackage is a software package already installed on a Linux
// device, as reported by the device's devicecontrol object (60001/0/28300).
type InstalledLinuxPackage struct {
	AppName     string `json:"appname"`
	Description string `json:"description"`
	PkgName     string `json:"pkgname"`
	PkgType     string `json:"pkgtype"`
	Version     string `json:"version"`
	Status      string `json:"status"`
	UpdateTime  string `json:"updatetime"`
}

type deployLinuxPackageRequest struct {
	Taskname     string `json:"taskname"`
	ProductName  string `json:"productname"`
	Version      string `json:"version"`
	DownloadPath string `json:"downloadpath"`
	Type         string `json:"type"`
}

type deployLinuxPackageGroupRequest struct {
	GroupName    string `json:"groupname"`
	Taskname     string `json:"taskname"`
	ProductName  string `json:"productname"`
	Version      string `json:"version"`
	DownloadPath string `json:"downloadpath"`
	Type         string `json:"type"`
	LabelName    string `json:"labelName"`
}

// ThresholdSetting is one monitored metric's alert configuration, as
// reported by the device's devicecontrol "get_threshold" action.
type ThresholdSetting struct {
	Metric  string  `json:"metric"`
	Enabled bool    `json:"enabled"`
	Warn    float64 `json:"warn"`
	Error   float64 `json:"error"`
}

type rawThresholdSetting struct {
	Warn   json.Number `json:"warn"`
	Error  json.Number `json:"error"`
	Status string      `json:"status"`
}

// thresholdMetrics is the full set of metrics the devicecontrol
// "get_threshold" action reports, in display order. See
// monitorAlert/settingdetails/single.vue's tableList().
var thresholdMetrics = []string{
	"cpu_temp",
	"cpu_usage",
	"memory_usage",
	"storage_usage",
	"disk_health",
	"battery_available",
	"network_upload_speed",
	"network_download_speed",
}

// settableThresholdMetrics is the subset of thresholdMetrics that can be
// configured via "set_threshold" (disk_health is read-only in the web
// console - it has no warn/error levels to set).
var settableThresholdMetrics = map[string]bool{
	"cpu_temp":               true,
	"cpu_usage":              true,
	"memory_usage":           true,
	"storage_usage":          true,
	"battery_available":      true,
	"network_upload_speed":   true,
	"network_download_speed": true,
}

// ProcessInfo is a running process/thread on a Windows or Linux device, as
// reported by the devicecontrol "get_software_info" action.
type ProcessInfo struct {
	ThreadCmd        string `json:"threadcmd"`
	ThreadPid        string `json:"threadpid"`
	ThreadCPULoading string `json:"threadcpuloading"`
	ThreadMemLoading string `json:"threadmemloading"`
	ThreadStatus     string `json:"threadstatus"`
	ThreadUsername   string `json:"threadusername"`
	ThreadIsMonitor  string `json:"threadismonitor"`
}

// DockerContainer is a single container belonging to a DockerGroup.
type DockerContainer struct {
	Container string `json:"container"`
	Image     string `json:"image"`
	Version   string `json:"version"`
	IsRunning string `json:"isRunning"`
}

// DockerGroup is a deployed Docker compose/swarm/image group on a device,
// as reported by GET .../60001/0/27300.
type DockerGroup struct {
	Tag     string            `json:"tag"`
	Version string            `json:"version"`
	Type    string            `json:"type"`
	Control string            `json:"control"`
	Docker  []DockerContainer `json:"docker"`
}

type dockerInfoResponse struct {
	Code int           `json:"code"`
	Data []DockerGroup `json:"data"`
}

// SystemLogEntry is one row from the device system log.
type SystemLogEntry struct {
	ID       int64  `json:"id"`
	DeviceID string `json:"deviceId"`
	DevName  string `json:"devname"`
	Content  string `json:"content"`
	OS       string `json:"os"`
	Level    string `json:"level"`
	Mode     string `json:"mode"`
	Name     string `json:"name"`
	Pid      string `json:"pid"`
	EventID  string `json:"eventId"`
	Ts       int64  `json:"ts"`
}

type deleteByDeviceIDRequest struct {
	DeviceID string `json:"deviceId"`
}

type batchIDsRequest struct {
	IDs []int64 `json:"ids"`
}

// OperationRecord is one audit-log entry of an admin action (device
// control, config change, etc).
type OperationRecord struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Event    string `json:"event"`
	Target   string `json:"target"`
	CType    string `json:"ctype"`
	Ts       int64  `json:"ts"`
}

// operationRecordsPageResponse mirrors OperationRecordServlet's response:
// {"count":<n>,"data":"<json-encoded []OperationRecord>"} - data is
// double-JSON-encoded (a string containing JSON, not a nested array).
type operationRecordsPageResponse struct {
	Count int64  `json:"count"`
	Data  string `json:"data"`
}

// contentBoolResponse is like contentResponse, but for LwM2M reads that
// return a raw JSON boolean in content.value instead of a JSON-string.
type contentBoolResponse struct {
	Status  string `json:"status"`
	Content struct {
		ID    int  `json:"id"`
		Value bool `json:"value"`
	} `json:"content"`
}

// agentLogRequestValue is the plain (non-devicecontrol) LwM2M write value
// used to ask a device's agent to upload a log file to a callback address.
type agentLogRequestValue struct {
	Address  string `json:"address"`
	ActionID string `json:"actionId"`
}

type logStatusResponse struct {
	Result   string `json:"result"` // "running", "success", "timeout", or "error"
	ActionID string `json:"actionId"`
}
