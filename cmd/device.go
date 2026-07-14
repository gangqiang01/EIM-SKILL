package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var deviceCmd = &cobra.Command{
	Use:   "device",
	Short: "Query and control devices managed by DeviceOn-EIM",
	Long: `Query and control devices managed by DeviceOn-EIM.

查询和控制由 DeviceOn-EIM 管理的设备：列表/详情/上下线与告警状态、
重启/关机/调音量/调背光（含按设备组批量执行）、加入分组、删除设备。

Every device-targeting command accepts either the device's endpoint id
(its MAC-like id, e.g. "08:00:27:33:EC:2E") or its human-readable device
name (devName, e.g. "146打包机") as <endpoint-or-name> - it is resolved
against the live device list automatically, so you never need to look up
the endpoint id yourself first.

每个以设备为目标的命令，<endpoint-or-name> 参数都同时支持填设备的
endpoint id（类似MAC地址）或人类可读的设备名称，程序会自动去当前设备
列表里查找对应的 endpoint，无需提前手动查询。`,
}

func init() {
	rootCmd.AddCommand(deviceCmd)
	deviceCmd.AddCommand(deviceListCmd)
	deviceCmd.AddCommand(deviceCountsCmd)
	deviceCmd.AddCommand(deviceHealthCmd)
	deviceCmd.AddCommand(deviceGetCmd)
	deviceCmd.AddCommand(deviceDetailsCmd)
	deviceCmd.AddCommand(deviceDeleteCmd)
	deviceCmd.AddCommand(deviceAddToGroupCmd)
	deviceCmd.AddCommand(deviceMonitorCmd)
	deviceCmd.AddCommand(deviceRebootCmd)
	deviceCmd.AddCommand(deviceShutdownCmd)
	deviceCmd.AddCommand(deviceSetVolumeCmd)
	deviceCmd.AddCommand(deviceSetBacklightCmd)
	deviceCmd.AddCommand(deviceRebootGroupCmd)
	deviceCmd.AddCommand(deviceShutdownGroupCmd)
	deviceCmd.AddCommand(deviceSetVolumeGroupCmd)
	deviceCmd.AddCommand(deviceSetBacklightGroupCmd)

	deviceListCmd.Flags().String("group", "", "only list devices in this device group (设备组名称)")
	deviceListCmd.Flags().Bool("online", false, "only list online devices (只看在线设备)")
	deviceListCmd.Flags().Bool("offline", false, "only list offline devices (只看离线设备)")
	deviceListCmd.Flags().Bool("alarm", false, "only list devices currently in an alarm state - software/hardware status not normal (只看当前有软件/硬件告警的设备)")
	deviceListCmd.MarkFlagsMutuallyExclusive("online", "offline", "alarm")

	deviceMonitorCmd.Flags().String("type", "", `monitor data type: "cpu" or "memory" (required) - 监控数据类型：cpu 或 memory（必填）`)
	deviceMonitorCmd.Flags().String("last", "", `only return samples from the last period, e.g. "30m", "12h", "3d" (default: no time filter) - 只返回最近一段时间的采样点，如 "30m"(30分钟)/"12h"(12小时)/"3d"(3天)，不填则不限制时间范围`)
	_ = deviceMonitorCmd.MarkFlagRequired("type")

	for _, c := range []*cobra.Command{deviceRebootGroupCmd, deviceShutdownGroupCmd, deviceSetVolumeGroupCmd, deviceSetBacklightGroupCmd} {
		c.Flags().String("label", "", "only target devices in the group that also carry this label (只对组内带有该标签的设备生效，留空则对整组生效)")
	}
}

var deviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all devices, or filter by group / online / offline / alarm state",
	Long: `List all devices, optionally filtered by group, online/offline status,
or alarm state. --online, --offline, and --alarm are mutually exclusive
(each is a distinct server-side filter); --group can combine with any one
of them, or be used alone.

列出所有设备，可选按设备组 / 在线状态 / 告警状态过滤。--online、
--offline、--alarm 三者互斥（同一时刻只能选一个），--group 可以单独使用，
也可以和它们中的任意一个组合使用，表示"某个组里的在线/离线/告警设备"。

Examples 示例:
  eim-skill device list                       # every device 所有设备
  eim-skill device list --group default       # devices in group "default" 组"default"下的设备
  eim-skill device list --online              # online devices only 只看在线设备
  eim-skill device list --alarm --group default  # alarming devices in group "default"

Example output 输出示例 (trimmed 已截断):
  {
    "status": 200, "msg": "",
    "data": [
      {
        "endpoint": "08:00:27:33:EC:2E",
        "devName": "08:00:27:33:EC:2E",
        "isonline": true,
        "deviceStatus": 0,
        "deviceStatusText": "normal",
        "boardName": "VirtualBox",
        "os": "linux"
      }
    ]
  }

deviceStatusText is "normal"/"warning"/"error", derived from the raw
deviceStatus code (0/1/2) - the same value --alarm filters on (non-zero).
deviceStatusText 是根据原始数字状态码 deviceStatus（0/1/2）转换出的可读
文本（正常/告警-警告/告警-错误），--alarm 过滤的就是 deviceStatus 非 0
的设备。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		group, _ := cmd.Flags().GetString("group")
		online, _ := cmd.Flags().GetBool("online")
		offline, _ := cmd.Flags().GetBool("offline")
		alarm, _ := cmd.Flags().GetBool("alarm")

		client, err := newClient()
		if err != nil {
			return err
		}

		if alarm {
			devices, err := client.ListAlarmDevices(group)
			if err != nil {
				return err
			}
			printJSON(devices)
			return nil
		}

		if online || offline {
			isOnline := online
			devices, err := client.ListDevicesByCondition(&isOnline, group, "", "")
			if err != nil {
				return err
			}
			printJSON(devices)
			return nil
		}

		if group != "" {
			devices, err := client.ListDevicesByGroup(group)
			if err != nil {
				return err
			}
			printJSON(devices)
			return nil
		}

		devices, err := client.ListDevices()
		if err != nil {
			return err
		}
		printJSON(devices)
		return nil
	},
}

var deviceCountsCmd = &cobra.Command{
	Use:   "counts",
	Short: "Get the number of online and offline devices",
	Long: `Get the total number of online and offline devices across the whole
server (not scoped to a group).

获取整个服务器范围内在线/离线设备的总数（不区分设备组）。

Example 示例: eim-skill device counts

Example output 输出示例:
  {"status": 200, "msg": "", "data": {"online": 1, "offline": 5}}`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		counts, err := client.GetDeviceCounts()
		if err != nil {
			return err
		}
		printJSON(counts)
		return nil
	},
}

var deviceHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Get the status breakdown (normal/warning/error) of online devices",
	Long: `Get how many currently-online devices are normal/warning/error, across
the whole server.

获取当前在线设备中，处于 正常/警告/错误 三种状态的设备各有多少台
（全服务器范围，不区分设备组）。

Example 示例: eim-skill device health

Example output 输出示例:
  {"status": 200, "msg": "", "data": {"normal": 1, "warning": 0, "error": 0}}`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		health, err := client.GetOnlineDeviceHealth()
		if err != nil {
			return err
		}
		printJSON(health)
		return nil
	},
}

var deviceGetCmd = &cobra.Command{
	Use:   "get <endpoint-or-name>",
	Short: "Get the details of a single device",
	Long: `Get the full device record: online status, group membership fields,
board name, OS, IP address, alarm status, location, etc.

获取单台设备的完整记录：在线状态、板卡名称、操作系统、IP地址、告警状态、
位置信息等。适合"这台设备现在是什么情况"这类总览查询；如果只需要CPU/
内存/存储/网卡等系统硬件信息，用 "device details" 更轻量。

Example 示例: eim-skill device get "146打包机"

