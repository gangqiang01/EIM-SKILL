package cmd

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

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
check whether it's currently on ("get-monitor"), and query/delete the
entries the server has collected ("list"/"delete-by-device"). A device
needs "set-monitor ... on" run against it at least once before "list"
will return anything for it.

开启或关闭某台设备的系统日志采集/上报（"set-monitor"），查询当前是否
开启（"get-monitor"），以及查询/删除服务器已采集到的日志条目
（"list"/"delete-by-device"）。需要先对目标设备执行过一次
"set-monitor ... on"，"list" 才会有该设备的数据。`,
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

var logAgentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Download a device's agent (client-side) log",
	Long: `Download a device's agent log - the DeviceOn-EIM client process log
running on the device itself. This is a different log from the one "log
system" covers (that's the device's own systemd/OS log, forwarded to and
queried from the server).

下载设备的 agent（客户端）日志——运行在设备本身上的 DeviceOn-EIM 客户端
进程日志。这与 "log system" 覆盖的日志不同（那是设备自身的
systemd/操作系统日志，上报到服务器后再查询）。`,
}

var logAgentDownloadCmd = &cobra.Command{
	Use:   "download <endpoint-or-name>",
	Short: "Download a device's agent log to a file",
	Long: `Download a device's agent log and save it to a file. The downloaded file
is a zip archive containing one log file per day plus the current day's
log - confirmed against a live server (the endpoint sends no
Content-Type header, but the body is reliably a zip archive by its
magic bytes).

This is a three-step, undocumented (in AppHub-v2.3.0.md) process confirmed
against a live server via a real packet capture: first the device is told
(via a plain LwM2M write to resource 40001/0/27012) to push its agent log
to this server, tagged with a fresh actionId; then
/api/devicemgr/logStatus is polled (every 2 seconds, up to a 3-minute
timeout) until it reports that actionId as "success" - meaning the push
has arrived; finally the log itself is fetched and saved. If the upload
doesn't complete within 3 minutes, this command exits with an error -
retrying may still succeed (e.g. if the device was briefly slow or
offline).

下载设备的 agent 日志并保存成文件。下载下来的是一个 zip 压缩包，里面
包含按天分割的日志文件和当天的日志——已用真实服务器验证过（该接口不
返回 Content-Type 响应头，但通过文件头字节确认内容始终是 zip 格式）。

这是一个未在 AppHub-v2.3.0.md 中记录的三步流程，已通过真实抓包在真实
服务器上验证过：先通过一次普通的 LwM2M 写操作（资源 40001/0/27012）
通知设备把 agent 日志推送到本服务器，并带上一个新生成的 actionId；然后
轮询 /api/devicemgr/logStatus（每2秒一次，最多3分钟超时）直到返回该
actionId 对应的状态为 "success"（说明推送已到达）；最后获取并保存日志
内容。如果3分钟内推送未完成，本命令会报错退出——重试仍有可能成功
（例如设备当时短暂离线或响应较慢）。

With --content, instead of (or alongside, if --output is also given) saving
the zip, this extracts it and returns the tail of the most recent day's
log file (preferring "EIM-Client.log", the currently-active log, when
present) as plain text in the JSON response - --lines controls how many
trailing lines (default 200), since a full day's log can run several MB
and isn't practical to return whole.

Example 示例: eim-skill log agent download "146打包机"
       eim-skill log agent download "146打包机" --output ./logs/146.zip
       eim-skill log agent download "146打包机" --content --lines 50

Example output 输出示例: {"status": 200, "msg": "", "data": {"endpoint": "C4:00:AD:11:DA:97", "file": "C400AD11DA97-agent-log.zip"}}`,
	Args: requireArgs(cobra.ExactArgs(1)),
	RunE: func(cmd *cobra.Command, args []string) error {
		output, _ := cmd.Flags().GetString("output")
		outputGiven := cmd.Flags().Changed("output")
		showContent, _ := cmd.Flags().GetBool("content")
		lines, _ := cmd.Flags().GetInt("lines")
		if showContent && lines <= 0 {
			return fmt.Errorf("--lines must be a positive number")
		}

		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}

		data, _, err := client.DownloadAgentLog(endpoint)
		if err != nil {
			return err
		}

		result := map[string]interface{}{"endpoint": endpoint}

		if showContent {
			content, logFile, err := tailLatestAgentLogEntry(data, lines)
			if err != nil {
				return fmt.Errorf("extract agent log content: %w", err)
			}
			result["logFile"] = logFile
			result["content"] = content
		}

		if !showContent || outputGiven {
			if output == "" {
				output = strings.ReplaceAll(endpoint, ":", "") + "-agent-log.zip"
			}
			if err := os.WriteFile(output, data, 0o644); err != nil {
				return fmt.Errorf("write %s: %w", output, err)
			}
			result["file"] = output
		}

		printJSON(result)
		return nil
	},
}

var logAgentFileCmd = &cobra.Command{
	Use:   "file <endpoint-or-name> <remote-path>",
	Short: "Fetch one specific file from a device's filesystem, by absolute path",
	Long: `Fetch one specific file from a device's filesystem (remote-path must be
