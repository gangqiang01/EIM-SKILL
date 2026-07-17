package cmd

import (
	"github.com/spf13/cobra"
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Device system logs and operation records (audit log)",
	Long: `Device system log collection/query/delete, and operation records (an
audit log of admin actions).

设备系统日志的采集开关/查询/删除，以及操作记录（管理员操作审计日志）
查询。`,
}

var logSystemCmd = &cobra.Command{
	Use:   "system",
	Short: "Turn a device's system log collection on/off, and query/delete collected entries",
	Long: `Turn a device's system log collection/reporting on or off ("set-monitor"),
and query/delete the entries the server has collected ("list"/
"delete-by-device"). A device needs "set-monitor ... on" run against it at
least once before "list" will return anything for it.

开启或关闭某台设备的系统日志采集/上报（"set-monitor"），以及查询/删除
服务器已采集到的日志条目（"list"/"delete-by-device"）。需要先对目标
设备执行过一次 "set-monitor ... on"，"list" 才会有该设备的数据。`,
}

var logOperationsCmd = &cobra.Command{
	Use:   "operations",
	Short: "Query and delete operation records (an audit log of admin actions)",
	Long: `Query and delete operation records: an audit trail of admin actions
performed against DeviceOn-EIM (who did what, to which device/target, and
when). Not documented in AppHub-v2.3.0.md - reverse-engineered from
OperationRecordServlet.java and confirmed against a live server.

查询和删除操作记录：记录了针对 DeviceOn-EIM 执行过的管理员操作审计
轨迹（谁在什么时候对哪个设备/目标做了什么操作）。未在 AppHub-v2.3.0.md
中记录，是从 OperationRecordServlet.java 反推并已用真实服务器验证过
的功能。`,
}

func init() {
	rootCmd.AddCommand(logCmd)
	logCmd.AddCommand(logSystemCmd)
	logCmd.AddCommand(logOperationsCmd)

	logSystemCmd.AddCommand(logSystemSetMonitorCmd)
	logSystemCmd.AddCommand(logSystemListCmd)
	logSystemCmd.AddCommand(logSystemDeleteByDeviceCmd)

	logOperationsCmd.AddCommand(logOperationsListCmd)
	logOperationsCmd.AddCommand(logOperationsDeleteAllCmd)

	logSystemListCmd.Flags().Int("limit", 20, "how many of the most recent entries to return (返回最近多少条记录)")
	logSystemListCmd.Flags().String("device-id", "", "only return entries for this device, endpoint or device name (只返回该设备的记录，可用endpoint或设备名)")
	logSystemListCmd.Flags().String("os", "", "only return entries for this device OS, e.g. \"linux\" (只返回该操作系统的记录，如\"linux\")")
	logSystemListCmd.Flags().String("keywords", "", "filter by keywords, substring match (按关键字做包含匹配过滤)")
	logSystemListCmd.Flags().String("mode", "", "filter by log mode, e.g. \"Systemdlog\" (按日志mode过滤，如\"Systemdlog\")")
	logSystemListCmd.Flags().String("level", "", "filter by log level (按日志级别过滤)")
	logSystemListCmd.Flags().String("last", "", `only return entries from the last period, e.g. "30m", "12h", "3d" (default: no time filter) - 只返回最近一段时间内的记录，如 "30m"(30分钟)/"12h"(12小时)/"3d"(3天)，不填则不限制时间范围`)

	logOperationsListCmd.Flags().Int("limit", 20, "how many of the most recent entries to return (返回最近多少条记录)")
	logOperationsListCmd.Flags().String("target", "", "filter by target, substring match (按目标做包含匹配过滤，如设备endpoint或名称)")
	logOperationsListCmd.Flags().String("last", "", `only return entries from the last period, e.g. "30m", "12h", "3d" (default: no time filter) - 只返回最近一段时间内的记录，如 "30m"(30分钟)/"12h"(12小时)/"3d"(3天)，不填则不限制时间范围`)
}

