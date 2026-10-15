package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

// parseOnOff parses an "on"/"off" (or "true"/"false") toggle argument.
func parseOnOff(s string) (bool, error) {
	switch s {
	case "on", "true":
		return true, nil
	case "off", "false":
		return false, nil
	default:
		return false, fmt.Errorf("expected \"on\" or \"off\", got %q", s)
	}
}

// percentThresholdMetrics is the subset of settable threshold metrics
// expressed as a 0-100 percentage. The server itself enforces none of
// this - confirmed live that it accepts and stores --warn 150 --error 200
// verbatim for cpu_usage - so the CLI validates it instead.
var percentThresholdMetrics = map[string]bool{
	"cpu_usage":         true,
	"memory_usage":      true,
	"storage_usage":     true,
	"battery_available": true,
}

// descendingThresholdMetrics is the subset of settable threshold metrics
// where a LOWER value is worse, so warn must be greater than error -
// opposite of the usual "higher is worse" direction (warn < error) that
// applies to every other metric. Currently just battery_available
// (remaining battery capacity: e.g. warn at 20%, error at 10%).
var descendingThresholdMetrics = map[string]bool{
	"battery_available": true,
}

// validateThresholdLevel checks a single --warn or --error value against
// metric's natural range. Percentage metrics must be within 0-100;
// everything else just can't be negative.
func validateThresholdLevel(metric string, level float64) error {
	if level < 0 {
		return fmt.Errorf("%v must not be negative", level)
	}
	if percentThresholdMetrics[metric] && level > 100 {
		return fmt.Errorf("%v is out of range for %q - percentage metrics must be within 0-100", level, metric)
	}
	return nil
}

// validateThresholdOrder checks that warn/error are ordered correctly for
// metric's direction: warn < error for every metric except
// descendingThresholdMetrics, where it's reversed (warn > error).
func validateThresholdOrder(metric string, warn, errorLevel float64) error {
	if descendingThresholdMetrics[metric] {
		if warn <= errorLevel {
			return fmt.Errorf("warn (%v) must be greater than error (%v) for %q - a lower value is worse for this metric, so error should trip at a lower value than warn", warn, errorLevel, metric)
		}
		return nil
	}
	if warn >= errorLevel {
		return fmt.Errorf("warn (%v) must be less than error (%v) for %q - error should trip at a more severe (higher) value than warn", warn, errorLevel, metric)
	}
	return nil
}

var deviceBoardInfoCmd = &cobra.Command{
	Use:   "get-monitor <endpoint-or-name>",
	Short: "Get a device's live CPU/memory/storage/battery/network snapshot",
	Long: `Get a live board-level monitoring snapshot: CPU usage/temperature, memory
usage, storage usage, battery level, and per-interface network I/O.

获取设备当前的实时监控快照：CPU使用率/温度、内存使用率、存储使用率、
电池电量、各网卡的收发流量。

Not documented in AppHub-v2.3.0.md (found by inspecting real devicecontrol
traffic). The response is normalized: the raw agent payload mixes numbers,
numeric strings, and "N/A" placeholders inconsistently, so this CLI parses
everything into consistent float64 fields (null when unavailable).

未在 AppHub-v2.3.0.md 中记录（通过抓包分析 devicecontrol 流量得到）。
返回值已做了归一化处理：原始数据里数字、数字字符串、"N/A" 占位符混杂
不一致，这里统一转换成 float64（无法获取时为 null），字段命名也做了
可读化处理（如 usage_percent、total_gb）。

Example 示例: eim-skill device get-monitor "08:00:27:33:EC:2E"

Example output 输出示例 (real, trimmed 真实返回，已截断):
  {
    "status": 200, "msg": "",
    "data": {
      "cpu": {"usage_percent": 30.2, "temp_celsius": null},
      "memory": {"total_gb": 10.39, "free_percent": 35.1, "usage_percent": 64.9},
      "storage": {"total_gb": 220.09, "free_percent": 39, "usage_percent": 61},
      "battery": {"available_percent": null},
      "net_io": [{"name": "enp0s3", "rx_bytes": 301688868, "tx_bytes": 351754097}]
    }
  }`,
	Args: requireArgs(cobra.ExactArgs(1)),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		info, err := client.GetBoardInfo(endpoint)
		if err != nil {
			return err
		}
		printJSON(info)
		return nil
	},
}

// threshold

var deviceThresholdCmd = &cobra.Command{
	Use:   "threshold",
	Short: "Get/set CPU, memory, storage, network, and battery alert thresholds",
	Long: `Configure and inspect per-metric warn/error thresholds. This is how
DeviceOn-EIM's "monitorAlert" feature is set up server-side - the
device itself evaluates these thresholds and reports its resulting
warning/error status, which shows up as deviceStatus/deviceStatusText on
"device get"/"device list --alarm" and in "device health".

配置和查看各监控指标的警告/错误阈值。这是 DeviceOn-EIM
"monitorAlert" 功能在服务端的配置入口——设备本身会依据这些阈值判定
自身状态，判定结果体现在 "device get"/"device list --alarm" 的
deviceStatus/deviceStatusText 字段，以及 "device health" 里。`,
}