Example output 输出示例 (trimmed 已截断):
  {
    "status": 200, "msg": "",
    "data": {
      "endpoint": "D8:CB:8A:0C:1E:49",
      "devName": "146打包机",
      "isonline": false,
      "boardName": "QiTianM4550-N000",
      "os": "linux",
      "deviceStatus": 0,
      "deviceStatusText": "normal"
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
		device, err := client.GetDevice(endpoint)
		if err != nil {
			return err
		}
		printJSON(device)
		return nil
	},
}

var deviceDetailsCmd = &cobra.Command{
	Use:   "details <endpoint-or-name>",
	Short: "Get a device's detailed system info (cpu, memory, storage, network interfaces, etc)",
	Long: `Get the agent-reported system info blob for a device: CPU model/cores,
memory size, storage layout, network interfaces/displays, BIOS info, etc.
This is a lighter-weight, hardware-focused call than "device get" (which
returns the full device record - online status, group membership, etc).

获取设备agent上报的系统硬件信息：CPU型号/核数、内存大小、存储布局、
网卡/显示器信息、BIOS信息等。相比 "device get"（返回完整设备记录，含
在线状态、分组等），这个命令更轻量，只关注硬件本身。

Not documented in AppHub-v2.3.0.md (found in ClientServlet's
get_1_details). 未在 AppHub-v2.3.0.md 中记录，是从 ClientServlet 的
get_1_details 接口反推得到。

Example 示例: eim-skill device details "08:00:27:33:EC:2E"

Example output 输出示例 (trimmed 已截断):
  {
    "status": 200, "msg": "",
    "data": {
      "cpu": "Intel(R) Core(TM) i7-7700 CPU @ 3.60GHz",
      "cores": 2,
      "arch": "amd64",
      "board_name": "VirtualBox",
      "agentversion": "3.0.5"
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
		details, err := client.GetDeviceDetails(endpoint)
		if err != nil {
			return err
		}
		printRawJSON(details)
		return nil
	},
}

var deviceDeleteCmd = &cobra.Command{
	Use:   "delete <endpoint-or-name>",
	Short: "Delete a device",
	Long: `Permanently remove a device's record from AppHub. This does not affect
the physical device or its agent software - it only removes AppHub's
record of it; if the agent is still running and checks in again, the
device will typically re-register.

从 AppHub 中永久删除该设备的记录。此操作不影响设备本身或其上运行的
agent 软件——只是删掉 AppHub 里的这条记录；如果设备的 agent 仍在运行
并再次上线签到，通常会重新注册出现。

Example 示例: eim-skill device delete "08:00:27:33:EC:2E"

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
		if err := client.DeleteDevice(endpoint); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var deviceMonitorCmd = &cobra.Command{
	Use:   "monitor <endpoint-or-name> --type cpu|memory",
	Short: "Get CPU or memory monitoring data for a device",
	Long: `Get historical CPU or memory usage samples for a device. --type is
required; --last optionally restricts the window to a recent relative
period (e.g. "30m", "12h", "3d") instead of returning every stored sample.

获取设备的 CPU 或内存历史监控数据。--type 必填；--last 可选，用来只看
最近一段时间的数据（如 "30m"=30分钟、"12h"=12小时、"3d"=3天），不填则
返回所有已保存的采样点。

Example 示例: eim-skill device monitor "146打包机" --type cpu --last 12h

Example output 输出示例 (empty result when no samples in that window
如果该时间窗口内没有数据会返回空数组):
  {"status": 200, "msg": "", "data": []}

  or, with samples 或有数据时:
  {"status": 200, "msg": "", "data": [{"id": 1, "value": 30.2, "ts": 1784011181000}]}`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dataType, _ := cmd.Flags().GetString("type")
		last, _ := cmd.Flags().GetString("last")
		begin, end, err := lastDurationWindow(last)
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
		points, err := client.MonitorData(endpoint, dataType, begin, end)
		if err != nil {
			return err
		}
		printJSON(points)
		return nil
	},
}

