package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gangqiang01/EIM-SKILL/internal/apphub"
	"github.com/spf13/cobra"
)

var weekdayNames = map[string]time.Weekday{
	"sun": time.Sunday, "sunday": time.Sunday,
	"mon": time.Monday, "monday": time.Monday,
	"tue": time.Tuesday, "tuesday": time.Tuesday,
	"wed": time.Wednesday, "wednesday": time.Wednesday,
	"thu": time.Thursday, "thursday": time.Thursday,
	"fri": time.Friday, "friday": time.Friday,
	"sat": time.Saturday, "saturday": time.Saturday,
}

// parseAbsoluteTime parses an absolute time value (used by --at, --begin,
// --end), accepting RFC3339 or the simpler local-time forms
// "2006-01-02 15:04"/"2006-01-02T15:04". The error message doesn't name a
// specific flag - callers wrap it with their own flag name.
func parseAbsoluteTime(s string) (time.Time, error) {
	layouts := []string{time.RFC3339, "2006-01-02T15:04", "2006-01-02 15:04"}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf(`%q - expected RFC3339 (e.g. "2026-08-01T08:00:00+08:00") or "YYYY-MM-DD HH:MM" (e.g. "2026-08-01 08:00")`, s)
}

// parseClockTime parses "HH:MM" into hour/minute.
func parseClockTime(s string) (hour, minute int, err error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf(`invalid --time %q - expected "HH:MM"`, s)
	}
	hour, err = strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return 0, 0, fmt.Errorf(`invalid hour in --time %q - expected "HH:MM" with HH 0-23`, s)
	}
	minute, err = strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf(`invalid minute in --time %q - expected "HH:MM" with MM 0-59`, s)
	}
	return hour, minute, nil
}

// nextWeekdayAt returns the next occurrence (today, if it's the right
// weekday and hasn't passed yet) of weekday at hour:minute, local time.
func nextWeekdayAt(weekday time.Weekday, hour, minute int) time.Time {
	now := time.Now()
	daysUntil := (int(weekday) - int(now.Weekday()) + 7) % 7
	candidate := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, time.Local).AddDate(0, 0, daysUntil)
	if !candidate.After(now) {
		candidate = candidate.AddDate(0, 0, 7)
	}
	return candidate
}

// nextDayOfMonthAt returns the next occurrence (this month, if day-of-month
// dom exists in it and hasn't passed yet) of dom at hour:minute, local
// time - rolling forward to the next month that has that day otherwise
// (e.g. day 31 requested during a 30-day month).
func nextDayOfMonthAt(dom, hour, minute int) time.Time {
	now := time.Now()
	year, month := now.Year(), now.Month()
	for i := 0; i < 12; i++ {
		daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, time.Local).Day()
		if dom <= daysInMonth {
			candidate := time.Date(year, month, dom, hour, minute, 0, 0, time.Local)
			if candidate.After(now) {
				return candidate
			}
		}
		month++
		if month > time.December {
			month = time.January
			year++
		}
	}
	return now // unreachable for dom in 1-31, but never panic over a date calc
}

// resolveSchedule reads --at/--weekly/--monthly/--time/--cancel (exactly
// one of --at/--weekly/--monthly/--cancel must be given) and returns the
// (enabled, atMs, cycleDays) triple to pass to Client.ScheduleReboot/
// ScheduleShutdown.
func resolveSchedule(cmd *cobra.Command) (enabled bool, atMs int64, cycleDays int, err error) {
	at, _ := cmd.Flags().GetString("at")
	weekly, _ := cmd.Flags().GetString("weekly")
	monthly, _ := cmd.Flags().GetInt("monthly")
	clock, _ := cmd.Flags().GetString("time")
	cancel, _ := cmd.Flags().GetBool("cancel")

	modes := 0
	for _, set := range []bool{at != "", weekly != "", monthly != 0, cancel} {
		if set {
			modes++
		}
	}
	if modes == 0 {
		return false, 0, 0, fmt.Errorf("specify exactly one of --at, --weekly, --monthly, or --cancel")
	}
	if modes > 1 {
		return false, 0, 0, fmt.Errorf("--at, --weekly, --monthly, and --cancel are mutually exclusive")
	}

	switch {
	case cancel:
		return false, time.Now().UnixMilli(), 0, nil
	case at != "":
		t, err := parseAbsoluteTime(at)
		if err != nil {
			return false, 0, 0, fmt.Errorf("invalid --at %w", err)
		}
		return true, t.UnixMilli(), 0, nil
	case weekly != "":
		wd, ok := weekdayNames[strings.ToLower(weekly)]
		if !ok {
			return false, 0, 0, fmt.Errorf(`unrecognized --weekly %q - expected mon/tue/wed/thu/fri/sat/sun (or full name)`, weekly)
		}
		if clock == "" {
			return false, 0, 0, fmt.Errorf(`--weekly requires --time "HH:MM"`)
		}
		hour, minute, err := parseClockTime(clock)
		if err != nil {
			return false, 0, 0, err
		}
		return true, nextWeekdayAt(wd, hour, minute).UnixMilli(), 7, nil
	default: // monthly != 0
		if monthly < 1 || monthly > 31 {
			return false, 0, 0, fmt.Errorf("--monthly must be between 1 and 31, got %d", monthly)
		}
		if clock == "" {
			return false, 0, 0, fmt.Errorf(`--monthly requires --time "HH:MM"`)
		}
		hour, minute, err := parseClockTime(clock)
		if err != nil {
			return false, 0, 0, err
		}
		return true, nextDayOfMonthAt(monthly, hour, minute).UnixMilli(), 30, nil
	}
}