var deviceThresholdGetCmd = &cobra.Command{
	Use:   "get <endpoint-or-name>",
	Short: "Get the device's current threshold configuration",
	Long: `Get every metric's current on/off state and warn/error levels.

获取设备当前每一项监控指标的开关状态和警告/错误阈值。

Example 示例: eim-skill device threshold get "08:00:27:33:EC:2E"

Example output 输出示例 (real 真实返回):
  {
    "status": 200, "msg": "",
    "data": [
      {"metric": "cpu_temp", "enabled": false, "warn": 70, "error": 95},
      {"metric": "cpu_usage", "enabled": true, "warn": 70, "error": 90},
      {"metric": "memory_usage", "enabled": true, "warn": 80, "error": 95},
      {"metric": "storage_usage", "enabled": true, "warn": 80, "error": 90},
      {"metric": "battery_available", "enabled": false, "warn": 90, "error": 95}
    ]
  }`,
	Args: requireArgs(cobra.ExactArgs(1)),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		settings, err := client.GetThresholds(endpoint)
		if err != nil {
			return err
		}
		printJSON(settings)
		return nil
	},
}

var deviceThresholdSetCmd = &cobra.Command{
	Use:   "set <endpoint-or-name> <metric> <on|off>",
	Short: "Enable/disable and configure one threshold metric",
	Long: `Enable or disable alerting for one metric, optionally changing its
warn/error levels via --warn/--error. If either flag is omitted, this
command first fetches the metric's current warn/error levels (via
"device threshold get") and resends those unchanged, rather than
overwriting them with 0 - the underlying set_threshold devicecontrol
action always requires both warn and error in the request, and blindly
sending 0 for an omitted flag would silently zero out an existing
threshold (confirmed via a real bug report: toggling on/off without
--warn/--error reset the threshold to 0/0).

启用或关闭某一项监控指标的告警，可通过 --warn/--error 修改对应的警告/
错误阈值。如果省略了其中任意一个，本命令会先查询该指标当前的警告/
错误阈值（相当于先执行一次 "device threshold get"），把原值原样带上，
而不是用0覆盖——底层 set_threshold 设备控制动作每次请求都必须同时带上
warn和error，如果省略的参数直接按0发送，会静默把已有的阈值清零（已根据
真实反馈确认此问题：只切换开关而不带 --warn/--error 会把阈值重置为
0/0）。

Supported metrics 支持的指标:
  cpu_temp              CPU 温度
  cpu_usage              CPU 使用率
  memory_usage          内存使用率
  storage_usage          存储使用率
  battery_available      电池剩余电量
  network_upload_speed    网络上传速度
  network_download_speed  网络下载速度

--warn/--error are validated client-side, since the device itself accepts
anything (confirmed live: it silently stores warn=150/error=200, or even
negative values, verbatim): percentage metrics (cpu_usage, memory_usage,
storage_usage, battery_available) must be within 0-100, and neither may be
negative. They must also be ordered correctly for the metric's direction:
warn < error for every metric except battery_available, where lower is
worse, so it's reversed (warn > error, e.g. warn=20, error=10). These
checks only run against whichever value(s) you actually pass - a pure
on/off toggle that leaves both flags unset skips them entirely.

--warn/--error 会在客户端做校验，因为设备本身不做任何检查（已实测：
warn=150、error=200，甚至负数都会被原样静默保存）：百分比类指标
（cpu_usage、memory_usage、storage_usage、battery_available）必须在
0-100 之间，且都不能为负数。两者的大小关系也必须符合该指标的方向：
除 battery_available 外都要求 warn < error；battery_available 因为数值
越低越糟糕，所以反过来要求 warn > error（如 warn=20、error=10）。这些
检查只针对你实际传入的参数——如果 --warn/--error 都不填，只是单纯切换
开关，则完全不会触发这些检查。

Example 示例: eim-skill device threshold set "146打包机" cpu_usage on --warn 70 --error 90
       eim-skill device threshold set "146打包机" cpu_temp off   # just toggles off, keeps existing warn/error

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED"}}`,
	Args: requireArgs(cobra.ExactArgs(3)),
	RunE: func(cmd *cobra.Command, args []string) error {
		metric := args[1]
		enabled, err := parseOnOff(args[2])
		if err != nil {
			return err
		}
		warn, err := cmd.Flags().GetFloat64("warn")
		if err != nil {
			return err
		}
		errorLevel, err := cmd.Flags().GetFloat64("error")
		if err != nil {
			return err
		}
		warnGiven := cmd.Flags().Changed("warn")
		errorGiven := cmd.Flags().Changed("error")

		if warnGiven {
			if err := validateThresholdLevel(metric, warn); err != nil {
				return fmt.Errorf("invalid --warn: %w", err)
			}
		}
		if errorGiven {
			if err := validateThresholdLevel(metric, errorLevel); err != nil {
				return fmt.Errorf("invalid --error: %w", err)
			}
		}

		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}

		if !warnGiven || !errorGiven {
			current, err := client.GetThresholds(endpoint)
			if err != nil {
				return err
			}
			for _, s := range current {
				if s.Metric != metric {
					continue
				}
				if !warnGiven {
					warn = s.Warn
				}
				if !errorGiven {
					errorLevel = s.Error
				}
				break
			}
		}

		if warnGiven || errorGiven {
			if err := validateThresholdOrder(metric, warn, errorLevel); err != nil {
				return fmt.Errorf("invalid --warn/--error: %w", err)
			}
		}

		if err := client.SetThreshold(endpoint, metric, enabled, warn, errorLevel); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

// alert history

var deviceAlertCmd = &cobra.Command{
	Use:   "alert",
	Short: "Query a device's recorded alert history (hardware/software/peripheral)",
	Long: `Query the alert-history records DeviceOn-EIM has stored for a device -
distinct from "device threshold get/set", which configures the
thresholds that generate these alerts, not the alert records themselves.

查询 DeviceOn-EIM 为某台设备保存的告警历史记录——和 "device threshold
get/set"（配置产生告警的阈值，而非告警记录本身）是两回事。`,
}

var deviceAlertHistoryCmd = &cobra.Command{
	Use:   "history <endpoint-or-name>",
	Short: "List a device's recorded alert-history entries",
	Long: `List one page of a device's recorded alert-history entries. --type
narrows to one alert category (hardware/software/peripheral, going by a
real request URL - the full accepted set isn't confirmed) and --level
narrows to one severity level (server-defined); both are passed through
to the server as-is, and either may be left empty to not filter on it.

列出设备的告警历史记录（分页）。--type 可按告警类别过滤（hardware/
software/peripheral——来自一次真实请求的URL，完整取值范围未经确认），
--level 可按告警级别过滤（具体取值由服务器定义）；两者都原样透传给
服务器，留空即不按该条件过滤。

Restrict the time range with either --last (a recent relative period,
e.g. "30m", "12h", "3d") or --begin/--end (an absolute range, RFC3339 or
"YYYY-MM-DD HH:MM" - either or both may be given; mutually exclusive with
--last).

可以用 --last（最近一段相对时间，如 "30m"/"12h"/"3d"）或者
--begin/--end（绝对时间区间，RFC3339 或 "YYYY-MM-DD HH:MM"——可以只填
一个也可以两个都填；不能和 --last 同时使用）来限定时间范围。

CAVEAT: this endpoint (GET /v1/alertHistory/byPage/condition, on the
apphub-ithings server) is not documented in AppHub-v2.3.0.md; the request
shape (query parameter names, this help text's flags) comes from a real
request URL, but no populated response was available during development
to confirm the returned entries' exact field names (the only reachable
test server rejected every credential tried), so each entry is passed
through as raw JSON rather than reshaped into named fields - inspect the
raw output once against your own server and treat this note as stale if
it no longer matches.

注意：这个接口（GET /v1/alertHistory/byPage/condition，位于
apphub-ithings 服务上）未在 AppHub-v2.3.0.md 中记录；请求参数名来自一次
真实请求的URL，但开发时未能用真实有数据的返回结果验证每条记录的具体
字段名（当时唯一能连通的测试服务器没有一个可用凭据能通过鉴权），因此
这里直接透传原始JSON，不做重新整形——请在你自己的服务器上实际跑一次，
用真实返回核对字段，如与本说明不符以实际为准。

Example 示例: eim-skill device alert history "146打包机" --type hardware --last 7d
       eim-skill device alert history "28:16:AD:F2:56:5C" --type software --level 2 --page 1 --limit 10`,
	Args: requireArgs(cobra.ExactArgs(1)),
	RunE: func(cmd *cobra.Command, args []string) error {
		page, _ := cmd.Flags().GetInt("page")
		limit, _ := cmd.Flags().GetInt("limit")
		level, _ := cmd.Flags().GetString("level")
		alertType, _ := cmd.Flags().GetString("type")
		begin, end, err := resolveTimeWindow(cmd)
		if err != nil {
			return err
		}

		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		ithingsClient, err := newIthingsClient(client)
		if err != nil {
			return err
		}

		alerts, total, err := ithingsClient.ListAlertHistory(endpoint, page, limit, level, alertType, begin, end)
		if err != nil {
			return err
		}
		printJSON(map[string]interface{}{
			"total":  total,
			"alerts": alerts,
		})
		return nil
	},
}

// process monitoring

var deviceProcessCmd = &cobra.Command{
	Use:   "process",
	Short: "List a device's processes, and add/remove ones selected for monitoring",
	Long: `List every running process on a device ("list"), just the ones currently
selected for application/process monitoring ("monitored"), or add/remove
a specific process (by pid) to/from that monitored list ("set-monitor").

列出设备上所有正在运行的进程（"list"），只列出当前被选中监控的进程
（"monitored"），或者把某个进程（按pid）加入/移出监控列表
（"set-monitor"）。`,
}

var deviceProcessListCmd = &cobra.Command{
	Use:   "list <endpoint-or-name>",
	Short: "List every process running on a device",
	Long: `List every running process/thread on a device (not just the monitored
subset - see "device process monitored" for that), with normalized
numeric CPU/memory percentages, a real is_monitored boolean, and pid as a
number.

列出设备上所有正在运行的进程/线程（不只是被监控的那部分——那个见
"device process monitored"），CPU/内存占用已转成数字百分比，
is_monitored 是真正的布尔值，pid 也已归一化成数字。

Not documented in AppHub-v2.3.0.md; uses the "get_software_info"
devicecontrol action, confirmed against a live server.

未在 AppHub-v2.3.0.md 中记录；使用 "get_software_info" 设备控制动作，
已用真实服务器验证过。

Example 示例: eim-skill device process list "08:00:27:33:EC:2E"

Example output 输出示例 (real, trimmed 真实返回，已截断):
  {
    "status": 200, "msg": "",
    "data": [
      {
        "command": "irqbalance",
        "command_line": "/usr/sbin/irqbalance --foreground",
        "exe": "/usr/sbin/irqbalance",
        "pid": 544,
        "cpu_percent": 0.02,
        "mem_percent": 0.04,
        "status": "S",
        "username": "root",
        "is_monitored": false
      }
    ]
  }`,
	Args: requireArgs(cobra.ExactArgs(1)),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		procs, err := client.ListProcesses(endpoint)
		if err != nil {
			return err
		}
		printJSON(procs)
		return nil
	},
}

