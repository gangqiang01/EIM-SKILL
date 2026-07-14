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

var deviceSetMonitorCmd = &cobra.Command{
	Use:   "set-monitor <endpoint-or-name> <on|off>",
	Short: "Enable or disable overall hardware monitoring (CPU/memory/storage/network/battery thresholds)",
	Long: `Master on/off switch for hardware threshold monitoring (CPU/memory/
storage/network/battery). When off, per-metric thresholds configured via
"device threshold set" are not evaluated/alerted on.

硬件阈值监控（CPU/内存/存储/网络/电池）的总开关。关闭后，即使用
"device threshold set" 配置了具体指标的阈值，也不会再进行告警判定。

Example 示例: eim-skill device set-monitor "146打包机" on

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
		if err := client.SetMonitorEnabled(endpoint, enabled); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
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
	Long: `Enable or disable alerting for one metric, and (when enabling) set its
warn/error levels via --warn/--error.

启用或关闭某一项监控指标的告警，启用时可通过 --warn/--error 设置对应
的警告/错误阈值。

Supported metrics 支持的指标:
  cpu_temp              CPU 温度
  cpu_usage              CPU 使用率
  memory_usage          内存使用率
  storage_usage          存储使用率
  battery_available      电池剩余电量
  network_upload_speed    网络上传速度
  network_download_speed  网络下载速度

Example 示例: eim-skill device threshold set "146打包机" cpu_usage on --warn 70 --error 90

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

		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
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
	Short: "List processes and control application/process monitoring",
	Long: `List a device's running processes and control whether individual-process
monitoring is turned on. Per-process selection (choosing exactly which
processes to watch) mirrors the web console's picker but isn't wired up in
this CLI yet - only the overall on/off toggle is.

列出设备上正在运行的进程，并控制应用/进程监控的总开关。像web控制台
那样"精确挑选监控哪几个进程"的功能，本CLI目前还没有实现，只支持
整体开关。`,
}