an absolute path on the device, e.g. "/media/recovery/EIM-Client/20260721.log"),
returned as the single file it contains - unlike "log agent download"
(which bundles many days' worth of the whole agent log), the server only
ever puts one file in this archive, so its content is returned directly
by default.

This is a three-step, undocumented (in AppHub-v2.3.0.md) process confirmed
against a live server via a real packet capture, sharing "log agent
download"'s trigger/poll/fetch shape but against a different LwM2M
resource (40001/0/27023, which takes the file path directly) and a
different download endpoint (/api/devicemgr/log, not
/api/devicemgr/agentLog).

By default the file's content is printed directly (as text - use
--output to save the raw bytes instead if it isn't text, or to also keep
a copy). --lines, if given, returns only the trailing N lines instead of
the whole file - useful for a large file where you only care about the
most recent entries.

获取设备文件系统上的某一个具体文件（remote-path 必须是设备上的绝对
路径，如 "/media/recovery/EIM-Client/20260721.log"），并返回其中唯一
的一个文件——和 "log agent download"（把好几天的完整 agent 日志打包在
一起）不同，服务器这里返回的压缩包里始终只有一个文件，所以默认直接
返回其内容。

这是一个未在 AppHub-v2.3.0.md 中记录的三步流程，已通过真实抓包在真实
服务器上验证过，流程结构与 "log agent download" 相同（触发/轮询/获取），
但用的是不同的 LwM2M 资源（40001/0/27023，直接指定文件路径）和不同的
下载接口（/api/devicemgr/log，而不是 /api/devicemgr/agentLog）。

默认直接打印文件内容（按文本处理——如果不是文本，或者除了打印之外
还想保留一份文件，请用 --output 保存原始字节）。如果指定了
--lines，则只返回文件末尾N行，而不是整个文件——适合只关心最近内容的
大文件。

Example 示例: eim-skill log agent file "146打包机" /media/recovery/EIM-Client/20260721.log
       eim-skill log agent file "146打包机" /media/recovery/EIM-Client/20260721.log --lines 100
       eim-skill log agent file "146打包机" /var/log/messages --output ./messages.log