var deviceProcessMonitoredCmd = &cobra.Command{
	Use:   "monitored <endpoint-or-name>",
	Short: "List the processes currently selected for application/process monitoring",
	Long: `List the processes currently selected for monitoring, with normalized
numeric CPU/memory percentages, a real is_monitored boolean, and pid as a
number (the agent reports it as a numeric string, e.g. "pid":"639" -
confirmed against a live server).

列出当前被选中监控的进程，CPU/内存占用已转成数字百分比，is_monitored
是真正的布尔值，pid 也已归一化成数字（agent 返回的是数字字符串，如
"pid":"639"——已用真实服务器验证过）。

Example 示例: eim-skill device process monitored "08:00:27:33:EC:2E"

Example output 输出示例 (empty when nothing is selected 未选中任何进程
时返回空数组): {"status": 200, "msg": "", "data": []}`,
	Args: requireArgs(cobra.ExactArgs(1)),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		procs, err := client.GetMonitoredProcesses(endpoint)
		if err != nil {
			return err
		}
		printJSON(procs)
		return nil
	},
}

var deviceProcessSetMonitorCmd = &cobra.Command{
	Use:   "set-monitor <endpoint-or-name> <pid> <on|off>",
	Short: "Add or remove a specific process from application/process monitoring",
	Long: `Add (on) or remove (off) one process, identified by pid, to/from the
device's monitored-process list (see "device process monitored"). Just
supply the pid - this looks up the matching process in "device process
list" itself and resends its full details, since the underlying device
API requires that (not just a bare pid).

把某个进程（用pid指定）加入（on）或移出（off）设备的监控进程列表
（见 "device process monitored"）。只需要提供pid——本命令会自动从
"device process list" 里查到这个进程的完整信息并重新提交，因为设备
底层接口要求的是完整信息，而不是只给一个pid。

Not documented in AppHub-v2.3.0.md for a single device - only a
batch/group variant is ("PUT 批量设备APP/进程监控设置-加入监控列表",
/api/plugins/batchsetsoftwaremonitor/{groupname}/40008/0/27601), which
was found to reliably fail with an empty-bodied HTTP 500 on a live
server (the same failure pattern as another documented-but-broken batch
endpoint found earlier). This single-device version was confirmed
against a live server via a real packet capture and cross-verified
end-to-end: adding a pid makes it appear in "device process monitored",
and removing it makes it disappear again.

未在 AppHub-v2.3.0.md 中记录单设备版本——文档里只记录了批量/分组版本
（"PUT 批量设备APP/进程监控设置-加入监控列表"，/api/plugins/
batchsetsoftwaremonitor/{groupname}/40008/0/27601），实测无论用哪个
分组都稳定返回空身体的HTTP 500（和此前发现的另一个文档记录但已损坏的
批量接口是同样的错误模式）。这个单设备版本已通过真实抓包在真实服务器
上验证过，并做了端到端交叉验证：加入一个pid后它会出现在
"device process monitored" 里，移出后又会消失。

Example 示例: eim-skill device process set-monitor "146打包机" 544 on

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED"}}`,
	Args: requireArgs(cobra.ExactArgs(3)),
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("%q is not an integer pid: %w", args[1], err)
		}
		enabled, err := parseOnOff(args[2])
		if err != nil {
			return err
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		if err := client.SetProcessMonitored(endpoint, pid, enabled); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

// docker monitoring

var deviceDockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "List a device's Docker containers, and add/remove ones selected for monitoring",
	Long: `List every Docker container on a device ("list"), just the ones
currently selected for monitoring ("monitored"), or add/remove a specific
container (by name) to/from that monitored list ("set-monitor"). Also
supports runtime control for one container: start, stop, and restart.

列出设备上所有 Docker 容器（"list"），只列出当前被选中监控的容器
（"monitored"），或者把某个容器（按名称）加入/移出监控列表
（"set-monitor"）。另外也支持单个容器的运行控制：start、stop、restart。`,
}

