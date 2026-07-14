package apphub

import (
	"encoding/json"
	"strconv"
	"strings"
)

// FlexibleNumber decodes a JSON value that may be a real number, a
// numeric-looking string (device agents often report metrics as strings),
// or a non-numeric placeholder like "N/A"/"" for "unavailable". It always
// exposes a *float64: nil means the value wasn't available, never a magic
// number or sentinel string.
type FlexibleNumber struct {
	Value *float64
}

// UnmarshalJSON implements json.Unmarshaler.
func (n *FlexibleNumber) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		n.Value = nil
		return nil
	}

	if trimmed[0] != '"' {
		var f float64
		if err := json.Unmarshal(data, &f); err != nil {
			return err
		}
		n.Value = &f
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "N/A") {
		n.Value = nil
		return nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		// Not a number we recognize - report "unavailable" rather than
		// failing the whole response over one unexpected value.
		n.Value = nil
		return nil
	}
	n.Value = &f
	return nil
}

// MarshalJSON implements json.Marshaler.
func (n FlexibleNumber) MarshalJSON() ([]byte, error) {
	if n.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*n.Value)
}

// FlexibleJSON stores a JSON value that different AppHub/DeviceLink
// endpoints encode inconsistently for what is logically the same field:
// sometimes a JSON string containing escaped JSON, sometimes a raw JSON
// object/array. Unmarshaling accepts either form and always stores the
// unwrapped JSON, so callers (and re-marshaling for CLI output) see a
// single consistent shape regardless of which endpoint it came from.
type FlexibleJSON json.RawMessage

// UnmarshalJSON implements json.Unmarshaler.
func (f *FlexibleJSON) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*f = nil
		return nil
	}
	if data[0] != '"' {
		// Already a raw JSON object/array/number/bool - keep as-is.
		*f = append(FlexibleJSON(nil), data...)
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == "" {
		*f = nil
		return nil
	}
	if json.Valid([]byte(s)) {
		*f = FlexibleJSON(s)
		return nil
	}
	// The string wasn't itself JSON - keep it as a proper JSON string
	// rather than corrupting output by embedding raw text.
	reencoded, err := json.Marshal(s)
	if err != nil {
		return err
	}
	*f = FlexibleJSON(reencoded)
	return nil
}

// MarshalJSON implements json.Marshaler.
func (f FlexibleJSON) MarshalJSON() ([]byte, error) {
	if len(f) == 0 {
		return []byte("null"), nil
	}
	return f, nil
}

// Device is a device as returned by the AppHub/DeviceLink device manager API.
type Device struct {
	Endpoint         string       `json:"endpoint"`
	IsOnline         bool         `json:"isonline"`
	RegistrationDate string       `json:"registrationDate"`
	UpdateDate       string       `json:"updateDate"`
	IPAddress        string       `json:"ipaddress"`
	DevName          string       `json:"devName"`
	BoardName        string       `json:"boardName"`
	OS               string       `json:"os"`
	SoftwareStatus   int          `json:"softwarestatus"`
	HardwareStatus   int          `json:"hardwarestatus"`
	SubDeviceStatus  string       `json:"subDeviceStatus"`
	IsBind           bool         `json:"isBind"`
	IthingsEnable    string       `json:"ithingsEnable"`
	DeviceStatus     int          `json:"deviceStatus"`
	DeviceStatusText string       `json:"deviceStatusText"`
	Details          FlexibleJSON `json:"details"`
	Lng              string       `json:"lng"`
	Lat              string       `json:"lat"`
}

// UnmarshalJSON implements json.Unmarshaler. Beyond the normal field
// decoding, it derives DeviceStatusText from DeviceStatus (the server's
// combined software/hardware status code: 0=normal, 1=error, 2=warning -
// see AppHub-v2.3.0.md "获取指定设备组的设备的软件、硬件运行状态"), so
// callers get a human-readable label instead of having to know the code.
func (d *Device) UnmarshalJSON(data []byte) error {
	type deviceAlias Device
	var alias deviceAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*d = Device(alias)
	d.DeviceStatusText = deviceStatusText(d.DeviceStatus)
	return nil
}