Example output 输出示例: {"status": 200, "msg": "", "data": {"endpoint": "C4:00:AD:11:DA:97", "remotePath": "/media/recovery/EIM-Client/20260721.log", "entryName": "20260721.log", "content": "..."}}`,
	Args: requireArgs(cobra.ExactArgs(2)),
	RunE: func(cmd *cobra.Command, args []string) error {
		remotePath := args[1]
		output, _ := cmd.Flags().GetString("output")
		lines, _ := cmd.Flags().GetInt("lines")
		if cmd.Flags().Changed("lines") && lines <= 0 {
			return fmt.Errorf("--lines must be a positive number")
		}

		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}

		zipData, _, err := client.DownloadDeviceFile(endpoint, remotePath)
		if err != nil {
			return err
		}

		fileData, entryName, err := singleZipEntry(zipData)
		if err != nil {
			return fmt.Errorf("extract fetched file: %w", err)
		}

		result := map[string]interface{}{
			"endpoint":   endpoint,
			"remotePath": remotePath,
			"entryName":  entryName,
		}

		if output != "" {
			if err := os.WriteFile(output, fileData, 0o644); err != nil {
				return fmt.Errorf("write %s: %w", output, err)
			}
			result["file"] = output
		} else {
			if !isProbablyText(fileData) {
				return fmt.Errorf("%q doesn't look like text - use --output to save it as a raw file instead", remotePath)
			}
			content := string(fileData)
			if lines > 0 {
				content = tailLines(content, lines)
			}
			result["content"] = content
		}

		printJSON(result)
		return nil
	},
}

// singleZipEntry extracts the one file DownloadDeviceFile's zip archive is
// expected to contain, returning its raw bytes and entry name. Errors if
// the archive doesn't have exactly one entry - confirmed against a live
// server that it always does, but this fails loudly instead of silently
// guessing if that ever changes.
func singleZipEntry(zipData []byte) ([]byte, string, error) {
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, "", fmt.Errorf("not a valid zip archive: %w", err)
	}
	if len(zr.File) != 1 {
		return nil, "", fmt.Errorf("expected exactly one file in the archive, got %d", len(zr.File))
	}

	rc, err := zr.File[0].Open()
	if err != nil {
		return nil, "", err
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, "", err
	}
	return data, zr.File[0].Name, nil
}

// isProbablyText reports whether data looks like text rather than binary
// - a null byte anywhere is treated as a reliable sign of binary content.
func isProbablyText(data []byte) bool {
	return !bytes.ContainsRune(data, 0)
}

// tailLatestAgentLogEntry extracts the most recent log file from an agent
// log zip archive (see DownloadAgentLog) and returns the last n lines of
// its text, along with the entry's name. "Most recent" prefers an entry
// named exactly "EIM-Client.log" - the currently-active log, confirmed
// live to duplicate that day's dated file - falling back to the
// lexicographically greatest entry name otherwise (dated entries observed
// live are named "YYYYMMDD.log", so this also picks the latest date).
func tailLatestAgentLogEntry(zipData []byte, n int) (content, entryName string, err error) {
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return "", "", fmt.Errorf("not a valid zip archive: %w", err)
	}
	if len(zr.File) == 0 {
		return "", "", fmt.Errorf("zip archive is empty")
	}

	var latest *zip.File
	for _, f := range zr.File {
		if f.Name == "EIM-Client.log" {
			latest = f
			break
		}
		if latest == nil || f.Name > latest.Name {
			latest = f
		}
	}

	rc, err := latest.Open()
	if err != nil {
		return "", "", err
	}
	defer rc.Close()
	raw, err := io.ReadAll(rc)
	if err != nil {
		return "", "", err
	}

	return tailLines(string(raw), n), latest.Name, nil
}

// tailLines returns the last n lines of s (or all of s if it has n or fewer).
func tailLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

func init() {
	rootCmd.AddCommand(logCmd)
	logCmd.AddCommand(logSystemCmd)
	logCmd.AddCommand(logOperationsCmd)
	logCmd.AddCommand(logAgentCmd)

	logSystemCmd.AddCommand(logSystemSetMonitorCmd)
	logSystemCmd.AddCommand(logSystemGetMonitorCmd)
	logSystemCmd.AddCommand(logSystemListCmd)
	logSystemCmd.AddCommand(logSystemDeleteByDeviceCmd)

	logOperationsCmd.AddCommand(logOperationsListCmd)
	logOperationsCmd.AddCommand(logOperationsDeleteAllCmd)

	logAgentCmd.AddCommand(logAgentDownloadCmd)
	logAgentDownloadCmd.Flags().String("output", "", "file path to save the downloaded zip archive to (default: \"<endpoint>-agent-log.zip\" in the current directory; skipped if --content is given without --output) (保存下载的zip压缩包的文件路径，默认为当前目录下的 \"<endpoint>-agent-log.zip\"——如果只给了 --content 而没给 --output，则不保存文件)")
	logAgentDownloadCmd.Flags().Bool("content", false, "return the most recent day's log content directly (tail of it - see --lines) instead of only saving a file (直接返回最新一天日志的内容（末尾部分，见 --lines），而不只是保存文件)")
	logAgentDownloadCmd.Flags().Int("lines", 200, "with --content, how many trailing lines of the log to return (配合 --content 使用，返回日志末尾多少行)")

	logAgentCmd.AddCommand(logAgentFileCmd)
	logAgentFileCmd.Flags().String("output", "", "save the raw file to this path instead of printing its content (把原始文件保存到这个路径，而不是打印内容)")
	logAgentFileCmd.Flags().Int("lines", 0, "only return the trailing N lines of the file's content (default: the whole file) (只返回文件末尾N行内容，默认返回整个文件)")

	logSystemListCmd.Flags().Int("limit", 20, "how many of the most recent entries to return (返回最近多少条记录)")
	logSystemListCmd.Flags().String("device-id", "", "only return entries for this device, endpoint or device name (只返回该设备的记录，可用endpoint或设备名)")
	logSystemListCmd.Flags().String("os", "", "only return entries for this device OS, e.g. \"linux\" (只返回该操作系统的记录，如\"linux\")")
	logSystemListCmd.Flags().String("keywords", "", "filter by keywords, substring match (按关键字做包含匹配过滤)")
	logSystemListCmd.Flags().String("mode", "", "filter by log mode, e.g. \"Systemdlog\" (按日志mode过滤，如\"Systemdlog\")")
	logSystemListCmd.Flags().String("level", "", "filter by log level (按日志级别过滤)")
	addTimeWindowFlags(logSystemListCmd)

	logOperationsListCmd.Flags().Int("limit", 20, "how many of the most recent entries to return (返回最近多少条记录)")
	logOperationsListCmd.Flags().String("target", "", "filter by target, substring match (按目标做包含匹配过滤，如设备endpoint或名称)")
	addTimeWindowFlags(logOperationsListCmd)
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
		if err := client.SetSystemLogMonitorEnabled(endpoint, enabled); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var logSystemGetMonitorCmd = &cobra.Command{
	Use:   "get-monitor <endpoint-or-name>",
	Short: "Report whether system log collection/reporting is enabled",
	Long: `Report whether system log collection/reporting is currently enabled for