var deviceDockerListCmd = &cobra.Command{
	Use:   "list <endpoint-or-name>",
	Short: "List every Docker container on a device",
	Long: `List every Docker container on a device, regardless of monitor status -
see "device docker monitored" for just the monitored subset.

列出设备上所有 Docker 容器，不论是否被监控——只看被监控的那部分见
"device docker monitored"。

Not documented in AppHub-v2.3.0.md; uses the "get_docker_info"
devicecontrol action, confirmed against a live server.

未在 AppHub-v2.3.0.md 中记录；使用 "get_docker_info" 设备控制动作，
已用真实服务器验证过。

Example 示例: eim-skill device docker list "08:00:27:33:EC:2E"

Example output 输出示例 (real, trimmed 真实返回，已截断):
  {
    "status": 200, "msg": "",
    "data": [
      {
        "id": "fadf722f405b...", "name": "apphub",
        "image": "172.21.73.109/apphub/apphub-manager:v6.9.288",
        "command": "/root/adv/run.sh", "ports": " 30001:30001 8080:8080 8081:8081",
        "state": "running", "compose": "edgemindmanagermain",
        "created": "2026-07-10 16:01:50", "ismonitor": "false"
      }
    ]
  }`,
	Args: requireArgs(cobra.ExactArgs(1)),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		containers, err := client.ListDockerContainers(endpoint)
		if err != nil {
			return err
		}
		printJSON(containers)
		return nil
	},
}