const scheduleFlagsLongEnglish = `--at schedules a one-time run at an absolute date/time (local time zone),
either RFC3339 (e.g. "2026-08-01T08:00:00+08:00") or "YYYY-MM-DD HH:MM"
(e.g. "2026-08-01 08:00"). --weekly <mon..sun> --time "HH:MM" repeats
weekly on that day at that time. --monthly <1-31> --time "HH:MM" repeats
on that day of the month at that time - see the caveat below. --cancel
disarms any pending schedule instead of creating one. Exactly one of
--at/--weekly/--monthly/--cancel must be given.

Not documented in AppHub-v2.3.0.md for a single device - only the
batch/group variant is. Confirmed against a live server: the write
returns {"status":"CHANGED"} (matching the documented batch endpoint's
example request shape exactly, down to its own "cycle":7 for a weekly
schedule), and "device schedule get-reboot"/"get-shutdown" (which read
back the same schedule via "get_reboot_plan"/"get_shutdown_plan") confirm
the schedule is genuinely stored: populated immediately after this
command, and back to empty after --cancel - including a --cancel that
used different --at/--weekly/--monthly values than the schedule it
cleared, confirming --cancel doesn't need to match the original values
exactly. The one thing NOT verified end-to-end is whether the device
actually fires the reboot/shutdown at the scheduled time - confirming
that would require letting an unattended reboot/shutdown actually occur
on a real device, which wasn't done here.

--monthly is only an approximation: the underlying mechanism repeats
every fixed number of days (30 here) from the first run, not on a true
calendar "day N of every month" - since months vary from 28 to 31 days,
the actual day it fires on will slowly drift away from the day you
picked the longer the schedule runs (unlike --weekly, which repeats every
exactly 7 days and stays exactly aligned to the weekday you chose).`

const scheduleFlagsLongChinese = `--at 用绝对日期时间（本地时区）安排一次性执行，支持 RFC3339（如
"2026-08-01T08:00:00+08:00"）或 "YYYY-MM-DD HH:MM"（如
"2026-08-01 08:00"）。--weekly <mon..sun> --time "HH:MM" 表示每周
在指定星期几的指定时间重复执行。--monthly <1-31> --time "HH:MM"
表示每月在指定号数的指定时间重复执行——注意下面的说明。--cancel
用于取消当前的定时任务，而不是新建一个。--at/--weekly/--monthly/
--cancel 四者必须且只能指定一个。

单设备版本未在 AppHub-v2.3.0.md 中记录——文档里只记录了批量/分组
版本。已用真实服务器验证过：写入会返回 {"status":"CHANGED"}（请求
结构与文档记录的批量接口示例完全一致，包括示例本身用的 "cycle":7
表示每周执行一次），并且 "device schedule get-reboot"/"get-shutdown"
（通过 "get_reboot_plan"/"get_shutdown_plan" 读取同一个定时任务）确认
这个定时任务确实被服务器保存了下来：执行本命令后立刻能查到数据，
执行 --cancel 后又变回空——即使取消时用的 --at/--weekly/--monthly
和被清除的那个任务不一样，也确认能成功取消，不需要精确匹配原来的值。
唯一还没有端到端验证过的是：设备是否真的会在预定时间触发重启/关机——
要验证这一点需要真的让一次无人值守的重启/关机在真实设备上发生，
本次没有这么做。

--monthly 只是一个近似实现：底层机制是从首次执行开始每隔固定天数
（这里是30天）重复一次，而不是真正的日历"每月第N号"——由于每个月
实际天数在28到31天之间浮动，随着定时任务运行时间越长，实际触发的
"号数"会逐渐偏离你最初选定的那个号数（这一点和 --weekly 不同，
--weekly 每次都精确间隔7天，会一直精确对准你选的那个星期几）。`

