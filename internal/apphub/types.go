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

// QRCodeConfig is the server's client-onboarding config blob, as returned
// by GET /api/configmgr/getqrcodedata - the same data the web console
// encodes into its device-pairing QR code. Pasting Data directly into a
// device agent's own configuration UI connects it to this server, without
// manually entering the server address/credentials there.
type QRCodeConfig struct {
	ID   string `json:"id"`
	Data string `json:"data"`
}

type statusResponse struct {
	Status string `json:"status"`
	ErrMsg string `json:"errmsg"`
}

// MonitorDataPoint is a single CPU or memory monitoring sample for a
// device. Value is the usage percentage (0-100) for whichever type the
// query asked for (CPU or memory usage %, inferred from observed values
// matching BoardInfo's cpu.usage_percent/memory.usage_percent, not a
// spec) at Unix-millisecond timestamp Ts. Value is a FlexibleNumber (not
// a plain float64) because some servers/devices report it as a numeric
// string rather than a JSON number - confirmed via a real bug report.
type MonitorDataPoint struct {
	ID    int64          `json:"id"`
	Value FlexibleNumber `json:"value"`
	Ts    int64          `json:"ts"`
}

type setGroupsRequest struct {
	Endpoint   string   `json:"endpoint"`
	GroupNames []string `json:"groupnames"`
}

type execActionRequest struct {
	ID    int    `json:"id"`
	Value string `json:"value"`
}

// monitorListValue is devicecontrol resource 27601's write-value shape
// for adding/removing a specific process or Docker container to/from
// application/process or Docker monitoring - confirmed against a live
// server via a real packet capture. Unlike most other devicecontrol
// writes, "pkgname" here is a sibling key of "pluginname"/"actions" (not
// nested inside the "actions" string) - each item is the exact object
// from "get_software_info" (for processes) or "get_docker_info" (for
// containers), resent verbatim. "actions"
// ("set_isProcessMonitor"/"set_isDockerMonitor": true/false) is what
// actually controls add vs remove - confirmed live that the item's own
// "threadismonitor"/"ismonitor" field is NOT what determines that (an
// item resent with either value is still added when the outer flag is
// true, and removed when it's false).
type monitorListValue struct {
	PluginName string            `json:"pluginname"`
	Actions    string            `json:"actions"`
	PkgName    []json.RawMessage `json:"pkgname"`
}

// dockerContainerControlValue is the write-value shape for per-container
// Docker runtime control on one device via /api/devicemgr/{agentId}/60002/0/{id},
// where id maps to start/stop/restart. pkgname carries the container name;
// version and taskname are fixed placeholders per server behavior.
type dockerContainerControlValue struct {
	PkgName string `json:"pkgname"`
	Version string `json:"version"`
	TaskName string `json:"taskname"`
	Target  string `json:"target"`
}

// execBatchRequest is the body for the batch-execute devicecontrol
// resource (PUT /api/plugins/batchExec/40008/0/27601), used to
// reboot/shutdown/set-volume/set-backlight an explicit list of devices at
// once. Unlike the single-device execActionRequest, there's no "id" field
// here - the resource id (27601) is only in the URL path. clientIds must
// be resolved client-side first (e.g. from a group+label via
// ListDevicesByCondition) - this endpoint has no server-side group
// lookup of its own.
type execBatchRequest struct {
	ClientIDs []string `json:"clientIds"`
	Value     string   `json:"value"`
}

// batchScreenshotRequest is the body for PUT /api/devicemgr/batchExec -
// not documented in AppHub-v2.3.0.md (only the single-device screenshot
// trigger, PUT /api/devicemgr/{agentId}/40005/0/27405, is), confirmed
// against a live server via a real packet capture. It tells every device
// in ClientIDs to start pushing periodic screenshots to the host/interval
// in Value. Unlike the other batchExec endpoint used elsewhere in this
// client (PUT /api/plugins/batchExec/40008/0/27601, body
// {"clientIds":[...],"value":...} with ClientIDs as a plain JSON array),
// this is a different endpoint (/api/devicemgr/batchExec, no "/plugins")
// with a different body shape: ClientIDs here is itself JSON-encoded as
// a string, and there's a "target" field carrying the LwM2M
// object/resource path (confirmed against a live server).
type batchScreenshotRequest struct {
	ClientIDs string `json:"clientIds"`
	Target    string `json:"target"`
	Value     string `json:"value"`
}