func deviceStatusText(status int) string {
	switch status {
	case 0:
		return "normal"
	case 1:
		return "error"
	case 2:
		return "warning"
	default:
		return "unknown"
	}
}

// devicesByConditionResponse is the documented (but not always observed in
// practice - see decodeDevicesByCondition) shape of GET
// /api/devicemgr/allDeviceByCondition: {"count":<n>,"data":[Device,...]}.
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

// nameValueStat is the raw {"name":...,"value":...} shape used by several
// AppHub summary-stat endpoints (alldevicestatus, onlinedeviceabnormalstatus).
type nameValueStat struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

// DeviceCounts is the number of online/offline devices, normalized from
// GET /api/devicemgr/alldevicestatus's [{"name":"online","value":N},...] shape.
type DeviceCounts struct {
	Online  int `json:"online"`
	Offline int `json:"offline"`
}

// OnlineDeviceHealth is the status breakdown of online devices, normalized
// from GET /api/devicemgr/onlinedeviceabnormalstatus's
// [{"name":"normal","value":N},...] shape.
type OnlineDeviceHealth struct {
	Normal  int `json:"normal"`
	Warning int `json:"warning"`
	Error   int `json:"error"`
}

// rawGroupStatus is GET /api/groupmgr/allStatusAndGroup's raw per-group shape.
type rawGroupStatus struct {
	GName   string `json:"gname"`
	Type    string `json:"type"`
	Total   int    `json:"total"`
	Online  int    `json:"online"`
	Warning int    `json:"warning"`
	Error   int    `json:"error"`
}

// GroupStatus is one device group's online/alarm status breakdown
// (normalized from rawGroupStatus - "gname" becomes "name" for
// consistency with DeviceGroup and other group-shaped output).
type GroupStatus struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Total   int    `json:"total"`
	Online  int    `json:"online"`
	Warning int    `json:"warning"`
	Error   int    `json:"error"`
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

