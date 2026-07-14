package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Operation records (audit log) and agent (client) logs",
	Long: `Operation records (an audit log of admin actions) and device agent
(client) log retrieval.

操作记录（管理员操作审计日志）查询，以及设备agent（客户端）日志的
获取。`,
}

var logSystemCmd = &cobra.Command{
	Use:   "system",
	Short: "Turn a device's system log collection/reporting on or off",
	Long: `Turn a device's system log collection/reporting on or off. This only
configures whether the device collects/reports its own system log; this
CLI does not currently provide a way to query those collected entries (see
"log system set-monitor"'s help for why).

开启或关闭某台设备的系统日志采集/上报。这里只负责配置"设备是否采集
并上报系统日志"这个开关；本CLI目前不提供查询这些已采集日志的命令
（原因见 "log system set-monitor" 的详细说明）。`,
}

var logOperationsCmd = &cobra.Command{
	Use:   "operations",
	Short: "Query and delete operation records (an audit log of admin actions)",
	Long: `Query and delete operation records: an audit trail of admin actions
performed against AppHub (who did what, to which device/target, and
when). Not documented in AppHub-v2.3.0.md - reverse-engineered from
OperationRecordServlet.java and confirmed against a live server.

查询和删除操作记录：记录了针对 AppHub 执行过的管理员操作审计轨迹
（谁在什么时候对哪个设备/目标做了什么操作）。未在 AppHub-v2.3.0.md
中记录，是从 OperationRecordServlet.java 反推并已用真实服务器验证过
的功能。`,
}

var logAgentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Request, poll, and download a device agent's client log",
	Long: `Fetching an agent's log is asynchronous: the server can't push a file to
you directly, so it asks the device agent to upload its log to a callback
address you provide. That means --callback-url must be an HTTP address the
device can reach and that is actually running a server accepting the
agent's upload - this CLI does not provide one; run "log agent request",
wait, poll "log agent status" until it reports success, then
"log agent download".

获取agent日志是异步流程：服务器不能直接把文件推送给你，而是让设备的
agent把日志上传到你提供的回调地址。这意味着 --callback-url 必须是
设备能访问到的、并且真的在运行一个接受上传的HTTP服务的地址——本CLI
不提供这个服务。流程：先 "log agent request"，等待，轮询
"log agent status" 直到返回成功，再 "log agent download"。`,
}

func init() {
	rootCmd.AddCommand(logCmd)
	logCmd.AddCommand(logSystemCmd)
	logCmd.AddCommand(logOperationsCmd)
	logCmd.AddCommand(logAgentCmd)

	logSystemCmd.AddCommand(logSystemSetMonitorCmd)

	logOperationsCmd.AddCommand(logOperationsListCmd)
	logOperationsCmd.AddCommand(logOperationsDeleteCmd)
	logOperationsCmd.AddCommand(logOperationsDeleteAllCmd)
	logOperationsCmd.AddCommand(logOperationsBatchDeleteCmd)

	logAgentCmd.AddCommand(logAgentRequestCmd)
	logAgentCmd.AddCommand(logAgentStatusCmd)
	logAgentCmd.AddCommand(logAgentDownloadCmd)

	logOperationsListCmd.Flags().Int("limit", 20, "how many of the most recent entries to return (返回最近多少条记录)")
	logOperationsListCmd.Flags().String("target", "", "filter by target, substring match (按目标做包含匹配过滤，如设备endpoint或名称)")
	logOperationsListCmd.Flags().String("last", "", `only return entries from the last period, e.g. "30m", "12h", "3d" (default: no time filter) - 只返回最近一段时间内的记录，如 "30m"(30分钟)/"12h"(12小时)/"3d"(3天)，不填则不限制时间范围`)

	logAgentDownloadCmd.Flags().String("out", "", "file path to save the downloaded log to (default: <endpoint>.zip) - 保存下载日志的文件路径（默认：<endpoint>.zip）")
}