var logSystemSetMonitorCmd = &cobra.Command{
	Use:   "set-monitor <endpoint-or-name> <on|off>",
	Short: "Turn system log collection/reporting on or off for a device",
	Long: `Turn a device's system log collection/reporting on or off.

开启或关闭设备的系统日志采集/上报。

Not documented in AppHub-v2.3.0.md (inferred from a captured devicecontrol
request using the "set_monitor_data_report"/"systemd_log_monitor"
action) - confirmed to succeed against a live device (visible afterward in
"log operations list"'s audit trail). A device needs this turned on before
"log system list" will return anything for it.

未在 AppHub-v2.3.0.md 中记录（通过抓包分析 devicecontrol 流量中
"set_monitor_data_report"/"systemd_log_monitor" 动作得到）——已在真实
设备上验证执行成功（执行后能在 "log operations list" 的审计记录里
看到）。需要先对目标设备打开这个开关，"log system list" 才会有该设备
的数据。

Example 示例: eim-skill log system set-monitor "146打包机" on

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
		if err := client.SetSystemLogMonitorEnabled(endpoint, enabled); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var logSystemListCmd = &cobra.Command{
	Use:   "list",
	Short: "List a device's collected system log entries",
	Long: `List system log entries the server has collected from devices (requires
"log system set-monitor <device> on" to have been run against a device at
least once - otherwise it has nothing to report).

Not documented in AppHub-v2.3.0.md - reverse-engineered from a real packet
capture. The endpoint path is /api/systemLog/byPage (capital P); an
earlier version of this CLI used the lowercase /api/systemLog/bypage,
which the server rejects with an HTTP 400 "Invalid path" error - that
mistake, not a genuine server limitation, is why this command was
previously removed.

列出服务器已采集到的设备系统日志条目（需要先对目标设备执行过一次
"log system set-monitor <device> on"，否则该设备不会有任何数据）。

未在 AppHub-v2.3.0.md 中记录——通过真实抓包反推得到。接口路径是
/api/systemLog/byPage（大写P）；本CLI早期版本用的是小写的
/api/systemLog/bypage，会被服务器返回HTTP 400 "Invalid path" 错误——
这是我们自己的错误，并非服务器本身的限制，此前因此错误地移除了这条
命令，现已恢复并修正。

Example 示例: eim-skill log system list --device-id C4:00:AD:11:DA:97 --limit 10

Example output 输出示例 (real 真实返回):
  {
    "status": 200, "msg": "",
    "data": [
      {
        "id": 109120,
        "deviceId": "C4:00:AD:11:DA:97",
        "devname": "Server84.172",
        "content": "Using degraded feature set (UDP) for DNS server 240e:f:a00b::6.",
        "os": "linux",
        "level": "4",
        "mode": "Systemdlog",
        "name": "systemd-resolved",
        "pid": "459",
        "eventId": "",
        "ts": 1784255095392
      }
    ]
  }`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		deviceID, _ := cmd.Flags().GetString("device-id")
		osFilter, _ := cmd.Flags().GetString("os")
		keywords, _ := cmd.Flags().GetString("keywords")
		mode, _ := cmd.Flags().GetString("mode")
		level, _ := cmd.Flags().GetString("level")
		last, _ := cmd.Flags().GetString("last")
		begin, end, err := lastDurationWindow(last)
		if err != nil {
			return err
		}

		client, err := newClient()
		if err != nil {
			return err
		}
		if deviceID != "" {
			deviceID, err = resolveDeviceEndpoint(client, deviceID)
			if err != nil {
				return err
			}
		}
		logs, err := client.ListSystemLogs(1, limit, deviceID, osFilter, keywords, mode, level, begin, end)
		if err != nil {
			return err
		}
		printJSON(logs)
		return nil
	},
}

var logSystemDeleteByDeviceCmd = &cobra.Command{
	Use:   "delete-by-device <endpoint-or-name>",
	Short: "Delete every system log entry for a device",
	Long: `Delete every collected system log entry for one device. This cannot be
undone.

删除某台设备的全部系统日志记录。此操作不可撤销，请谨慎使用。

Example 示例: eim-skill log system delete-by-device "146打包机"

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
		if err := client.DeleteSystemLogsByDevice(endpoint); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var logOperationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List operation records (audit log)",
	Long: `List admin operation records: who (username) did what (event - the raw
action payload sent), to what (target - typically "<endpoint>(<devName>)"),
and when (ts).

列出管理员操作记录：谁（username）做了什么（event，发送的原始动作
内容）、对什么目标（target，通常是"<endpoint>(<设备名>)"）、什么时候
（ts）。

Example 示例: eim-skill log operations list --limit 3 --last 7d

Example output 输出示例 (real 真实返回):
  {
    "status": 200, "msg": "",
    "data": [
      {
        "orid": 113,
        "type": "device",
        "event": "{\"set_monitor_data_report\":[{\"systemd_log_monitor\":true}]} ",
        "target": "08:00:27:33:EC:2E(08:00:27:33:EC:2E)",
        "username": "admin",
        "ts": 1784014625884
      }
    ]
  }`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		target, _ := cmd.Flags().GetString("target")
		last, _ := cmd.Flags().GetString("last")
		begin, end, err := lastDurationWindow(last)
		if err != nil {
			return err
		}

		client, err := newClient()
		if err != nil {
			return err
		}
		records, err := client.ListOperationRecords(1, limit, target, begin, end)
		if err != nil {
			return err
		}
		printJSON(records)
		return nil
	},
}

var logOperationsDeleteAllCmd = &cobra.Command{
	Use:   "delete-all",
	Short: "Delete every operation record",
	Long: `Delete every operation record - clears the entire audit log. This cannot
be undone.

删除所有操作记录——清空整个审计日志。此操作不可撤销，请谨慎使用。

Example 示例: eim-skill log operations delete-all

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED"}}`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		if err := client.DeleteAllOperationRecords(); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}