// contentFlexibleResponse is like contentResponse, but for actions whose
// content.value shape isn't confirmed ahead of time - FlexibleJSON accepts
// either a JSON-encoded string or a raw JSON object/array.
type contentFlexibleResponse struct {
	Status  string `json:"status"`
	Content struct {
		ID    int          `json:"id"`
		Value FlexibleJSON `json:"value"`
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

// --- Android APP OTA ---

type deployAndroidAppRequest struct {
	Taskname     string `json:"taskname"`
	Filename     string `json:"filename"`
	PkgName      string `json:"pkgname"`
	VersionCode  string `json:"versioncode"`
	Version      string `json:"version"`
	DownloadPath string `json:"downloadpath"`
}

type deployAndroidAppGroupRequest struct {
	GroupName    string `json:"groupname"`
	Taskname     string `json:"taskname"`
	Filename     string `json:"filename"`
	PkgName      string `json:"pkgname"`
	VersionCode  string `json:"versioncode"`
	Version      string `json:"version"`
	DownloadPath string `json:"downloadpath"`
	LabelName    string `json:"labelName"`
}

// --- Windows exe OTA ---

type deployWindowsExeRequest struct {
	Taskname     string `json:"taskname"`
	ProjectName  string `json:"projectname"`
	Tool         string `json:"tool"`
	Version      string `json:"version"`
	DownloadPath string `json:"downloadpath"`
}

type deployWindowsExeGroupRequest struct {
	GroupName    string `json:"groupname"`
	Taskname     string `json:"taskname"`
	ProjectName  string `json:"projectname"`
	Tool         string `json:"tool"`
	Version      string `json:"version"`
	DownloadPath string `json:"downloadpath"`
}

// --- Windows zip OTA ---

type deployWindowsZipRequest struct {
	Taskname     string `json:"taskname"`
	ProjectName  string `json:"projectname"`
	Version      string `json:"version"`
	DownloadPath string `json:"downloadpath"`
	Parameter    string `json:"parameter,omitempty"`
}

type deployWindowsZipGroupRequest struct {
	GroupName    string `json:"groupname"`
	Taskname     string `json:"taskname"`
	ProjectName  string `json:"projectname"`
	Version      string `json:"version"`
	DownloadPath string `json:"downloadpath"`
	Parameter    string `json:"parameter,omitempty"`
	LabelName    string `json:"labelName,omitempty"`
}

// --- Docker compose OTA ---

type deployDockerComposeRequest struct {
	Taskname     string `json:"taskname"`
	Filename     string `json:"filename"`
	ProjectName  string `json:"projectname"`
	Version      string `json:"version"`
	DownloadPath string `json:"downloadpath"`
}

type deployDockerComposeGroupRequest struct {
	GroupName    string `json:"groupname"`
	Taskname     string `json:"taskname"`
	Filename     string `json:"filename"`
	ProjectName  string `json:"projectname"`
	Version      string `json:"version"`
	DownloadPath string `json:"downloadpath"`
	LabelName    string `json:"labelName,omitempty"`
}

// --- generic file deploy ---

type deployFileRequest struct {
	Taskname     string `json:"taskname"`
	SavePath     string `json:"savepath"`
	Filename     string `json:"filename"`
	DownloadPath string `json:"downloadpath"`
	IsExec       bool   `json:"isExec"`
}

type deployFileGroupRequest struct {
	GroupName    string `json:"groupname"`
	Taskname     string `json:"taskname"`
	SavePath     string `json:"savepath"`
	Filename     string `json:"filename"`
	DownloadPath string `json:"downloadpath"`
	IsExec       bool   `json:"isExec"`
	LabelName    string `json:"labelName,omitempty"`
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

// MonitoredProcess is one process currently selected for application/
// process monitoring, normalized from the agent's string-typed fields
// (devicecontrol "get_monitor_app" action) into proper types.
//
// NOTE: not documented in AppHub-v2.3.0.md. Field shape is inferred by
// analogy with the sibling "get_software_info" action (same process-
// monitor subsystem, see ProcessInfo) - not empirically confirmed against
// a populated list, since no processes were being monitored on the device
// this was tested against (only the empty-list case is confirmed).
type MonitoredProcess struct {
	Command     string         `json:"command"`
	PID         string         `json:"pid"`
	CPUPercent  FlexibleNumber `json:"cpu_percent"`
	MemPercent  FlexibleNumber `json:"mem_percent"`
	Status      string         `json:"status"`
	Username    string         `json:"username"`
	IsMonitored bool           `json:"is_monitored"`
}

type rawMonitoredProcess struct {
	ThreadCmd        string         `json:"threadcmd"`
	ThreadPid        string         `json:"threadpid"`
	ThreadCPULoading FlexibleNumber `json:"threadcpuloading"`
	ThreadMemLoading FlexibleNumber `json:"threadmemloading"`
	ThreadStatus     string         `json:"threadstatus"`
	ThreadUsername   string         `json:"threadusername"`
	ThreadIsMonitor  string         `json:"threadismonitor"`
}

func (raw rawMonitoredProcess) normalize() MonitoredProcess {
	return MonitoredProcess{
		Command:     raw.ThreadCmd,
		PID:         raw.ThreadPid,
		CPUPercent:  raw.ThreadCPULoading,
		MemPercent:  raw.ThreadMemLoading,
		Status:      raw.ThreadStatus,
		Username:    raw.ThreadUsername,
		IsMonitored: raw.ThreadIsMonitor == "true",
	}
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

// BoardInfo is a device's board-level monitoring snapshot (devicecontrol
// "get_board_info" action), normalized from the raw agent response - which
// mixes numbers, numeric strings, and "N/A" placeholders - into consistent
// float64 metrics (nil when unavailable).
//
// NOTE: not documented in AppHub-v2.3.0.md. Units/semantics below are
// inferred from observed values, not a spec: CPU/memory/storage
// free/usage figures are percentages (memory's free+usage and storage's
// free+usage were each observed to sum to 100); *_total figures are
// absolute (GB, matching typical capacities); CPU temp's unit is assumed
// to be Celsius by convention but was never observed non-nil to confirm.
type BoardInfo struct {
	CPU     BoardCPUInfo     `json:"cpu"`
	Memory  BoardMemoryInfo  `json:"memory"`
	Storage BoardStorageInfo `json:"storage"`
	Battery BoardBatteryInfo `json:"battery"`
	NetIO   []BoardNetIOStat `json:"net_io"`
}

type BoardCPUInfo struct {
	UsagePercent FlexibleNumber `json:"usage_percent"`
	TempCelsius  FlexibleNumber `json:"temp_celsius"`
}

type BoardMemoryInfo struct {
	TotalGB      FlexibleNumber `json:"total_gb"`
	FreePercent  FlexibleNumber `json:"free_percent"`
	UsagePercent FlexibleNumber `json:"usage_percent"`
}

type BoardStorageInfo struct {
	TotalGB      FlexibleNumber `json:"total_gb"`
	FreePercent  FlexibleNumber `json:"free_percent"`
	UsagePercent FlexibleNumber `json:"usage_percent"`
}

type BoardBatteryInfo struct {
	AvailablePercent FlexibleNumber `json:"available_percent"`
}

// BoardNetIOStat is one network interface's I/O counters. "all" is a
// synthetic entry the agent reports summing every real interface.
type BoardNetIOStat struct {
	Name        string         `json:"name"`
	RxBytes     int64          `json:"rx_bytes"`
	TxBytes     int64          `json:"tx_bytes"`
	RxRateBps   int64          `json:"rx_rate_bps"`
	TxRateBps   int64          `json:"tx_rate_bps"`
	PktDropRate FlexibleNumber `json:"pkt_drop_rate"`
	PktErrRate  FlexibleNumber `json:"pkt_err_rate"`
}

// rawBoardInfo mirrors the device agent's actual (undocumented, loosely
// typed) get_board_info response shape before normalization.
type rawBoardInfo struct {
	CPU struct {
		Usage FlexibleNumber `json:"usage"`
		Temp  FlexibleNumber `json:"temp"`
	} `json:"cpu"`
	Memory struct {
		Total FlexibleNumber `json:"total"`
		Free  FlexibleNumber `json:"free"`
		Usage FlexibleNumber `json:"usage"`
	} `json:"memory"`
	Storage struct {
		Total FlexibleNumber `json:"total"`
		Free  FlexibleNumber `json:"free"`
		Usage FlexibleNumber `json:"usage"`
	} `json:"storage"`
	Battery struct {
		Available FlexibleNumber `json:"available"`
	} `json:"battery"`
	NetIO []struct {
		Name        string         `json:"name"`
		RxBytes     int64          `json:"rx_bytes"`
		TxBytes     int64          `json:"tx_bytes"`
		RxRate      int64          `json:"rx_rate"`
		TxRate      int64          `json:"tx_rate"`
		PktDropRate FlexibleNumber `json:"pkt_drop_rate"`
		PktErrRate  FlexibleNumber `json:"pkt_err_rate"`
	} `json:"net_io"`
}

func (raw rawBoardInfo) normalize() *BoardInfo {
	info := &BoardInfo{
		CPU: BoardCPUInfo{
			UsagePercent: raw.CPU.Usage,
			TempCelsius:  raw.CPU.Temp,
		},
		Memory: BoardMemoryInfo{
			TotalGB:      raw.Memory.Total,
			FreePercent:  raw.Memory.Free,
			UsagePercent: raw.Memory.Usage,
		},
		Storage: BoardStorageInfo{
			TotalGB:      raw.Storage.Total,
			FreePercent:  raw.Storage.Free,
			UsagePercent: raw.Storage.Usage,
		},
		Battery: BoardBatteryInfo{
			AvailablePercent: raw.Battery.Available,
		},
		NetIO: make([]BoardNetIOStat, 0, len(raw.NetIO)),
	}
	for _, n := range raw.NetIO {
		info.NetIO = append(info.NetIO, BoardNetIOStat{
			Name:        n.Name,
			RxBytes:     n.RxBytes,
			TxBytes:     n.TxBytes,
			RxRateBps:   n.RxRate,
			TxRateBps:   n.TxRate,
			PktDropRate: n.PktDropRate,
			PktErrRate:  n.PktErrRate,
		})
	}
	return info
}