// screenshotStreamValue is Value's shape for batchScreenshotRequest.
// Interval is a string (e.g. "5000"), not a JSON number, confirmed
// against a live server via a real packet capture.
type screenshotStreamValue struct {
	Host     string `json:"host"`
	Interval string `json:"interval"`
}

// agentLogTriggerValue is execActionRequest's Value shape for triggering
// an agent log upload (see DownloadAgentLog) - tells the device where to
// push its log (Address, this server's own base URL) and what ActionID to
// tag the upload with, confirmed against a live server via a real packet
// capture.
type agentLogTriggerValue struct {
	Address  string `json:"address"`
	ActionID string `json:"actionId"`
}

// agentLogStatusResponse is GET /api/devicemgr/logStatus's response shape
// - Result is "success" once the device's push (started by
// agentLogTriggerValue's ActionID) has arrived, confirmed against a live
// server via a real packet capture.
type agentLogStatusResponse struct {
	Result   string `json:"result"`
	ActionID string `json:"actionId"`
}

// agentFileTriggerValue is execActionRequest's Value shape for triggering
// a push of one specific file (see DownloadDeviceFile) - like
// agentLogTriggerValue, but scoped to an absolute path (LogFile, a
// single-element slice - confirmed against a live server that it's an
// array even for one file) under an AppName label that ties the trigger
// to the download's filename/value query parameters. Confirmed against a
// live server via a real packet capture.
type agentFileTriggerValue struct {
	Address  string   `json:"address"`
	ActionID string   `json:"actionId"`
	AppName  string   `json:"appname"`
	LogFile  []string `json:"logFile"`
}

type actionValue struct {
	PluginName string `json:"pluginname"`
	Actions    string `json:"actions"`
}

// scheduledActionValue is the write-value shape for a scheduled (rather
// than immediate) devicecontrol reboot/shutdown - the same
// {"pluginname","actions"} envelope as actionValue, with "time" (the
// first/anchor execution, Unix milliseconds) and an optional "cycle"
// (repeat interval in days - omitted entirely for a one-time schedule)
// added. Documented for the batch/group variant ("PUT schedule
// shutdown-批量设置定时关机", /api/plugins/batchschedule/{groupname}/
// 40008/0/27601 in AppHub-v2.3.0.md, whose own example uses "cycle":7 for
// a weekly schedule) but not for a single device - the single-device
// version used here was confirmed via a real packet capture, sharing the
// same value shape (minus "taskname", which is batch/group-only).
type scheduledActionValue struct {
	PluginName string `json:"pluginname"`
	Actions    string `json:"actions"`
	Time       int64  `json:"time"`
	Cycle      int    `json:"cycle,omitempty"`
}

// ScheduledPower is a device's currently configured reboot or shutdown
// schedule, as reported by the "get_reboot_plan"/"get_shutdown_plan"
// devicecontrol actions - confirmed against a live server: the action
// returns an empty object when nothing is scheduled (TimeMs stays 0),
// and is populated after a real "device schedule reboot/shutdown" call,
// then empty again after --cancel - including a cancel that didn't reuse
// the original time/cycle values, confirming --cancel doesn't require an
// exact match.
type ScheduledPower struct {
	TimeMs int64          `json:"timeMs"`    // Unix ms; 0 if nothing is currently scheduled
	Cycle  FlexibleNumber `json:"cycleDays"` // repeat interval in days; nil if one-time or nothing scheduled
}

// rawRebootPlan/rawShutdownPlan mirror get_reboot_plan/get_shutdown_plan's
// actual field names, which differ (rebootTime/rebootCycle vs
// shutdownTime/shutdownCycle) - normalized into the common ScheduledPower
// shape above. Cycle is a FlexibleNumber because it's reported as an
// empty string when unset and a real JSON number when set (confirmed
// against a live server: {"rebootCycle":"","rebootTime":...} for a
// one-time schedule vs {"rebootCycle":7,"rebootTime":...} for a weekly
// one).
type rawRebootPlan struct {
	RebootTime  int64          `json:"rebootTime"`
	RebootCycle FlexibleNumber `json:"rebootCycle"`
}