func addScheduleFlags(cmd *cobra.Command) {
	cmd.Flags().String("at", "", `one-time absolute date/time, RFC3339 or "YYYY-MM-DD HH:MM" (一次性的绝对日期时间，RFC3339 或 "YYYY-MM-DD HH:MM")`)
	cmd.Flags().String("weekly", "", `repeat weekly on this day: mon/tue/wed/thu/fri/sat/sun - requires --time (每周在这一天重复：mon/tue/wed/thu/fri/sat/sun——需要同时指定 --time)`)
	cmd.Flags().Int("monthly", 0, `repeat monthly on this day of month (1-31, approximate - see help) - requires --time (每月在这个号数重复，1-31，近似实现见帮助说明——需要同时指定 --time)`)
	cmd.Flags().String("time", "", `clock time "HH:MM" for --weekly/--monthly (配合 --weekly/--monthly 使用的时间 "HH:MM")`)
	cmd.Flags().Bool("cancel", false, "cancel any pending schedule instead of creating one (取消当前的定时任务，而不是新建一个)")
}

var deviceScheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Schedule (or cancel) a future or recurring reboot/shutdown",
	Long: `Schedule a device to reboot or shut down at a future time, optionally
repeating weekly or (approximately) monthly, or cancel a pending
schedule.

安排设备在未来的某个时间重启或关机，可选每周或（近似）每月重复执行，
或取消当前的定时任务。`,
}