var deviceProcessListCmd = &cobra.Command{
	Use:   "list <endpoint-or-name>",
	Short: "List running processes/threads and their monitor status",
	Long: `List every running process/thread on the device, with CPU/memory load and
whether it's currently selected for monitoring.

列出设备上所有正在运行的进程/线程，包含各自的CPU/内存占用，以及是否
被选中监控。

Example 示例: eim-skill device process list "08:00:27:33:EC:2E"

Example output 输出示例 (real, trimmed 真实返回，已截断):
  {
    "status": 200, "msg": "",
    "data": [
      {
        "threadcmd": "systemd-udevd", "threadpid": "301",
        "threadcpuloading": "0.06", "threadmemloading": "0.05",
        "threadstatus": "S", "threadusername": "root",
        "threadismonitor": "false"
      }
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
	Long: `List only the processes currently selected for monitoring (a subset of
"device process list"), with normalized numeric CPU/memory percentages and
a real is_monitored boolean.

只列出当前被选中监控的进程（是 "device process list" 的一个子集），
CPU/内存占用已转成数字百分比，is_monitored 是真正的布尔值。

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

var deviceProcessSetMonitorCmd = &cobra.Command{
	Use:   "set-monitor <endpoint-or-name> <on|off>",
	Short: "Enable or disable application/process monitoring",
	Long: `Master on/off switch for application/process monitoring.

应用/进程监控的总开关。

Example 示例: eim-skill device process set-monitor "146打包机" on

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
		if err := client.SetProcessMonitorEnabled(endpoint, enabled); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

// docker monitoring

var deviceDockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Get Docker container info and control Docker monitoring",
	Long: `List Docker compose/swarm groups and containers deployed on a device, and
control whether Docker monitoring is on. Per-container selection isn't
wired up in this CLI - only the overall on/off toggle is.

列出设备上部署的 Docker compose/swarm 分组及容器，并控制Docker监控
总开关。像web控制台那样精确挑选监控哪个容器的功能，本CLI暂未实现。`,
}

var deviceDockerInfoCmd = &cobra.Command{
	Use:   "info <endpoint-or-name>",
	Short: "Get the Docker compose/swarm/image groups deployed on a device",
	Long: `List every Docker container on the device grouped by its compose
project, including each container's running state.

列出设备上所有 Docker 容器，按 compose 项目分组，包含每个容器的运行
状态。

Example 示例: eim-skill device docker info "08:00:27:33:EC:2E"

Example output 输出示例 (empty when Docker isn't used/deployed 设备未
部署 Docker 时返回空数组): {"status": 200, "msg": "", "data": []}`,
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
		groups, err := client.GetDockerInfo(endpoint)
		if err != nil {
			return err
		}
		printJSON(groups)
		return nil
	},
}

var deviceDockerSetMonitorCmd = &cobra.Command{
	Use:   "set-monitor <endpoint-or-name> <on|off>",
	Short: "Enable or disable Docker container monitoring",
	Long: `Master on/off switch for Docker container monitoring.

Docker 容器监控的总开关。

Example 示例: eim-skill device docker set-monitor "146打包机" on

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
		if err := client.SetDockerMonitorEnabled(endpoint, enabled); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

// USB peripheral monitoring

var deviceUsbCmd = &cobra.Command{
	Use:   "usb",
	Short: "USB peripheral change-alerting and mass-storage access control",
	Long: `Two distinct USB features live under this group: change-alerting
(get-monitor/set-monitor - alert when a USB peripheral is plugged/
unplugged) and mass-storage access control (set-storage - allow/deny the
device acting as USB storage).

这个命令组下有两个互相独立的 USB 相关功能：变更告警
（get-monitor/set-monitor，USB外设插拔时触发提醒）和U盘存储访问控制
（set-storage，允许/禁止设备被当作U盘使用）。`,
}

var deviceUsbGetMonitorCmd = &cobra.Command{
	Use:   "get-monitor <endpoint-or-name>",
	Short: "Report whether USB peripheral change monitoring is enabled",
	Long: `Report whether USB peripheral change-alerting is currently on (alert when
a USB device is connected/disconnected) - distinct from "usb set-storage",
which controls whether USB mass storage itself is allowed.

查询USB外设变更告警（插拔提醒）当前是否开启——注意这和 "usb set-storage"
（是否允许U盘存储访问）是两个不同的功能。

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
	Short: "Enable or disable USB peripheral change monitoring",
	Long: `Turn USB peripheral change-alerting on or off (alert when a USB device
is connected/disconnected).

开启或关闭USB外设变更告警（插拔提醒）。

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

var deviceUsbSetStorageCmd = &cobra.Command{
	Use:   "set-storage <endpoint-or-name> <on|off>",
	Short: "Enable or disable USB mass-storage access (unverified - see Long help)",
	Long: `Enable or disable USB mass-storage access on a device (allow/deny the
device being used as a USB flash drive) - distinct from "usb set-monitor",
which is about change-alerting, not the storage toggle itself.

启用或禁用设备的U盘存储访问（允许/禁止设备被当作U盘使用）——注意这
和 "usb set-monitor"（插拔告警）是两个不同的功能。

UNVERIFIED 未验证: this is not documented in AppHub-v2.3.0.md and was
never confirmed against a live device. Its sibling read command ("usb
get-storage") queried a guessed action name and was removed after live
testing showed the server doesn't recognize it (it silently returned an
empty result instead of a real flag) - so this write-side action name may
be equally wrong. Verify it actually changes device behavior before
relying on it in any unattended workflow.

未在 AppHub-v2.3.0.md 中记录，也从未在真实设备上验证过。它的配套查询
命令 "usb get-storage" 此前查询了一个猜测的动作名，实测中被服务器
忽略（只返回空结果而非真实状态），已因此被移除——这个写入动作名也
可能同样是错的，正式无人值守流程使用前请务必先自行验证效果。

Example 示例: eim-skill device usb set-storage "146打包机" off`,
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
	deviceCmd.AddCommand(deviceSetMonitorCmd)
	deviceCmd.AddCommand(deviceBoardInfoCmd)
	deviceCmd.AddCommand(deviceSyncTimeCmd)

	deviceCmd.AddCommand(deviceThresholdCmd)
	deviceThresholdCmd.AddCommand(deviceThresholdGetCmd)
	deviceThresholdCmd.AddCommand(deviceThresholdSetCmd)
	deviceThresholdSetCmd.Flags().Float64("warn", 0, "warning level (告警阈值)")
	deviceThresholdSetCmd.Flags().Float64("error", 0, "error level (错误阈值)")

	deviceCmd.AddCommand(deviceProcessCmd)
	deviceProcessCmd.AddCommand(deviceProcessListCmd)
	deviceProcessCmd.AddCommand(deviceProcessMonitoredCmd)
	deviceProcessCmd.AddCommand(deviceProcessSetMonitorCmd)

	deviceCmd.AddCommand(deviceDockerCmd)
	deviceDockerCmd.AddCommand(deviceDockerInfoCmd)
	deviceDockerCmd.AddCommand(deviceDockerSetMonitorCmd)

	deviceCmd.AddCommand(deviceUsbCmd)
	deviceUsbCmd.AddCommand(deviceUsbGetMonitorCmd)
	deviceUsbCmd.AddCommand(deviceUsbSetMonitorCmd)
	deviceUsbCmd.AddCommand(deviceUsbSetStorageCmd)
}
