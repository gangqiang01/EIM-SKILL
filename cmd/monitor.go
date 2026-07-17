package cmd

import (
	"fmt"

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
	Args: cobra.ExactArgs(1),
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
	Args: cobra.ExactArgs(1),
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

Example 示例: eim-skill device threshold set "146打包机" cpu_usage on --warn 70 --error 90
       eim-skill device threshold set "146打包机" cpu_temp off   # just toggles off, keeps existing warn/error

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED"}}`,
	Args: cobra.ExactArgs(3),
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

		if err := client.SetThreshold(endpoint, metric, enabled, warn, errorLevel); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

// process monitoring

var deviceProcessCmd = &cobra.Command{
	Use:   "process",
	Short: "List the processes currently selected for application/process monitoring",
	Long: `List the processes on a device currently selected for application/process
monitoring.

列出设备上当前被选中监控的进程。`,
}

var deviceProcessMonitoredCmd = &cobra.Command{
	Use:   "monitored <endpoint-or-name>",
	Short: "List the processes currently selected for application/process monitoring",
	Long: `List the processes currently selected for monitoring, with normalized
numeric CPU/memory percentages and a real is_monitored boolean.

列出当前被选中监控的进程，CPU/内存占用已转成数字百分比，is_monitored
是真正的布尔值。

Example 示例: eim-skill device process monitored "08:00:27:33:EC:2E"

Example output 输出示例 (empty when nothing is selected 未选中任何进程
时返回空数组): {"status": 200, "msg": "", "data": []}`,
	Args: cobra.ExactArgs(1),
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

// docker monitoring

var deviceDockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Get info on a device's monitored Docker containers",
	Long: `List the Docker containers on a device that are currently selected for
monitoring.

列出设备上当前被选中监控的 Docker 容器。`,
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
	Args: cobra.ExactArgs(1),
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

// USB storage device access

var deviceUsbCmd = &cobra.Command{
	Use:   "usb",
	Short: "Get/set whether USB storage device access is enabled",
	Long: `Get/set whether the device allows USB storage devices (e.g. a USB flash
drive) to be used.

获取/设置设备是否允许使用 USB 存储设备（如U盘）。`,
}

var deviceUsbGetMonitorCmd = &cobra.Command{
	Use:   "get-monitor <endpoint-or-name>",
	Short: "Report whether USB storage device access is enabled",
	Long: `Report whether the device currently allows USB storage devices to be used.

查询设备当前是否允许使用 USB 存储设备。

Not documented in AppHub-v2.3.0.md; uses the "get_usb_storage_status"
devicecontrol action, confirmed against a live server via a real packet
capture. An earlier version instead used "get_usb_change" (based on an
incorrect assumption that this command was about alerting on USB
connect/disconnect rather than storage access) - fixed via the real
packet capture. Some devices report an empty result for this action
(returned as enabled: false here) rather than a definite value.

未在 AppHub-v2.3.0.md 中记录；使用 "get_usb_storage_status" 设备控制
动作，已通过真实抓包在真实服务器上验证过。早期版本用的是
"get_usb_change"（错误地以为这个命令是"USB插拔告警"而不是"存储访问
开关"）——已根据真实抓包修复。部分设备对这个动作会返回空结果（这里会
归一化为 enabled: false），而不是一个确切的值。

Example 示例: eim-skill device usb get-monitor "08:00:27:33:EC:2E"

Example output 输出示例 (real 真实返回): {"status": 200, "msg": "", "data": {"enabled": false}}`,
	Args: cobra.ExactArgs(1),
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
	Short: "Enable or disable USB storage device access",
	Long: `Turn USB storage device access on or off - when off, the device won't
allow USB storage devices (e.g. a USB flash drive) to be used.

开启或关闭 USB 存储设备访问——关闭后设备将不允许使用 USB 存储设备
（如U盘）。

Not documented in AppHub-v2.3.0.md; uses the "set_usb_storage_status"
devicecontrol action, confirmed against a live server via a real packet
capture and cross-checked in "log operations list"'s audit trail, where
this action is logged under the friendly event names "enableUsb"/
"disableUsb" (unlike most actions, which are logged as raw JSON) -
confirming the server specially recognizes it as a real, meaningful
action. An earlier version instead sent "set_peripheral_monitor"/
"usb_change" (based on an incorrect assumption that this command was
about alerting on USB connect/disconnect), which the audit trail logged
as raw unrecognized JSON rather than a friendly event name - fixed via
the real packet capture.

未在 AppHub-v2.3.0.md 中记录；使用 "set_usb_storage_status" 设备控制
动作，已通过真实抓包在真实服务器上验证过，并在 "log operations list"
的审计记录里交叉核实——这个动作会被记录为友好的事件名
"enableUsb"/"disableUsb"（不同于大多数动作只会记录原始JSON），说明
服务器确实识别并处理了这个动作。早期版本用的是
"set_peripheral_monitor"/"usb_change"（错误地以为这个命令是"USB插拔
告警"），审计记录里只留下了未被识别的原始JSON，而不是友好的事件名——
已根据真实抓包修复。

Example 示例: eim-skill device usb set-monitor "146打包机" on

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED"}}`,
	Args: cobra.ExactArgs(2),
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
	Args: cobra.ExactArgs(1),
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

	deviceCmd.AddCommand(deviceProcessCmd)
	deviceProcessCmd.AddCommand(deviceProcessMonitoredCmd)

	deviceCmd.AddCommand(deviceDockerCmd)
	deviceDockerCmd.AddCommand(deviceDockerMonitoredCmd)

	deviceCmd.AddCommand(deviceUsbCmd)
	deviceUsbCmd.AddCommand(deviceUsbGetMonitorCmd)
	deviceUsbCmd.AddCommand(deviceUsbSetMonitorCmd)
}