var deviceAddToGroupCmd = &cobra.Command{
	Use:   "add-to-group <endpoint-or-name> <group-name>",
	Short: "Add a device to a device group (keeps its existing group memberships)",
	Long: `Add a device to group-name. This is additive - the device keeps every
group it already belonged to; it does not move the device out of any
other group.

把设备加入指定的设备组。这是"追加"操作——设备原本所在的其他组不会被
移除，只是多加入了一个组。

Example 示例: eim-skill device add-to-group "08:00:27:33:EC:2E" default

Example output 输出示例:
  {"status": 200, "msg": "", "data": {"status": "CHANGED", "groups": ["default"]}}`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}

		groups, err := client.AddDeviceToGroup(endpoint, args[1])
		if err != nil {
			return err
		}
		printJSON(map[string]interface{}{
			"status": "CHANGED",
			"groups": groups,
		})
		return nil
	},
}

var deviceRebootCmd = &cobra.Command{
	Use:   "reboot <endpoint-or-name>",
	Short: "Reboot a device immediately",
	Long: `Reboot a single device right now, no confirmation/delay. Documented in
AppHub-v2.3.0.md ("PUT reboot-设备立即重启").

立即重启单台设备，没有二次确认或延迟。该接口在 AppHub-v2.3.0.md 中有
文档记录（"PUT reboot-设备立即重启"）。

Example 示例: eim-skill device reboot "146打包机"

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
		if err := client.Reboot(endpoint); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var deviceShutdownCmd = &cobra.Command{
	Use:   "shutdown <endpoint-or-name>",
	Short: "Shut down a device immediately",
	Long: `Shut down a single device right now, no confirmation/delay.

立即关闭单台设备，没有二次确认或延迟。关机后设备需要人工/远程电源
管理手段才能重新开机，请谨慎在自动化流程中使用。

Example 示例: eim-skill device shutdown "146打包机"

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
		if err := client.Shutdown(endpoint); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var deviceSetVolumeCmd = &cobra.Command{
	Use:   "set-volume <endpoint-or-name> <0-100>",
	Short: "Set a device's system audio volume (best effort, undocumented)",
	Long: `Set a device's system audio volume, 0 (mute) to 100 (max).

设置设备的系统音量，取值 0（静音）到 100（最大）。

UNVERIFIED: not documented in AppHub-v2.3.0.md; inferred from DeviceLink's
PluginsServlet source (the "audio" devicecontrol action). Verify against
your device firmware before relying on it in production.

未在 AppHub-v2.3.0.md 中记录，是根据 DeviceLink 的 PluginsServlet 源码
（"audio" devicecontrol 动作）推断实现，尚未在真实设备固件上验证过，
生产环境使用前请自行验证效果。

Example 示例: eim-skill device set-volume "146打包机" 50

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED"}}`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		volume, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("volume must be an integer 0-100: %w", err)
		}

		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		if err := client.SetVolume(endpoint, volume); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var deviceSetBacklightCmd = &cobra.Command{
	Use:   "set-backlight <endpoint-or-name> <0-100>",
	Short: "Set a device's screen backlight/brightness (best effort, undocumented)",
	Long: `Set a device's screen backlight/brightness, 0 (dimmest) to 100 (brightest).

设置设备的屏幕背光/亮度，取值 0（最暗）到 100（最亮）。

UNVERIFIED: not documented in AppHub-v2.3.0.md; inferred from DeviceLink's
PluginsServlet source (the "brightness" devicecontrol action). Verify
against your device firmware before relying on it in production.

未在 AppHub-v2.3.0.md 中记录，是根据 DeviceLink 的 PluginsServlet 源码
（"brightness" devicecontrol 动作）推断实现，尚未在真实设备固件上验证
过，生产环境使用前请自行验证效果。

Example 示例: eim-skill device set-backlight "146打包机" 80

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED"}}`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		brightness, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("brightness must be an integer 0-100: %w", err)
		}

		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		if err := client.SetBacklight(endpoint, brightness); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var deviceRebootGroupCmd = &cobra.Command{
	Use:   "reboot-group <group-name>",
	Short: "Reboot every device in a device group immediately",
	Long: `Reboot every device in group-name in a single call (documented in
AppHub-v2.3.0.md as "PUT send command-向批量设备发送命令",
PUT /api/plugins/batch/{groupname}/40008/0/27601), rather than looping
"device reboot" over each device individually. --label optionally narrows
the target to devices in the group that also carry that label.

一次性重启整个设备组下的所有设备（对应 AppHub-v2.3.0.md 文档中的
"PUT send command-向批量设备发送命令" 接口），而不是对组内每台设备
逐个调用 "device reboot"。--label 可选，用来只对组内带有该标签的设备
生效，不填则对整组生效。

Example 示例: eim-skill device reboot-group default --label edge-line-1

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED"}}`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		label, _ := cmd.Flags().GetString("label")
		client, err := newClient()
		if err != nil {
			return err
		}
		if err := client.RebootGroup(args[0], label); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var deviceShutdownGroupCmd = &cobra.Command{
	Use:   "shutdown-group <group-name>",
	Short: "Shut down every device in a device group immediately",
	Long: `Shut down every device in group-name in a single call, using the same
group-batch devicecontrol resource as "device reboot-group". --label
optionally narrows the target to devices in the group that also carry
that label.

一次性关闭整个设备组下的所有设备，使用和 "device reboot-group" 相同的
批量设备控制接口。--label 可选，用来只对组内带有该标签的设备生效。

Example 示例: eim-skill device shutdown-group default

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED"}}`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		label, _ := cmd.Flags().GetString("label")
		client, err := newClient()
		if err != nil {
			return err
		}
		if err := client.ShutdownGroup(args[0], label); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var deviceSetVolumeGroupCmd = &cobra.Command{
	Use:   "set-volume-group <group-name> <0-100>",
	Short: "Set the system audio volume on every device in a device group (best effort, undocumented)",
	Long: `Set the system audio volume (0-100) on every device in group-name in a
single call. --label optionally narrows the target to devices in the
group that also carry that label. Same undocumented/best-effort caveat as
"device set-volume" applies.

一次性设置整个设备组下所有设备的系统音量（0-100）。--label 可选，用来
只对组内带有该标签的设备生效。和 "device set-volume" 一样，这是未文档化
的尽力而为实现，使用前请自行验证。

Example 示例: eim-skill device set-volume-group default 50

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED"}}`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		volume, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("volume must be an integer 0-100: %w", err)
		}
		label, _ := cmd.Flags().GetString("label")

		client, err := newClient()
		if err != nil {
			return err
		}
		if err := client.SetVolumeGroup(args[0], volume, label); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var deviceSetBacklightGroupCmd = &cobra.Command{
	Use:   "set-backlight-group <group-name> <0-100>",
	Short: "Set the screen backlight/brightness on every device in a device group (best effort, undocumented)",
	Long: `Set the screen backlight/brightness (0-100) on every device in
group-name in a single call. --label optionally narrows the target to
devices in the group that also carry that label. Same
undocumented/best-effort caveat as "device set-backlight" applies.

一次性设置整个设备组下所有设备的屏幕背光/亮度（0-100）。--label 可选，
用来只对组内带有该标签的设备生效。和 "device set-backlight" 一样，这是
未文档化的尽力而为实现，使用前请自行验证。

Example 示例: eim-skill device set-backlight-group default 80

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED"}}`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		brightness, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("brightness must be an integer 0-100: %w", err)
		}
		label, _ := cmd.Flags().GetString("label")

		client, err := newClient()
		if err != nil {
			return err
		}
		if err := client.SetBacklightGroup(args[0], brightness, label); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}