a device (see "log system set-monitor"). If it's off, "log system list"
will have nothing new to show for that device going forward.

查询设备当前是否开启了系统日志采集/上报（见 "log system
set-monitor"）。如果是关闭的，"log system list" 对这台设备之后就不会
再有新数据。

Not documented in AppHub-v2.3.0.md; uses the "get_monitor_data_report"
devicecontrol action (the same action "device get-monitor-report" uses
for CPU/memory data collection, just reading a different key from the
same response), confirmed against a live server.

未在 AppHub-v2.3.0.md 中记录；使用 "get_monitor_data_report" 设备控制
动作（和 "device get-monitor-report" 查询CPU/内存数据采集用的是同一个
动作，只是从同一个返回结果里读取了不同的字段），已用真实服务器
验证过。

Example 示例: eim-skill log system get-monitor "08:00:27:33:EC:2E"

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
		enabled, err := client.GetSystemLogMonitorEnabled(endpoint)
		if err != nil {
			return err
		}
		printJSON(map[string]bool{"enabled": enabled})
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

"level" and "pid" are normalized to real JSON numbers - the server
reports both as numeric strings (e.g. "level":"4"), confirmed against a
live server via a real packet capture.

"level" 和 "pid" 已归一化成真正的JSON数字——服务器返回的是数字字符串
（如 "level":"4"），已通过真实抓包在真实服务器上验证过。

Restrict the time range with either --last (a recent relative period,
e.g. "30m", "12h", "3d") or --begin/--end (an absolute range, RFC3339 or
"YYYY-MM-DD HH:MM" - either or both may be given; mutually exclusive with
--last).

可以用 --last（最近一段相对时间，如 "30m"/"12h"/"3d"）或者
--begin/--end（绝对时间区间，RFC3339 或 "YYYY-MM-DD HH:MM"——可以只填
一个也可以两个都填；不能和 --last 同时使用）来限定时间范围。

Example 示例: eim-skill log system list --device-id C4:00:AD:11:DA:97 --limit 10
       eim-skill log system list --begin "2026-07-01 00:00" --end "2026-07-02 00:00"

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
        "level": 4,
        "mode": "Systemdlog",
        "name": "systemd-resolved",
        "pid": 459,
        "eventId": "",
        "ts": 1784255095392
      }
    ]
  }`,
	Args: requireArgs(cobra.NoArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		deviceID, _ := cmd.Flags().GetString("device-id")
		osFilter, _ := cmd.Flags().GetString("os")
		keywords, _ := cmd.Flags().GetString("keywords")
		mode, _ := cmd.Flags().GetString("mode")
		level, _ := cmd.Flags().GetString("level")
		begin, end, err := resolveTimeWindow(cmd)
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

Restrict the time range with either --last (a recent relative period,
e.g. "30m", "12h", "3d") or --begin/--end (an absolute range, RFC3339 or
"YYYY-MM-DD HH:MM" - either or both may be given; mutually exclusive with
--last).

可以用 --last（最近一段相对时间，如 "30m"/"12h"/"3d"）或者
--begin/--end（绝对时间区间，RFC3339 或 "YYYY-MM-DD HH:MM"——可以只填
一个也可以两个都填；不能和 --last 同时使用）来限定时间范围。

Example 示例: eim-skill log operations list --limit 3 --last 7d
       eim-skill log operations list --begin "2026-07-01 00:00" --end "2026-07-02 00:00"

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
	Args: requireArgs(cobra.NoArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		target, _ := cmd.Flags().GetString("target")
		begin, end, err := resolveTimeWindow(cmd)
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
	Args: requireArgs(cobra.NoArgs),
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