func parseInt64Args(args []string) ([]int64, error) {
	ids := make([]int64, 0, len(args))
	for _, a := range args {
		id, err := strconv.ParseInt(a, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%q is not an integer id: %w", a, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

var logSystemSetMonitorCmd = &cobra.Command{
	Use:   "set-monitor <endpoint-or-name> <on|off>",
	Short: "Turn system log collection/reporting on or off for a device",
	Long: `Turn a device's system log collection/reporting on or off.

开启或关闭设备的系统日志采集/上报。

Not documented in AppHub-v2.3.0.md (inferred from a captured devicecontrol
request using the "set_monitor_data_report"/"systemd_log_monitor"
action) - confirmed to succeed against a live device (visible afterward in
"log operations list"'s audit trail). There is currently no working "list"
command for querying the collected system log entries themselves: the
documented /api/systemLog/bypage endpoint returned an HTTP 400 "Invalid
path" error from a real AppHub server during testing (a server-side
routing issue, not a request-shape problem - the request matched
AppHub-v2.3.0.md exactly), so that command (and the delete/
delete-by-device/batch-delete commands that depended on it to discover
ids) was removed rather than shipped non-functional.

未在 AppHub-v2.3.0.md 中记录（通过抓包分析 devicecontrol 流量中
"set_monitor_data_report"/"systemd_log_monitor" 动作得到）——已在真实
设备上验证执行成功（执行后能在 "log operations list" 的审计记录里
看到）。目前本CLI没有可用的"查询已采集系统日志"命令：文档记录的
/api/systemLog/bypage 接口在真实 AppHub 服务器测试时返回了HTTP 400
"Invalid path" 错误（是服务端路由层面的问题，不是请求格式问题——
请求参数与 AppHub-v2.3.0.md 文档完全一致），因此这条命令（以及依赖它
来发现日志id的 delete/delete-by-device/batch-delete）已被整组移除，
而不是保留一个实际用不了的功能。

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

var logOperationsDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a single operation record",
	Long: `Delete one operation record by its id (the "orid" field from
"log operations list").

按id（"log operations list" 返回的 "orid" 字段）删除单条操作记录。

Example 示例: eim-skill log operations delete 113

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED"}}`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return err
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		if err := client.DeleteOperationRecord(id); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
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

var logOperationsBatchDeleteCmd = &cobra.Command{
	Use:   "batch-delete <id> [id...]",
	Short: "Delete multiple operation records",
	Long: `Delete multiple operation records by id in one call.

一次性按id删除多条操作记录。

Example 示例: eim-skill log operations batch-delete 113 114 115

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED"}}`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ids, err := parseInt64Args(args)
		if err != nil {
			return err
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		if err := client.BatchDeleteOperationRecords(ids); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var logAgentRequestCmd = &cobra.Command{
	Use:   "request <endpoint-or-name> <callback-url> <action-id>",
	Short: "Ask a device's agent to upload its client log to callback-url",
	Long: `Ask a device's agent to upload its client log to callback-url. action-id
should be a fresh, unique string you choose - use the same value with
"log agent status"/"log agent download" to track this specific request.

请求设备的agent把客户端日志上传到 callback-url。action-id 需要你自己
生成一个唯一字符串，之后用同一个值配合 "log agent status"/
"log agent download" 来追踪这次请求。

Example 示例: eim-skill log agent request "146打包机" http://192.168.1.10:9000/upload req-20260714-01

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED"}}`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		if err := client.RequestAgentLogUpload(endpoint, args[1], args[2]); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var logAgentStatusCmd = &cobra.Command{
	Use:   "status <action-id>",
	Short: "Poll the status of a previously requested agent log upload",
	Long: `Poll the status of a previously requested agent log upload (the same
action-id passed to "log agent request"). Keep polling until the inner
data.result field reports "success", then call "log agent download".

查询之前请求的agent日志上传进度（使用和 "log agent request" 相同的
action-id）。持续轮询，直到返回的 data.result 字段变成 "success"，
再调用 "log agent download"。

Don't confuse the outer envelope's "status" (200/500 - did the CLI call
itself succeed) with the inner data.result field ("running"/"success"/
"timeout"/"error" - whether the device's log upload finished).

注意区分外层信封的 "status"（200/500，表示这次CLI调用本身是否成功）
和内层 data.result 字段（"running"/"success"/"timeout"/"error"，
表示设备端日志上传本身的进度）。

Example 示例: eim-skill log agent status req-20260714-01`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		status, err := client.GetAgentLogStatus(args[0])
		if err != nil {
			return err
		}
		printJSON(status)
		return nil
	},
}

var logAgentDownloadCmd = &cobra.Command{
	Use:   "download <endpoint-or-name>",
	Short: "Download a device agent's uploaded client log (can only be downloaded once)",
	Long: `Download a device agent's previously-uploaded client log, saving it to
--out (default: "<endpoint>.zip" in the current directory, with ":"
stripped from the endpoint). Can only be downloaded once - the server
deletes the file after serving it, so re-running this after a successful
download will fail.

下载设备agent之前已上传的客户端日志，保存到 --out 指定路径（默认为
当前目录下的 "<endpoint>.zip"，endpoint 中的冒号会被去掉）。只能下载
一次——服务器在提供文件后会将其删除，成功下载后再次运行此命令会失败。

Example 示例: eim-skill log agent download "146打包机" --out ./146-agent.zip

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED", "file": "./146-agent.zip"}}`,
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

		out, _ := cmd.Flags().GetString("out")
		if out == "" {
			out = strings.ReplaceAll(endpoint, ":", "") + ".zip"
		}

		data, err := client.DownloadAgentLog(endpoint)
		if err != nil {
			return err
		}
		if err := os.WriteFile(out, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", out, err)
		}
		printJSON(map[string]string{"status": "CHANGED", "file": out})
		return nil
	},
}