type rawShutdownPlan struct {
	ShutdownTime  int64          `json:"shutdownTime"`
	ShutdownCycle FlexibleNumber `json:"shutdownCycle"`
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

// MonitorDataReportStatus reports whether CPU/memory historical-data
// collection/reporting is currently enabled on a device - this is what
// feeds "device monitor"'s samples. Confirmed against a live server via
// the "get_monitor_data_report" devicecontrol action, which also reports
// systemd log monitoring in the same response (see
// GetSystemLogMonitorEnabled for that - not exposed here since it's a
// separate feature/domain, "log system", even though it shares this one
// underlying action).
type MonitorDataReportStatus struct {
	CPU    bool `json:"cpu"`
	Memory bool `json:"memory"`
}

// rawMonitorDataReport mirrors "get_monitor_data_report"'s actual
// response shape, confirmed against a live server:
// {"cpu_monitor":false,"memory_monitor":false,"systemd_log_monitor":true}.
type rawMonitorDataReport struct {
	CPUMonitor        bool `json:"cpu_monitor"`
	MemoryMonitor     bool `json:"memory_monitor"`
	SystemdLogMonitor bool `json:"systemd_log_monitor"`
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

// MonitoredProcess is one process currently selected for application/
// process monitoring, normalized from the agent's string-typed fields
// (devicecontrol "get_monitor_app" action) into proper types.
//
// NOTE: not documented in AppHub-v2.3.0.md. Field shape confirmed against
// a live server's populated list (via SetProcessMonitored adding a real
// process to the list, then reading it back here).
type MonitoredProcess struct {
	Command     string         `json:"command"`
	PID         FlexibleNumber `json:"pid"`
	CPUPercent  FlexibleNumber `json:"cpu_percent"`
	MemPercent  FlexibleNumber `json:"mem_percent"`
	Status      string         `json:"status"`
	Username    string         `json:"username"`
	IsMonitored bool           `json:"is_monitored"`
}

type rawMonitoredProcess struct {
	ThreadCmd        string         `json:"threadcmd"`
	ThreadPid        FlexibleNumber `json:"threadpid"`
	ThreadCPULoading FlexibleNumber `json:"threadcpuloading"`
	ThreadMemLoading FlexibleNumber `json:"threadmemloading"`
	ThreadStatus     string         `json:"threadstatus"`
	ThreadUsername   string         `json:"threadusername"`
	ThreadIsMonitor  string         `json:"threadismonitor"`
}

// normalize builds a MonitoredProcess. IsMonitored is always true here,
// not raw.ThreadIsMonitor == "true" as an earlier version assumed -
// confirmed against a live server that "get_monitor_app" (what feeds
// this) always reports "threadismonitor":"" (empty) for every item it
// returns, monitored or not; being present in this list at all is what
// "monitored" actually means for this action.
func (raw rawMonitoredProcess) normalize() MonitoredProcess {
	return MonitoredProcess{
		Command:     raw.ThreadCmd,
		PID:         raw.ThreadPid,
		CPUPercent:  raw.ThreadCPULoading,
		MemPercent:  raw.ThreadMemLoading,
		Status:      raw.ThreadStatus,
		Username:    raw.ThreadUsername,
		IsMonitored: true,
	}
}

// ProcessInfo is a running process/thread on a Windows or Linux device, as
// reported by the devicecontrol "get_software_info" action, normalized
// the same way as MonitoredProcess. Not documented in AppHub-v2.3.0.md;
// confirmed against a live server, including CommandLine/Exe, which
// aren't in the older doc example this session found for the sibling
// batch-monitor-set endpoint.
type ProcessInfo struct {
	Command     string         `json:"command"`
	CommandLine string         `json:"command_line"`
	Exe         string         `json:"exe"`
	PID         FlexibleNumber `json:"pid"`
	CPUPercent  FlexibleNumber `json:"cpu_percent"`
	MemPercent  FlexibleNumber `json:"mem_percent"`
	Status      string         `json:"status"`
	Username    string         `json:"username"`
	IsMonitored bool           `json:"is_monitored"`
}

type rawProcessInfo struct {
	ThreadCmd        string         `json:"threadcmd"`
	ThreadCmdline    string         `json:"threadcmdline"`
	ThreadExe        string         `json:"threadexe"`
	ThreadPid        FlexibleNumber `json:"threadpid"`
	ThreadCPULoading FlexibleNumber `json:"threadcpuloading"`
	ThreadMemLoading FlexibleNumber `json:"threadmemloading"`
	ThreadStatus     string         `json:"threadstatus"`
	ThreadUsername   string         `json:"threadusername"`
	ThreadIsMonitor  string         `json:"threadismonitor"`
}

func (raw rawProcessInfo) normalize() ProcessInfo {
	return ProcessInfo{
		Command:     raw.ThreadCmd,
		CommandLine: raw.ThreadCmdline,
		Exe:         raw.ThreadExe,
		PID:         raw.ThreadPid,
		CPUPercent:  raw.ThreadCPULoading,
		MemPercent:  raw.ThreadMemLoading,
		Status:      raw.ThreadStatus,
		Username:    raw.ThreadUsername,
		IsMonitored: raw.ThreadIsMonitor == "true",
	}
}

// DockerContainer is a single Docker container on a device, as reported
// by the devicecontrol "get_docker_info" action - confirmed against a
// live server via a real packet capture. The action returns every
// container regardless of monitor status; GetDockerInfo filters to the
// ones with IsMonitor == "true".
//
// NOTE: not documented in AppHub-v2.3.0.md. An earlier implementation
// instead read GET .../60001/0/27300, assuming a response grouped by
// compose project (`{"code":...,"data":[{"tag","version","type",
// "control","docker":[...]}]}`) - that endpoint returned an empty result
// on a live server with real containers deployed, so this was switched
// to the working devicecontrol action/response shape.
type DockerContainer struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Image     string `json:"image"`
	Command   string `json:"command"`
	Ports     string `json:"ports"`
	State     string `json:"state"`
	Compose   string `json:"compose"`
	Created   string `json:"created"`
	IsMonitor string `json:"ismonitor"`
}

// OperationRecord is one audit-log entry of an admin action (device
// control, config change, etc): who (Username) did what (Event, the raw
// action payload sent to the device) to what (Target, typically
// "<endpoint>(<devName>)"), when (Ts, Unix ms). Field names/shapes are
// confirmed against a live server's actual response, not just
// OperationRecordServlet.java's assumed shape. Event is FlexibleJSON, not
// a plain string - the server encodes it inconsistently across entries:
// sometimes a raw JSON object (e.g. {"shutdown":true}), sometimes a JSON
// string containing escaped JSON text, and sometimes a plain path-like
// string - confirmed via a live server returning all three shapes across
// one page of results (a real decode error: "cannot unmarshal object into
// Go struct field OperationRecord.data.event of type string").
type OperationRecord struct {
	ID       int64        `json:"orid"`
	Type     string       `json:"type"`
	Event    FlexibleJSON `json:"event"`
	Target   string       `json:"target"`
	Username string       `json:"username"`
	Ts       int64        `json:"ts"`
}

// operationRecordsPageResponse mirrors /api/records/byPage's actual
// response: {"count":<n>,"data":[...]} - data is a plain JSON array of
// OperationRecord, not a double-JSON-encoded string as originally assumed.
type operationRecordsPageResponse struct {
	Count int64             `json:"count"`
	Data  []OperationRecord `json:"data"`
}

// SystemLogEntry is one row from a device's collected system log (fed by
// SetSystemLogMonitorEnabled turning collection on for that device).
// Level and Pid are FlexibleNumber (not plain strings) because the server
// reports them as numeric strings (e.g. "level":"5", "pid":"16662") -
// confirmed against a live server via a real packet capture.
type SystemLogEntry struct {
	ID       int64          `json:"id"`
	DeviceID string         `json:"deviceId"`
	DevName  string         `json:"devname"`
	Content  string         `json:"content"`
	OS       string         `json:"os"`
	Level    FlexibleNumber `json:"level"`
	Mode     string         `json:"mode"`
	Name     string         `json:"name"`
	Pid      FlexibleNumber `json:"pid"`
	EventID  string         `json:"eventId"`
	Ts       int64          `json:"ts"`
}

// systemLogsPageResponse mirrors /api/systemLog/byPage's actual response:
// {"count":<n>,"data":[...]} - confirmed against a live server, same
// wrapper shape as operationRecordsPageResponse.
type systemLogsPageResponse struct {
	Count int64            `json:"count"`
	Data  []SystemLogEntry `json:"data"`
}

type deleteByDeviceIDRequest struct {
	DeviceID string `json:"deviceId"`
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
