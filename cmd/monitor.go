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

// USB peripheral change monitoring

var deviceUsbCmd = &cobra.Command{
	Use:   "usb",
	Short: "USB peripheral change-alerting",
	Long: `Get/set whether USB peripheral change-alerting is on (alert when a USB
device is connected/disconnected).

获取/设置 USB 外设变更告警是否开启（设备插拔时触发提醒）。`,
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
(whether the device allows USB storage devices like a flash drive to be
used) - the two shouldn't be confused, even though both are exposed
under devicecontrol resource 27601. A "get_usb_storage_status"/
"set_usb_storage_status" action pair also exists for that (its SET call
succeeds and is logged under a distinct friendly audit-log event name,
"enableUsb"/"disableUsb"), but that action's GET side never reflected
the change on a real device tested (always returned an empty result), so
it isn't currently exposed as a command here.

这和"USB存储设备访问权限"（是否允许USB存储设备如U盘被使用）是两个
独立的开关，虽然都挂在同一个设备控制资源（27601）下，但不要混淆。
也存在一组 "get_usb_storage_status"/"set_usb_storage_status" 动作对应
后者（它的SET调用会成功，并且会被记录为独立的友好审计事件名
"enableUsb"/"disableUsb"），但在真实设备上测试时，这个动作的GET一侧
从未正确反映过变化（一直返回空结果），所以目前没有把它做成命令。

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

	deviceCmd.AddCommand(deviceProcessCmd)
	deviceProcessCmd.AddCommand(deviceProcessMonitoredCmd)

	deviceCmd.AddCommand(deviceDockerCmd)
	deviceDockerCmd.AddCommand(deviceDockerMonitoredCmd)

	deviceCmd.AddCommand(deviceUsbCmd)
	deviceUsbCmd.AddCommand(deviceUsbGetMonitorCmd)
	deviceUsbCmd.AddCommand(deviceUsbSetMonitorCmd)
}