var deviceScheduleRebootCmd = &cobra.Command{
	Use:   "reboot <endpoint-or-name>",
	Short: "Schedule (or cancel) a future or recurring reboot for a device",
	Long: `Schedule a device to reboot once at a future time, weekly, or
(approximately) monthly - or cancel a pending schedule.

安排设备在未来某个时间重启一次、每周重复、或（近似地）每月重复，
或取消当前的定时任务。

` + scheduleFlagsLongEnglish + `

` + scheduleFlagsLongChinese + `

Check what's currently scheduled with "device schedule get-reboot".
用 "device schedule get-reboot" 查看当前的定时任务。

Example 示例:
  eim-skill device schedule reboot "146打包机" --at "2026-08-01 03:00"
  eim-skill device schedule reboot "146打包机" --weekly sun --time 03:00
  eim-skill device schedule reboot "146打包机" --monthly 1 --time 03:00
  eim-skill device schedule reboot "146打包机" --cancel

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED"}}`,
	Args: requireArgs(cobra.ExactArgs(1)),
	RunE: func(cmd *cobra.Command, args []string) error {
		enabled, atMs, cycleDays, err := resolveSchedule(cmd)
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
		if err := client.ScheduleReboot(endpoint, enabled, atMs, cycleDays); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var deviceScheduleShutdownCmd = &cobra.Command{
	Use:   "shutdown <endpoint-or-name>",
	Short: "Schedule (or cancel) a future or recurring shutdown for a device",
	Long: `Schedule a device to shut down once at a future time, weekly, or
(approximately) monthly - or cancel a pending schedule.

安排设备在未来某个时间关机一次、每周重复、或（近似地）每月重复，
或取消当前的定时任务。

` + scheduleFlagsLongEnglish + `

` + scheduleFlagsLongChinese + `

Check what's currently scheduled with "device schedule get-shutdown".
用 "device schedule get-shutdown" 查看当前的定时任务。

Example 示例:
  eim-skill device schedule shutdown "146打包机" --at "2026-08-01 22:00"
  eim-skill device schedule shutdown "146打包机" --weekly fri --time 22:00
  eim-skill device schedule shutdown "146打包机" --monthly 15 --time 22:00
  eim-skill device schedule shutdown "146打包机" --cancel

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED"}}`,
	Args: requireArgs(cobra.ExactArgs(1)),
	RunE: func(cmd *cobra.Command, args []string) error {
		enabled, atMs, cycleDays, err := resolveSchedule(cmd)
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
		if err := client.ScheduleShutdown(endpoint, enabled, atMs, cycleDays); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

// scheduledPowerOutput is the CLI-facing shape for "device schedule
// get-reboot"/"get-shutdown" - a human-readable local RFC3339 time
// alongside the raw fields, omitted entirely when nothing is scheduled.
type scheduledPowerOutput struct {
	Scheduled bool     `json:"scheduled"`
	Time      *string  `json:"time,omitempty"`
	TimeMs    *int64   `json:"timeMs,omitempty"`
	CycleDays *float64 `json:"cycleDays,omitempty"`
}

func formatScheduledPower(p *apphub.ScheduledPower) scheduledPowerOutput {
	if p.TimeMs == 0 {
		return scheduledPowerOutput{Scheduled: false}
	}
	t := time.UnixMilli(p.TimeMs).Local().Format(time.RFC3339)
	timeMs := p.TimeMs
	out := scheduledPowerOutput{Scheduled: true, Time: &t, TimeMs: &timeMs}
	if p.Cycle.Value != nil {
		out.CycleDays = p.Cycle.Value
	}
	return out
}

var deviceScheduleGetRebootCmd = &cobra.Command{
	Use:   "get-reboot <endpoint-or-name>",
	Short: "Report a device's currently configured reboot schedule, if any",
	Long: `Report the device's currently configured reboot schedule (from "device
schedule reboot"), or "scheduled": false if nothing is currently armed.

查询设备当前配置的定时重启任务（来自 "device schedule reboot"），如果
当前没有任何定时任务，返回 "scheduled": false。

Not documented in AppHub-v2.3.0.md; uses the "get_reboot_plan"
devicecontrol action, confirmed against a live server: empty when nothing
is scheduled, populated after "device schedule reboot", and back to empty
after "device schedule reboot --cancel" - including a cancel that didn't
reuse the original --at/--weekly/--monthly values, confirming --cancel
doesn't require an exact match to disarm a schedule.

未在 AppHub-v2.3.0.md 中记录；使用 "get_reboot_plan" 设备控制动作，
已通过真实服务器验证：没有定时任务时为空，执行 "device schedule
reboot" 后会有数据，执行 "device schedule reboot --cancel" 后又变回
空——即使取消时用的 --at/--weekly/--monthly 和原定时任务不一样，也确认
能成功取消，不需要精确匹配原来的值。

Example 示例: eim-skill device schedule get-reboot "08:00:27:33:EC:2E"

Example output 输出示例 (real 真实返回):
  {"status": 200, "msg": "", "data": {"scheduled": true, "time": "2026-07-27T03:00:00+08:00", "timeMs": 1785006000000, "cycleDays": 7}}
  or, when nothing is scheduled 或没有定时任务时:
  {"status": 200, "msg": "", "data": {"scheduled": false}}`,
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
		plan, err := client.GetScheduledReboot(endpoint)
		if err != nil {
			return err
		}
		printJSON(formatScheduledPower(plan))
		return nil
	},
}

var deviceScheduleGetShutdownCmd = &cobra.Command{
	Use:   "get-shutdown <endpoint-or-name>",
	Short: "Report a device's currently configured shutdown schedule, if any",
	Long: `Report the device's currently configured shutdown schedule (from "device
schedule shutdown"), or "scheduled": false if nothing is currently armed.

查询设备当前配置的定时关机任务（来自 "device schedule shutdown"），
如果当前没有任何定时任务，返回 "scheduled": false。

Not documented in AppHub-v2.3.0.md; uses the "get_shutdown_plan"
devicecontrol action - the shutdown counterpart of "get-reboot", see its
help for verification details.

未在 AppHub-v2.3.0.md 中记录；使用 "get_shutdown_plan" 设备控制动作——
是 "get-reboot" 的关机版本，验证细节见其帮助说明。

Example 示例: eim-skill device schedule get-shutdown "08:00:27:33:EC:2E"

Example output 输出示例: {"status": 200, "msg": "", "data": {"scheduled": false}}`,
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
		plan, err := client.GetScheduledShutdown(endpoint)
		if err != nil {
			return err
		}
		printJSON(formatScheduledPower(plan))
		return nil
	},
}

func init() {
	deviceCmd.AddCommand(deviceScheduleCmd)
	deviceScheduleCmd.AddCommand(deviceScheduleRebootCmd)
	deviceScheduleCmd.AddCommand(deviceScheduleShutdownCmd)
	deviceScheduleCmd.AddCommand(deviceScheduleGetRebootCmd)
	deviceScheduleCmd.AddCommand(deviceScheduleGetShutdownCmd)

	addScheduleFlags(deviceScheduleRebootCmd)
	addScheduleFlags(deviceScheduleShutdownCmd)
}