var deviceDockerMonitoredCmd = &cobra.Command{
	Use:   "monitored <endpoint-or-name>",
	Short: "Get the Docker containers on a device currently selected for monitoring",
	Long: `List only the Docker containers on the device currently selected for
monitoring (ismonitor == true), not every deployed container.

只列出设备上当前被选中监控的 Docker 容器（ismonitor == true），
而不是所有已部署的容器。

Not documented in AppHub-v2.3.0.md; uses the "get_docker_info"
devicecontrol action, confirmed against a live server via a real packet
capture - the action itself returns every container regardless of
monitor status, so this command filters to the monitored subset
client-side. An earlier version read from a different, undocumented
resource (LwM2M .../60001/0/27300) assuming a response grouped by compose
project, which returned an empty result on a live server with real
containers deployed - fixed via the real packet capture.

未在 AppHub-v2.3.0.md 中记录；使用 "get_docker_info" 设备控制动作，
已通过真实抓包在真实服务器上验证过——这个动作本身会返回所有容器
（不论是否被监控），本命令会在客户端过滤出被监控的那部分。早期版本
读取的是另一个未文档化的资源（LwM2M .../60001/0/27300），假设返回值
按 compose 项目分组，但在一台真实部署了容器的服务器上测试时返回了
空结果——已根据真实抓包修复。

Example 示例: eim-skill device docker monitored "08:00:27:33:EC:2E"

Example output 输出示例 (real, trimmed 真实返回，已截断):
  {
    "status": 200, "msg": "",
    "data": [
      {
        "id": "d099f6081f45...", "name": "ithing",
        "image": "172.21.73.109/apphub/apphub-ithings:v2.9.72",
        "command": "/root/adv/run.sh", "ports": " 8082:8082",
        "state": "running", "compose": "edgemindmanagermain",
        "created": "2026-07-10 16:02:03", "ismonitor": "true"
      }
    ]
  }

Example output 输出示例 (empty when nothing is selected for monitoring
未选中任何容器监控时返回空数组): {"status": 200, "msg": "", "data": []}`,
	Args: requireArgs(cobra.ExactArgs(1)),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		containers, err := client.GetDockerInfo(endpoint)
		if err != nil {
			return err
		}
		printJSON(containers)
		return nil
	},
}

var deviceDockerSetMonitorCmd = &cobra.Command{
	Use:   "set-monitor <endpoint-or-name> <container-name> <on|off>",
	Short: "Add or remove a specific Docker container from monitoring",
	Long: `Add (on) or remove (off) one Docker container, identified by name,
to/from the device's monitored-container list (see "device docker
monitored"). Just supply the container name - this looks up the matching
container in "device docker list" itself and resends its full details,
since the underlying device API requires that (not just a bare name).

把某个 Docker 容器（用名称指定）加入（on）或移出（off）设备的监控
容器列表（见 "device docker monitored"）。只需要提供容器名称——本命令
会自动从 "device docker list" 里查到这个容器的完整信息并重新提交，
因为设备底层接口要求的是完整信息，而不是只给一个名称。

Not documented in AppHub-v2.3.0.md; confirmed against a live server via
a real packet capture and cross-verified end-to-end: adding a container
makes it appear in "device docker monitored", and removing it makes it
disappear again (without affecting any other container already being
monitored).

未在 AppHub-v2.3.0.md 中记录；已通过真实抓包在真实服务器上验证过，
并做了端到端交叉验证：加入一个容器后它会出现在 "device docker
monitored" 里，移出后又会消失（不影响其他已经在被监控的容器）。

Example 示例: eim-skill device docker set-monitor "146打包机" apphub on

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED"}}`,
	Args: requireArgs(cobra.ExactArgs(3)),
	RunE: func(cmd *cobra.Command, args []string) error {
		enabled, err := parseOnOff(args[2])
		if err != nil {
			return err
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		if err := client.SetDockerContainerMonitored(endpoint, args[1], enabled); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var deviceDockerStartCmd = &cobra.Command{
	Use:   "start <endpoint-or-name> <container-name>",
	Short: "Start one Docker container on a device",
	Long: `Start one Docker container by name on a device.

按名称启动设备上的单个 Docker 容器。

Uses PUT /api/devicemgr/{agentId}/60002/0/1009 with body
{"id":1009,"value":"{\"pkgname\":...,\"version\":\"\",\"taskname\":\"###\",\"target\":\"/60002/0/1009\"}"}.

使用 PUT /api/devicemgr/{agentId}/60002/0/1009，请求体为
{"id":1009,"value":"{\"pkgname\":...,\"version\":\"\",\"taskname\":\"###\",\"target\":\"/60002/0/1009\"}"}。

Example 示例: eim-skill device docker start "146打包机" sdk440llminfer123

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED"}}`,
	Args: requireArgs(cobra.ExactArgs(2)),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		if err := client.StartDockerContainer(endpoint, args[1]); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var deviceDockerStopCmd = &cobra.Command{
	Use:   "stop <endpoint-or-name> <container-name>",
	Short: "Stop one Docker container on a device",
	Long: `Stop one Docker container by name on a device.

按名称停止设备上的单个 Docker 容器。

Uses PUT /api/devicemgr/{agentId}/60002/0/1010 with body
{"id":1010,"value":"{\"pkgname\":...,\"version\":\"\",\"taskname\":\"###\",\"target\":\"/60002/0/1010\"}"}.

使用 PUT /api/devicemgr/{agentId}/60002/0/1010，请求体为
{"id":1010,"value":"{\"pkgname\":...,\"version\":\"\",\"taskname\":\"###\",\"target\":\"/60002/0/1010\"}"}。

Example 示例: eim-skill device docker stop "146打包机" sdk440llminfer123

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED"}}`,
	Args: requireArgs(cobra.ExactArgs(2)),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		if err := client.StopDockerContainer(endpoint, args[1]); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var deviceDockerRestartCmd = &cobra.Command{
	Use:   "restart <endpoint-or-name> <container-name>",
	Short: "Restart one Docker container on a device",
	Long: `Restart one Docker container by name on a device.

按名称重启设备上的单个 Docker 容器。

Uses PUT /api/devicemgr/{agentId}/60002/0/1011 with body
{"id":1011,"value":"{\"pkgname\":...,\"version\":\"\",\"taskname\":\"###\",\"target\":\"/60002/0/1011\"}"}.

使用 PUT /api/devicemgr/{agentId}/60002/0/1011，请求体为
{"id":1011,"value":"{\"pkgname\":...,\"version\":\"\",\"taskname\":\"###\",\"target\":\"/60002/0/1011\"}"}。

Example 示例: eim-skill device docker restart "146打包机" sdk440llminfer123

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED"}}`,
	Args: requireArgs(cobra.ExactArgs(2)),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		if err := client.RestartDockerContainer(endpoint, args[1]); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

// USB peripheral change monitoring

var deviceUsbCmd = &cobra.Command{
	Use:   "usb",
	Short: "USB peripheral change-alerting and mass-storage device access",
	Long: `Two separate, independent USB-related switches: peripheral
change-alerting (get-monitor/set-monitor - alert when a USB device is
connected/disconnected) and mass-storage device access (set-storage -
whether USB storage devices like a flash drive can be used).

两个相互独立的USB相关开关：外设变更告警（get-monitor/set-monitor——
设备插拔时触发提醒）和USB存储设备访问权限（set-storage——是否允许
使用USB存储设备如U盘）。`,
}

var deviceUsbGetMonitorCmd = &cobra.Command{
	Use:   "get-monitor <endpoint-or-name>",
	Short: "Report whether USB peripheral change monitoring is enabled",
	Long: `Report whether USB peripheral change-alerting is currently on (alert when
a USB device is connected/disconnected).

查询USB外设变更告警（插拔提醒）当前是否开启。

Not documented in AppHub-v2.3.0.md; uses the "get_usb_change"
devicecontrol action, confirmed against a live server via a real packet
capture and cross-verified by toggling "usb set-monitor" true/false and
observing this correctly reflects each change.

未在 AppHub-v2.3.0.md 中记录；使用 "get_usb_change" 设备控制动作，
已通过真实抓包在真实服务器上验证过，并通过交替切换 "usb set-monitor"
true/false 后观察这里能正确反映每次变化，做了交叉验证。

Example 示例: eim-skill device usb get-monitor "08:00:27:33:EC:2E"

Example output 输出示例 (real 真实返回): {"status": 200, "msg": "", "data": {"enabled": false}}`,
	Args: requireArgs(cobra.ExactArgs(1)),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		enabled, err := client.GetUSBMonitorEnabled(endpoint)
		if err != nil {
			return err
		}
		printJSON(map[string]bool{"enabled": enabled})
		return nil
	},
}

var deviceUsbSetMonitorCmd = &cobra.Command{
	Use:   "set-monitor <endpoint-or-name> <on|off>",
	Short: "Enable or disable USB peripheral change monitoring",
	Long: `Turn USB peripheral change-alerting on or off (alert when a USB device
is connected/disconnected).

开启或关闭USB外设变更告警（插拔提醒）。

Not documented in AppHub-v2.3.0.md; uses the "set_peripheral_monitor"/
"usb_change" devicecontrol action, confirmed against a live server via a
real packet capture and cross-verified against "usb get-monitor":
toggling this true/false and re-reading "get-monitor" correctly reflects
each change.

未在 AppHub-v2.3.0.md 中记录；使用 "set_peripheral_monitor"/
"usb_change" 设备控制动作，已通过真实抓包在真实服务器上验证过，并与
"usb get-monitor" 做了交叉验证：交替切换 true/false 后再查询
"get-monitor"，能正确反映每次变化。

This is a separate, independent switch from USB mass-storage access
("device usb set-storage" - whether the device allows USB storage
devices like a flash drive to be used) - the two shouldn't be confused,
even though both are exposed under devicecontrol resource 27601.

这和"USB存储设备访问权限"（"device usb set-storage"，是否允许USB存储
设备如U盘被使用）是两个独立的开关，虽然都挂在同一个设备控制资源
（27601）下，但不要混淆。

Example 示例: eim-skill device usb set-monitor "146打包机" on

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED"}}`,
	Args: requireArgs(cobra.ExactArgs(2)),
	RunE: func(cmd *cobra.Command, args []string) error {
		enabled, err := parseOnOff(args[1])
		if err != nil {
			return err
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		if err := client.SetUSBMonitorEnabled(endpoint, enabled); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var deviceUsbGetStorageCmd = &cobra.Command{
	Use:   "get-storage <endpoint-or-name>",
	Short: "Report whether USB mass-storage device access is enabled",
	Long: `Report whether the device currently allows USB storage devices (e.g. a
USB flash drive) to be used.

查询设备当前是否允许使用USB存储设备（如U盘）。

Not documented in AppHub-v2.3.0.md; uses the "get_usb_storage_status"
devicecontrol action, confirmed against a live Windows device via a real
packet capture and cross-verified by toggling "usb set-storage" true/
false and observing this correctly reflects each change in real time.
Not every device supports this action - a Linux industrial PC tested
returned an empty result regardless of the last value set, which this
CLI normalizes to "enabled": false (there's no way to distinguish
"genuinely off" from "not supported" via this action alone).

未在 AppHub-v2.3.0.md 中记录；使用 "get_usb_storage_status" 设备控制
动作，已通过真实抓包在一台真实的Windows设备上验证过，并交替切换
"usb set-storage" 的 true/false 后观察这里能实时正确反映每次变化。
并非所有设备都支持这个动作——测试过的一台Linux工控机无论最后设置成
什么值，都返回空结果，本CLI将其归一化为 "enabled": false（仅凭这个
动作本身无法区分"确实关闭"和"设备不支持"两种情况）。

Example 示例: eim-skill device usb get-storage "08:00:27:33:EC:2E"

Example output 输出示例 (real 真实返回): {"status": 200, "msg": "", "data": {"enabled": true}}`,
	Args: requireArgs(cobra.ExactArgs(1)),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		enabled, err := client.GetUSBStorageEnabled(endpoint)
		if err != nil {
			return err
		}
		printJSON(map[string]bool{"enabled": enabled})
		return nil
	},
}

var deviceUsbSetStorageCmd = &cobra.Command{
	Use:   "set-storage <endpoint-or-name> <on|off>",
	Short: "Enable or disable USB mass-storage device access on a device",
	Long: `Turn USB mass-storage device access on or off - when off, the device
won't allow USB storage devices (e.g. a USB flash drive) to be used.

开启或关闭设备的USB存储设备访问权限——关闭后设备将不允许使用USB存储
设备（如U盘）。

Not documented in AppHub-v2.3.0.md; uses the "set_usb_storage_status"
devicecontrol action, confirmed against a live server via a real packet
capture and cross-checked in "log operations list"'s audit trail: on
(true) is logged as the friendly event "enableUsb", off (false) as
"disableUsb". Also cross-verified against "usb get-storage" on a Windows
device: toggling this true/false and re-reading "get-storage" correctly
reflects each change in real time.

未在 AppHub-v2.3.0.md 中记录；使用 "set_usb_storage_status" 设备控制
动作，已通过真实抓包在真实服务器上验证过，并在 "log operations list"
的审计记录里交叉核实：on（true）会被记录为友好事件名 "enableUsb"，
off（false）为 "disableUsb"。还在一台Windows设备上与
"usb get-storage" 做了交叉验证：交替切换 true/false 后再查询
"get-storage"，能实时正确反映每次变化。

This is a separate, independent switch from USB peripheral
change-alerting ("device usb set-monitor") - the two shouldn't be
confused, even though both are exposed under devicecontrol resource
27601.

这和USB外设变更告警（"device usb set-monitor"）是两个独立的开关，虽然
都挂在同一个设备控制资源（27601）下，但不要混淆。

Example 示例: eim-skill device usb set-storage "146打包机" off

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED"}}`,
	Args: requireArgs(cobra.ExactArgs(2)),
	RunE: func(cmd *cobra.Command, args []string) error {
		enabled, err := parseOnOff(args[1])
		if err != nil {
			return err
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		if err := client.SetUSBStorageEnabled(endpoint, enabled); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var deviceSyncTimeCmd = &cobra.Command{
	Use:   "sync-time <endpoint-or-name>",
	Short: "Trigger an immediate clock sync on a device (best effort, undocumented)",
	Long: `Trigger the device to sync its system clock immediately.

触发设备立即同步系统时钟。

UNVERIFIED: not documented in AppHub-v2.3.0.md; inferred from the
"syncTime" action key handled by DeviceLink's PluginsServlet for batch
operations. Hasn't been verified against real firmware.

未在 AppHub-v2.3.0.md 中记录，是根据 DeviceLink 的 PluginsServlet
批量操作里 "syncTime" 动作推断实现，尚未在真实固件上验证过。

Example 示例: eim-skill device sync-time "146打包机"

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED"}}`,
	Args: requireArgs(cobra.ExactArgs(1)),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		if err := client.SyncTime(endpoint); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

func init() {
	deviceCmd.AddCommand(deviceBoardInfoCmd)
	deviceCmd.AddCommand(deviceSyncTimeCmd)

	deviceCmd.AddCommand(deviceThresholdCmd)
	deviceThresholdCmd.AddCommand(deviceThresholdGetCmd)
	deviceThresholdCmd.AddCommand(deviceThresholdSetCmd)
	deviceThresholdSetCmd.Flags().Float64("warn", 0, "warning level (告警阈值)")
	deviceThresholdSetCmd.Flags().Float64("error", 0, "error level (错误阈值)")

	deviceCmd.AddCommand(deviceAlertCmd)
	deviceAlertCmd.AddCommand(deviceAlertHistoryCmd)
	addTimeWindowFlags(deviceAlertHistoryCmd)
	deviceAlertHistoryCmd.Flags().String("type", "", `alert category filter: "hardware", "software", or "peripheral" (empty = no filter) - 告警类别过滤：hardware(硬件)/software(软件)/peripheral(外设)，留空则不过滤`)
	deviceAlertHistoryCmd.Flags().String("level", "", "alert severity level filter, server-defined (empty = no filter) - 告警级别过滤，具体取值由服务器定义（留空则不过滤）")
	deviceAlertHistoryCmd.Flags().Int("page", 1, "page number, 1-based (页码，从1开始)")
	deviceAlertHistoryCmd.Flags().Int("limit", 20, "how many entries per page (每页返回多少条)")

	deviceCmd.AddCommand(deviceProcessCmd)
	deviceProcessCmd.AddCommand(deviceProcessListCmd)
	deviceProcessCmd.AddCommand(deviceProcessMonitoredCmd)
	deviceProcessCmd.AddCommand(deviceProcessSetMonitorCmd)

	deviceCmd.AddCommand(deviceDockerCmd)
	deviceDockerCmd.AddCommand(deviceDockerListCmd)
	deviceDockerCmd.AddCommand(deviceDockerMonitoredCmd)
	deviceDockerCmd.AddCommand(deviceDockerSetMonitorCmd)
	deviceDockerCmd.AddCommand(deviceDockerStartCmd)
	deviceDockerCmd.AddCommand(deviceDockerStopCmd)
	deviceDockerCmd.AddCommand(deviceDockerRestartCmd)

	deviceCmd.AddCommand(deviceUsbCmd)
	deviceUsbCmd.AddCommand(deviceUsbGetMonitorCmd)
	deviceUsbCmd.AddCommand(deviceUsbSetMonitorCmd)
	deviceUsbCmd.AddCommand(deviceUsbGetStorageCmd)
	deviceUsbCmd.AddCommand(deviceUsbSetStorageCmd)
}
