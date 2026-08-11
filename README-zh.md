# EIM-SKILL

[English](README.md)

`eim-skill` 是一个命令行技能（skill）工具，封装了 DeviceOn-EIM 的设备
管理 REST API（接口文档见 `/opt/DeviceOn-EIM-v2.3.0.md`），使设备相关
操作既可以通过命令行直接执行，也可以被 AI Agent 作为技能调用。

## 输出格式

无论执行的是哪个命令、是否成功，每个命令都只向 stdout 输出一个 JSON
对象：

```json
{"status": 200, "msg": "", "data": ...}
{"status": 500, "msg": "..."}
```

进程退出码在成功时为 `0`，失败时为 `1`——脚本/Agent 既可以检查退出码，
也可以检查 `status` 字段（`200`=成功，`500`=失败）。`msg` 成功时为空，
失败时包含错误信息。`data` 的内容因命令而异（设备列表、状态对象，或者
不需要返回内容时为 `null`/省略），失败时整个 `data` 字段都会省略。

每个命令的 `--help`（包括子命令的 `--help`）都包含详细说明、参数含义、
示例调用和示例输出——中英文双语。运行 `eim-skill <命令> --help`（或更
深一层子命令的 `--help`，例如 `eim-skill device set-volume --help`）可以
看到比本文档表格更详细的信息。

## 支持的命令

```
eim-skill device list                                             # 列出所有设备
eim-skill device list --group <group-name>                        # 列出某个设备组下的设备
eim-skill device list --online|--offline                          # 按在线状态过滤
eim-skill device list --alarm                                      # 有软件/硬件告警的设备
  [--group <group-name>]                                           # 以上任意过滤都可以叠加 --group
eim-skill device counts                                             # 在线/离线设备数量
eim-skill device health                                             # 在线设备的状态分布（正常/警告/错误）
eim-skill device get <endpoint-or-name>                            # 获取完整设备记录
eim-skill device details <endpoint-or-name>                        # 获取设备系统信息（CPU/内存/存储/网络）
eim-skill device delete <endpoint-or-name>                         # 删除设备
eim-skill device add-to-group <endpoint-or-name> <group>           # 把设备加入分组
eim-skill device monitor <endpoint-or-name> --type cpu|memory      # 获取CPU/内存监控数据
  [--last <30m|12h|3d>]                                            # 可选，限定为最近一段时间
  [--begin <时间>] [--end <时间>]                                   # 或者用绝对时间区间代替 --last
eim-skill device get-monitor-report <endpoint-or-name>             # CPU/内存历史数据采集是否开启？
eim-skill device set-monitor-report <endpoint-or-name> <cpu|memory> <on|off>  # 开启/关闭CPU/内存数据采集
eim-skill device reboot <endpoint-or-name>                         # 立即重启设备
eim-skill device shutdown <endpoint-or-name>                       # 立即关闭设备
eim-skill device schedule reboot <endpoint-or-name> --at <日期时间>          # 定时（一次性）重启
  | --weekly <mon..sun> --time <HH:MM>                              # ……或每周重复
  | --monthly <1-31> --time <HH:MM>                                 # ……或近似每月重复
  | --cancel                                                        # 取消当前定时任务
eim-skill device schedule shutdown <endpoint-or-name> ...           # 参数同上，用于关机
eim-skill device schedule get-reboot <endpoint-or-name>            # 查看当前的定时重启任务
eim-skill device schedule get-shutdown <endpoint-or-name>          # 查看当前的定时关机任务
eim-skill device screenshot <endpoint-or-name> [<endpoint-or-name>...]      # 截屏并保存
  [--interval <ms>] [--out-dir <dir>] [--host <url>]               # （支持批量——一次传入多台设备）
eim-skill device get-volume <endpoint-or-name>                     # 获取最后记录的音量
eim-skill device set-volume <endpoint-or-name> <0-100>             # 设置音量
eim-skill device get-backlight <endpoint-or-name>                  # 获取最后记录的屏幕背光/亮度
eim-skill device set-backlight <endpoint-or-name> <0-100>          # 设置屏幕背光/亮度
eim-skill device reboot-group <group-name> [--label <name>]        # 重启设备组下所有设备
eim-skill device shutdown-group <group-name> [--label <name>]      # 关闭设备组下所有设备
eim-skill device set-volume-group <group-name> <0-100> [--label <name>]     # 设置设备组下所有设备的音量
eim-skill device set-backlight-group <group-name> <0-100> [--label <name>] # 设置设备组下所有设备的背光/亮度
eim-skill group list                                               # 列出所有设备组
eim-skill group status [group-name]                                # 设备组的在线/警告/错误状态分布（全部或指定一个）
eim-skill group create <name> <type> [description] [--creator <username>]  # 创建设备组
```

`<endpoint-or-name>` 既可以填设备的 endpoint id（`device list` 显示的
类似MAC地址的id），也可以填人类可读的设备名称（`devName`）——每个以
设备为目标的命令都会去当前设备列表里解析这个值，所以设备名称始终可以
作为 endpoint 的替代填法。

`device list`/`device get` 返回的每个设备对象都包含 `deviceStatusText`
（`"normal"`正常/`"warning"`警告/`"error"`错误），对应原始数字状态码
`deviceStatus`（0/1/2，也就是 `device list --alarm` 过滤所用的字段）——
已转换成可读文本，无需自己记忆数字含义。

`get-volume`/`set-volume` 和 `get-backlight`/`set-backlight` 未在
DeviceOn-EIM-v2.3.0.md 中记录；使用和重启/关机相同的设备控制接口，
动作键为 `get_audio`/`set_audio` 和 `get_brightness`/`set_brightness`，
均已用真实服务器验证过（此前版本 `set-*` 用的键是错的
`audio`/`brightness`，虽然返回成功但实际不生效，已根据真实反馈修复）。
`get-volume`/`get-backlight` 反映的是对应 `set-*` 命令最后一次记录的
服务端值——并未确认在所有设备型号上都真实对应物理硬件的实际值。
`set-volume`/`set-backlight`/`set-volume-group`/`set-backlight-group`
在客户端拒绝 0-100 之外的取值——服务器对超出范围的值也会返回成功而不
报错，已根据真实反馈确认此问题。

`add-to-group` 会先对照 `group list` 校验 `<group>` 是否存在，不存在
则报错——否则底层服务器接口会静默接受并记录一个不存在的组名，已根据
真实反馈确认此问题。同时也会校验设备的操作系统是否和目标组的 type
匹配（`linux`/`windows`/`android`/`openharmony` 是服务器内置的几种
默认类型；`mix` 类型的组可以接受任何操作系统的设备）——底层接口同样
不做这个校验，已根据真实反馈确认：它会静默接受把一台Windows设备加入
一个Linux类型的组。

`reboot-group`/`shutdown-group`/`set-volume-group`/`set-backlight-group`
通过一次调用即可对整个设备组下的所有设备生效：先在客户端把 group-name
（和可选的 --label）解析成具体的设备 endpoint 列表，再通过 DeviceOn-EIM
文档记录的批量执行接口（`PUT /api/plugins/batchExec/40008/0/27601`，
请求体为 `{"clientIds":[...],"value":...}`）一次性对这些设备执行动作。
`--label` 可选，用来只对组内带有该标签的设备生效。

早期实现直接调用了文档里另一个看起来更简单、URL 里直接带组名的接口
（`PUT /api/plugins/batch/{groupname}/40008/0/27601`），但经真实用户
反馈、并在另一台服务器上独立复现确认，这个接口对任何请求体都会稳定
返回一个空内容的 HTTP 500——因此这几个命令现在改为先在客户端把组
解析成具体设备。

`device details` 同样未文档化（源自 `ClientServlet` 的 `get_1_details`
接口）；当你只需要agent上报的系统硬件信息（CPU、内存、存储、网卡）而
不是完整设备记录（在线状态、分组等）时，这个命令比 `device get` 更轻量。
这个信息块的结构因设备操作系统而异，大部分原样透传，只有 `memorysize`
和每个存储条目的 `total` 例外——已在 Linux 和 Windows 设备上用真实
服务器验证过，agent 上报的是数字字符串（如 `"memorysize":"7.64"`），
本CLI会把它们转换成真正的JSON数字。

`device counts`/`device health`/`group status` 虽然在 DeviceOn-EIM-v2.3.0.md
中有记录，但服务器返回的是扁平的 `[{"name":...,"value":...}]` 数组格式
（`group status` 用的是 `"gname"` 而不是 `"name"`）；本CLI已将其归一化
为字段清晰的单个对象。

`device schedule reboot`/`shutdown` 用于设置未来某个时间或周期性的
重启/关机（`--at` 一次性、`--weekly`/`--monthly` 周期重复、`--cancel`
取消），`device schedule get-reboot`/`get-shutdown` 用于查看当前配置的
定时任务。已用真实服务器验证过：写入调用走的是和 `device
reboot`/`shutdown` 一样的单设备接口，只是多带了 `time`（作为锚点的
Unix毫秒时间戳）和可选的 `cycle`（重复间隔天数），会返回
`{"status":"CHANGED"}`；`get-reboot`/`get-shutdown`（通过
`get_reboot_plan`/`get_shutdown_plan` 设备控制动作）确认这个定时任务
确实被服务器保存了下来——执行写入命令后立刻能查到数据，执行
`--cancel` 后又变回空，即使取消时用的 `--at`/`--weekly`/`--monthly`
和被清除的那个任务不一样，也能成功取消。这个功能针对单设备的部分
未在 DeviceOn-EIM-v2.3.0.md 中记录——文档里只记录了批量/分组版本
（`PUT /api/plugins/batchschedule/{groupname}/40008/0/27601`），其自带
示例用了 `cycle:7` 表示每周重复，印证了 `cycle` 是以"天"为单位的
间隔。`--weekly` 正好能精确对应这个机制（`cycle=7`，锚定在选定的
星期几上）；`--monthly` 只是一个近似实现（`cycle=30`）——由于每个月
实际天数不固定，定时任务运行得越久，实际触发的"号数"会逐渐偏离你
最初选定的那个号数。唯一还没有端到端验证过的是：设备是否真的会在
预定时间触发重启/关机——要验证这一点需要真的让一次无人值守的重启/
关机在真实设备上发生，本次没有这么做。

`device screenshot` 一次调用可以传入一台或多台设备，一起截屏，而不是
逐台调用——这是一个未文档化的两步流程（已通过真实抓包在真实服务器上
验证过）：先发一个批量请求（`PUT /api/devicemgr/batchExec`，接口和
请求体结构都和 `reboot-group` 等命令用的 `batchExec` 不一样），通知
所有列出的设备开始每隔 `--interval` 毫秒（默认5000）向本服务器推送
一次截屏；等待同样长的时间之后，再逐台获取每台设备最近一次的截屏
（`GET /api/devicemgr/screenshotimg/{agentId}`），并根据返回的
Content-Type 保存成 `.jpg`/`.png`，存到 `--out-dir`（默认为当前
目录）。已在一台真实的Windows设备上端到端验证过（拿到了一张真实的
1920x1080 JPEG截屏）。如果某台设备没有活动的图形界面会话——已在一台
真实的无头Linux设备上确认——那台设备会报错（"Get screen image
error"），但不会影响同一批次里的其他设备。

### 监控与告警

```
eim-skill device get-monitor <endpoint-or-name>                      # CPU/内存/存储/电池/网络实时快照

eim-skill device threshold get <endpoint-or-name>                   # 当前阈值配置
eim-skill device threshold set <endpoint-or-name> <metric> <on|off> # 配置某一项指标
  [--warn <n>] [--error <n>]
  # metric（指标）: cpu_temp（CPU温度）, cpu_usage（CPU使用率）,
  #   memory_usage（内存使用率）, storage_usage（存储使用率）,
  #   battery_available（电池剩余电量）,
  #   network_upload_speed（网络上传速度）,
  #   network_download_speed（网络下载速度）

eim-skill device process list <endpoint-or-name>                    # 设备上所有进程
eim-skill device process monitored <endpoint-or-name>               # 当前被选中监控的进程
eim-skill device process set-monitor <endpoint-or-name> <pid> <on|off>       # 把某个进程加入/移出监控

eim-skill device docker list <endpoint-or-name>                     # 设备上所有docker容器
eim-skill device docker monitored <endpoint-or-name>                 # 当前被选中监控的docker容器
eim-skill device docker set-monitor <endpoint-or-name> <container-name> <on|off>  # 把某个容器加入/移出监控

eim-skill device usb get-monitor <endpoint-or-name>                 # USB变更告警是否开启？
eim-skill device usb set-monitor <endpoint-or-name> <on|off>        # USB变更告警开关
eim-skill device usb get-storage <endpoint-or-name>                 # USB存储设备访问是否开启？
eim-skill device usb set-storage <endpoint-or-name> <on|off>        # USB存储设备访问开关
```

这对应的是 DeviceOn-EIM 网页控制台的"monitorAlert"（监控告警）功能：
CPU、内存、存储、网络是通过可配置的警告/错误阈值（`device threshold`）
来监控的，而不是实时仪表盘——可以搭配 `device monitor --type cpu|memory`
查看历史CPU/内存采样，或用 `device get-monitor` 查看实时快照（见下方）。

`device threshold set` 只有在你传了 `--warn`/`--error` 时才会覆盖对应的
值。如果省略了其中任意一个（例如只是切换某个指标的开关），会先通过
`device threshold get` 查询该指标当前的警告/错误阈值，把原值原样带上
——底层 `set_threshold` 设备控制动作每次请求都必须同时带上warn和error，
早期版本对省略的参数直接发送 `0`，导致每次切换开关都会静默把已有的
阈值清零（已根据真实反馈和抓包确认此问题）。

`--warn`/`--error` 会在客户端做校验，因为服务器本身不做任何检查——
已实测：`warn=150`、`error=200`，甚至负数都会被原样静默保存到
`cpu_usage`。百分比类指标（`cpu_usage`、`memory_usage`、
`storage_usage`、`battery_available`）必须在 0-100 之间，且都不能为
负数；两者的大小关系也必须符合该指标的方向：除 `battery_available`
外都要求 `warn < error`；`battery_available` 因为数值越低越糟糕，所以
反过来要求 `warn > error`（如 `warn=20, error=10`）。如果
`--warn`/`--error` 都不填，只是单纯切换开关，则完全不会触发这些检查。

没有 `device set-monitor`（整体硬件监控总开关）命令——它使用
`set_isMonitor` 设备控制动作，已被移除；`device threshold set` 已经
支持按指标单独关闭告警。也没有 `device process set-monitor`/
`device docker set-monitor` 命令：早期版本曾提供这两个开关（使用
`set_isProcessMonitor`/`set_isDockerMonitor` 设备控制动作），但已用
真实服务器验证过——返回成功但实际不生效，因此已移除，而不是保留一个
实际用不了的功能。

`device process list`/`device docker list` 返回设备上所有进程/所有
Docker容器（不论是否被监控）；`device process monitored`/`device
docker monitored` 则只返回当前被选中监控的那部分。两组命令背后用的是
同一个设备控制动作（`get_software_info`、`get_docker_info`）——
`monitored` 只是在客户端对 `list` 的结果做了过滤（`is_monitored`/
`ismonitor == true`），并不是单独发起了一次请求。`device docker list`
用的接口已通过真实抓包确认；早期版本读取的是另一个未文档化的 LwM2M
资源（`.../60001/0/27300`），假设返回值按 compose 项目分组——但在一台
真实部署了容器的服务器上测试时返回了空结果，因此已切换成能正常工作的
`get_docker_info` 动作。

`device process set-monitor`/`device docker set-monitor` 用于把某个
进程（按pid）或容器（按名称）加入/移出监控列表——你只需要提供
标识符，底层接口实际要求的是重新提交*完整的*进程/容器对象（来自
`list`）并带上加入/移出的标志，所以本CLI会自动先查询当前列表、找到
匹配的那一项，再原样重新提交。已通过真实抓包在真实服务器上验证过，
并做了端到端交叉验证（加入某个pid/容器后它会出现在 `monitored` 里，
移出后又会消失，且不影响其他已经在被监控的项）。DeviceOn-EIM-v2.3.0.md
里没有记录进程这块的单设备版本——只记录了批量/分组接口（`PUT
/api/plugins/batchsetsoftwaremonitor/{groupname}/40008/0/27601`），
实测这个接口稳定返回空身体的HTTP 500（和本CLI里另一个文档记录但已
损坏的批量接口是同样的错误模式），因此改用了单设备路径（未文档化，
但请求结构相同）。顺带发现并修复了一个解码bug：`device process
monitored` 的 `is_monitored` 此前一直返回 `false`，即使是真正被监控的
进程也一样——原因是 `get_monitor_app` 实际上从未在返回结果里把
`threadismonitor` 字段填成过有意义的值；对这个接口而言，"出现在这个
列表里"本身就代表"被监控"，因此现在 `is_monitored` 对这个接口返回的
每一项都固定为 `true`。

`device usb get-monitor`/`set-monitor` 控制的是USB外设变更告警是否
开启（设备插拔时触发提醒）。使用的是 `get_usb_change`/
`set_peripheral_monitor` 设备控制动作，已通过真实抓包在真实服务器上
验证过，并交替切换 `set-monitor` 的 true/false 后查询 `get-monitor`
做了交叉验证，能正确反映每次变化。（本CLI早期有一个短暂版本改用了
`get_usb_storage_status`/`set_usb_storage_status`，起因是一份实际
对应另一个独立功能——USB存储访问权限，而非这个功能——的抓包；那个
动作的SET调用会成功，并且会被记录为一个友好的审计事件名，当时看起来
是很有力的证据，但在真实设备上测试时，它的GET一侧从未反映过变化，
因此已恢复。GET能真正反映SET的效果，比审计日志里的事件名更能说明
这是一个真实生效的开关。）

`device usb get-storage`/`set-storage` 是另一组独立的开关（和上面的
get-monitor/set-monitor 不同）——控制设备是否允许使用USB存储设备
（如U盘），而不是插拔告警。`set-storage` 已用 `set_usb_storage_status`
设备控制动作真实验证过，并在 `log operations list` 的审计记录里交叉
核实：on会被记录为 `enableUsb`，off为 `disableUsb`；`get-storage` 使用
`get_usb_storage_status`，已在一台真实的Windows设备上交替切换
`set-storage` 的 true/false 后做了交叉验证，能实时正确反映每次变化。
并非所有设备都支持这个动作——测试过的一台Linux工控机无论最后设置成
什么值，都返回空结果，本CLI将其归一化为 `enabled: false`（仅凭这个
动作本身无法区分"确实关闭"和"设备不支持"两种情况）。

`device monitor` 返回的 `value` 是所选 `--type` 对应的使用率百分比
（0-100）——`--type cpu` 时是CPU使用率，`--type memory` 时是内存使用率
（根据观察到的真实数值推断，与 `device get-monitor` 的
`cpu.usage_percent`/`memory.usage_percent` 字段吻合，并非来自官方规范）。
采用宽松解析（JSON数字或数字字符串都会被归一化），因为有的服务器
返回的是字符串。

`device get-monitor-report`/`set-monitor-report` 控制的是这些历史
数据是否真的在被采集——如果某一项是关闭的，`device monitor --type
cpu|memory` 对应的类型之后就不会再有新数据。未在 DeviceOn-EIM-v2.3.0.md
中记录；`set-monitor-report` 使用的是 `set_monitor_data_report` 设备
控制动作（已通过真实抓包确认）——和 `log system set-monitor` 控制
系统日志用的是同一个动作，只是键名换成了 `cpu_monitor`/
`memory_monitor`，而不是 `systemd_log_monitor`。`get-monitor-report`
（`get_monitor_data_report`）是在调查这个问题时顺带发现的，已通过
真实服务器验证过（分别对cpu和memory做了写入再回读的验证，写入前后
都查询过）——它碰巧在同一个返回结果里也带了 `systemd_log_monitor`，
现在 `log system get-monitor` 单独把这个字段暴露了出来。

`device get-monitor` 未在 DeviceOn-EIM-v2.3.0.md 中记录（通过抓包分析
`get_board_info` 设备控制接口流量得到）。其返回值已做归一化处理：原始
agent数据里数字、数字字符串、`"N/A"` 占位符混杂不一致，本CLI统一转换
成 `float64` 字段（无法获取时为 `null`），字段命名也做了可读化处理（如
`usage_percent`、`total_gb`）。单位/含义是根据观察到的数值推断的，并非
来自官方规范——详见 `internal/DeviceOn-EIM/types.go` 里 `BoardInfo` 的
注释。

`device process monitored` 同样未文档化（`get_monitor_app` 设备控制
接口，用相同方式发现），归一化方式相似——`threadcpuloading`/
`threadmemloading` 转换成数字 `cpu_percent`/`mem_percent`，
`threadismonitor` 转换成真正的布尔值 `is_monitored`，`pid` 也转换成
真正的数字（agent 返回的是数字字符串，如 `"pid":"639"`——已用真实
服务器验证过）。其单条记录的字段结构是参照同一子系统的
`get_software_info` 接口类比推断的，并未在有数据的真实场景下验证
过——详见 `internal/DeviceOn-EIM/types.go` 里 `MonitoredProcess` 的
注释。


### 日志与操作记录

```
eim-skill log system set-monitor <endpoint-or-name> <on|off>       # 开启/关闭设备的系统日志采集/上报

eim-skill log system list [--device-id <endpoint-or-name>] [--os <os>] [--keywords <substr>]
                           [--mode <mode>] [--level <level>] [--limit N] [--last <30m|12h|3d>]
                           [--begin <时间>] [--end <时间>]            # 或者用绝对时间区间代替 --last
eim-skill log system delete-by-device <endpoint-or-name>

eim-skill log agent download <endpoint-or-name> [--output <路径>]  # 下载agent（客户端）日志，是一个zip压缩包
                             [--content [--lines N]]           # 或者直接返回最新一天日志的末尾内容

eim-skill log agent file <endpoint-or-name> <设备端绝对路径> [--lines N] [--output <路径>]  # 按绝对路径获取设备上的某一个文件

eim-skill log operations list [--limit N] [--target <substr>] [--last <30m|12h|3d>]
                               [--begin <时间>] [--end <时间>]        # 或者用绝对时间区间代替 --last
eim-skill log operations delete-all
```

`--last`（用在 `log system list`、`log operations list` 和 `device
monitor` 上）接受相对时间窗口，而不是原始的 Unix 毫秒时间戳——例如
`--last 30m`（30分钟）、`--last 12h`（12小时）、`--last 3d`（3天）——
表示从这段时间之前到现在；不填则不限制时间范围。`--begin`/`--end`
是另一种指定同样时间范围的方式，用绝对时间——RFC3339 或
`"YYYY-MM-DD HH:MM"`（如 `"2026-07-01 00:00"`）；可以只填一个也可以
两个都填（不填的一端表示不限制），并且不能和 `--last` 同时使用。
`--limit`（默认20）只是限制返回最近多少条记录，不需要关心分页
（`--page`），因为这个命令总是从结果集的最前面开始取。

`log agent download` 是一个未在 AppHub-v2.3.0.md 中记录的三步流程，
已通过真实抓包在真实服务器上验证过：先通过一次 LwM2M 写操作（资源
40001/0/27012，带上一个新生成的 actionId 和本服务器自己的地址）通知
设备推送日志；然后轮询 `/api/devicemgr/logStatus?actionId=...`（每2秒
一次，最多3分钟超时）直到返回 "success"；最后获取并保存
`/api/devicemgr/agentLog/{endpoint}`。下载结果是一个 zip 压缩包，里面
是按天分割的日志文件加上当天的日志（默认输出文件名为
`<endpoint>-agent-log.zip`）——这与 `log system` 不同（那是设备自身的
systemd/操作系统日志），这里下载的是 DeviceOn-EIM 客户端进程自己的
日志。

`--content` 会解压这个压缩包，直接以文本形式返回最新一天日志条目的
末尾内容（`data.content`），而不需要处理文件——实测最小的单日日志文件
也有几MB，直接全部返回并不现实。"最新"优先选择名字正好是
`EIM-Client.log` 的条目（当前正在写入的日志，实测和当天的按日期命名
条目内容相同）；`--lines`（默认200）控制返回末尾多少行。如果只给了
`--content` 而没给 `--output`，压缩包本身不会保存到磁盘——两者都想要
就都传上。

`log agent file <endpoint-or-name> <设备端绝对路径>` 按绝对路径获取
设备文件系统上的某一个具体文件（如
`/media/recovery/EIM-Client/20260721.log`）——同样未在 AppHub-v2.3.0.md
中记录，已通过真实抓包在真实服务器上验证过。它和 `log agent download`
的流程结构相同（触发/轮询/获取），但用的是不同的 LwM2M 资源
（40001/0/27023，直接指定文件路径）和不同的下载接口
（`/api/devicemgr/log?filename=<endpoint>_<label>.zip&type=app&clientId=<endpoint>&value=<label>`，
而不是 `/api/devicemgr/agentLog`），并且需要一个把触发请求和下载请求
关联起来的标签（触发请求里叫 `appname`，下载请求的查询参数里叫
`value`）——这里直接复用了 `actionId` 本身作为这个标签，因为抓包里没有
迹象表明它的取值还有别的意义。和整包下载 agent 日志不同，这里返回的
压缩包里始终只有一个文件（获取一个不存在的路径会在触发阶段就直接失败
并返回明确的服务器错误，而不是返回错误的内容），所以默认直接打印其
内容；`--output` 可以改为保存原始文件（比如内容是二进制，或者除了
打印之外还想留一份文件），`--lines` 则只返回文件末尾N行，而不是整个
文件。

`log operations`（管理员操作审计日志）未在 DeviceOn-EIM-v2.3.0.md 中
记录——是从 `OperationRecordServlet.java` 反推得到的，其字段名/结构
（`orid`、`type`、`event`、`target`、`username`、`ts`）已经过真实服务器
返回结果验证（`data` 是一个普通的 JSON 数组，而不是像 servlet 命名那样
是双重编码的字符串）。`event` 字段已做归一化处理（类型是 `FlexibleJSON`
而不是普通字符串）——服务器对不同条目的编码方式并不一致（有的是原始
JSON 对象，有的是包含转义JSON的字符串，有的是纯数字或普通字符串），
这是通过一次真实的解码报错（"cannot unmarshal object into Go struct
field OperationRecord.data.event of type string"）发现并确认的。

`log system set-monitor` 同样未文档化（通过抓包分析
`set_monitor_data_report`/`systemd_log_monitor` 设备控制动作得到），但
已在真实设备上验证执行成功（执行后能在 `log operations list` 的审计
记录里看到）。需要先对目标设备打开这个开关，`log system list` 才会有
该设备的数据。`log system get-monitor` 用来查询当前的开关状态
（`get_monitor_data_report`——和 `device get-monitor-report` 查询
CPU/内存采集状态用的是同一个动作，只是从同一个返回结果里读取了不同
的字段）。

`log system list` 用于查询设备上报的系统日志条目；`delete-by-device`
一次性清空某台设备的全部条目（没有按id单条删除或批量删除——
`delete-by-device` 是唯一的删除命令）。这个接口同样未在
DeviceOn-EIM-v2.3.0.md 中记录——通过真实抓包反推得到，确认接口路径是
`/api/systemLog/byPage`（大写 `P`）。本CLI早期版本用的是小写的
`/api/systemLog/bypage`，会被服务器返回 HTTP 400 "Invalid path"
错误——这是客户端自身的bug，并非服务器本身的限制，此前曾因这个错误的
结论把整组命令都移除了，现已找到问题并修正、恢复。`level` 和 `pid`
已归一化成真正的JSON数字——服务器返回的是数字字符串（如
`"level":"4"`），已用真实服务器验证过。

### 子设备

```
eim-skill subdevice list [--parent <endpoint-or-name>]              # 列出子设备，可选只看某个父设备下的
eim-skill subdevice get <device-id-or-name> [--service <名称>] [--property <名称>]  # 获取服务/属性列表 + 当前实时值（"数字孪生"）
eim-skill subdevice history <device-id-or-name> <service-name> <property-name>
  [--last <30m|12h|3d>] [--begin <时间>] [--end <时间>] [--count N]
eim-skill subdevice events <device-id-or-name> [--page N] [--limit N]
  [--last <30m|12h|3d>] [--begin <时间>] [--end <时间>]
```

子设备是注册在某个父设备（DeviceOn-EIM 管理的普通设备，即"edge"）下的
物模型设备，通过 Modbus/SNMP/executor/restful 等协议接入——每个子设备
上报若干属性（按"服务"分组），也可以上报事件。这是一套独立的只读查询
能力，由 `apphub-ithings` 服务提供（有自己的地址配置项
`--ithings-base-url`/`ithings_base_url`，默认
`http://127.0.0.1:8082`）——其中大部分接口未在 DeviceOn-EIM-v2.3.0.md
中记录（只有子设备列表、数字孪生、历史数据这三个有文档；`events` 完全
未记录），已通过真实抓包在真实服务器上验证过。它接受和主服务器
`/login` 相同的 accesstoken，不需要单独登录。

`subdevice get` 同时承担"发现有哪些服务/属性"的作用：这里没有单独
封装"只看结构不看数据"的接口，所以"数字孪生"（每个上报过的属性及其
当前值）也是用来查找 `service-name`/`property-name` 组合（供
`subdevice history` 使用）的方式。`<device-id-or-name>` 可以是子设备的
id，也可以是名称——但和父设备的 endpoint/devName 不同，子设备的名称
在不同父设备之间并不保证唯一，如果名称有歧义会报错并列出所有候选项，
而不是随便选一个。

`subdevice get` 的 `--service`/`--property` 用来在客户端缩小"数字孪生"
的范围（底层接口本身不支持按这两个条件过滤，总是返回全部已上报的
属性）——两者都指定时，如果找到唯一匹配，直接返回这一个属性对象
（不再包在列表里）；如果指定的 服务+属性 组合当前没有值，则报错。
只指定其中一个时，返回所有匹配项组成的列表。

`subdevice history` 的 `--last`/`--begin`/`--end`/`--count` 会按照文档
里的请求格式发送，但已用真实服务器验证：**目前没有任何可观察到的
过滤效果**——无论传入什么范围（包括故意构造的极端范围）返回的都是
同样的最近约20条数据。`subdevice events` 的时间过滤参数未经验证（测试
环境里没有子设备有真实的事件记录可供验证）。

### OTA 部署

每种软件包类型都遵循相同的流程：`list` 命令查询软件仓库
（m2m-management-edge）里有哪些可部署的包；`apply`/`apply-group`
（Windows 是 `install`/`upgrade`）根据你指定的包名+版本号，在那份
列表里找到对应的包、拼出下载地址，再通过 DeviceOn-EIM 网页控制台
使用的同一个设备控制接口推送给单台设备或整个设备组。

```
eim-skill ota linux list                                            # 仓库里可部署的软件包
eim-skill ota linux installed <endpoint-or-name>                    # 设备上已安装的软件包
eim-skill ota linux status <endpoint-or-name>                       # 最近一次OTA任务的状态
eim-skill ota linux apply <endpoint-or-name> <product-name> <version>
eim-skill ota linux apply-group <group-name> <product-name> <version> [--label <name>]

eim-skill ota android list
eim-skill ota android installed <endpoint-or-name>
eim-skill ota android status <endpoint-or-name>
eim-skill ota android apply <endpoint-or-name> <pkg-name> <version>
eim-skill ota android apply-group <group-name> <pkg-name> <version> [--label <name>]
eim-skill ota android uninstall <endpoint-or-name> <pkg-name>

eim-skill ota windows-exe list
eim-skill ota windows-exe installed <endpoint-or-name>
eim-skill ota windows-exe status <endpoint-or-name>
eim-skill ota windows-exe install <endpoint-or-name> <project-name> <version>
eim-skill ota windows-exe install-group <group-name> <project-name> <version>
eim-skill ota windows-exe upgrade <endpoint-or-name> <project-name> <version>
eim-skill ota windows-exe upgrade-group <group-name> <project-name> <version>
eim-skill ota windows-exe uninstall <endpoint-or-name> <project-name> <version> [--tool <tool>]

eim-skill ota windows-zip list
eim-skill ota windows-zip installed <endpoint-or-name>
eim-skill ota windows-zip status <endpoint-or-name>
eim-skill ota windows-zip install <endpoint-or-name> <project-name> <version> [--parameter <args>]
eim-skill ota windows-zip install-group <group-name> <project-name> <version> [--parameter <args>] [--label <name>]
eim-skill ota windows-zip update <endpoint-or-name> <project-name> <version> [--parameter <args>]
eim-skill ota windows-zip update-group <group-name> <project-name> <version> [--parameter <args>] [--label <name>]
eim-skill ota windows-zip uninstall <endpoint-or-name> <project-name> <version>

eim-skill ota docker list
eim-skill ota docker status <endpoint-or-name>
eim-skill ota docker apply <endpoint-or-name> <tag> <version>
eim-skill ota docker apply-group <group-name> <tag> <version> [--label <name>]
eim-skill ota docker uninstall <endpoint-or-name> <tag> <version>

eim-skill ota file list --type <type>
eim-skill ota file status <endpoint-or-name> --type <type>
eim-skill ota file deploy <endpoint-or-name> <file-name> <save-path> --type <type> [--exec]
eim-skill ota file deploy-group <group-name> <file-name> <save-path> --type <type> [--exec] [--label <name>]

eim-skill ota bsp status <endpoint-or-name>                         # BSP/系统镜像更新状态（仅支持status，无list/apply）
```

说明：
- `windows-exe uninstall` 的 `--tool` 默认为 `"Inno Setup"`；如果安装包
  是用 "WiX Toolset" 打包的，需要显式传 `--tool "WiX Toolset"`。
  `install`/`upgrade` 都会自动从仓库信息里读取，两种情况都无需手动指定。
- Android/Docker/Linux 的安装和更新共用同一个设备端接口（服务器会根据
  包是否已安装来判断）；Windows 的安装和升级在设备端是真正独立的两个
  接口，所以本CLI用独立的命令表示，而不是合并成一个。
- `ota file list`/`deploy`/`deploy-group` 都需要 `--type`，且必须和文件
  当初上传到仓库时所属的分类一致——没有"不区分类型列出所有文件"的接口。
  这里的 `--type` 是随意选定的分类标签，不是操作系统/平台名称——
  `ota file list --type linux`（或 `windows`/`android`）通常只会返回
  空数组（已根据真实反馈确认：有用户这样用，以为能查到对应平台的
  安装包），除非恰好有通用文件是用这个字面值当分类名上传的。想查询
  真正的 Linux/Android/Windows/Docker 安装包，请改用 `ota linux
  list`/`ota android list`/`ota windows-exe list`/`ota windows-zip
  list`/`ota docker list`。
- `ota <type> status`（包括 `bsp`）查询设备最近一次 OTA/BSP 任务的状态。
  这些命令均未在 AppHub-v2.3.0.md 中记录——通过 apifox 导出的 AppHub
  网页控制台 OpenAPI 文档找到，已确认对真实服务器请求能成功返回，但
  该服务器当时没有任何任务历史记录，因此没有一个命令的"有数据时"返回
  结构得到过验证。`linux`/`android`/`windows-exe`/`windows-zip`/`docker`
  共用一个后端，其 "type" 分类参数只有 Android 的取值（固定值 "sg"）
  有文档记录；这五个命令都复用 "sg"，假设它是整个管理器共用的，但除
  Android 外均未验证。`bsp` 使用完全独立、不需要 type 参数的后端，
  风险相对更低。完整说明见 `skills/ota-deployment/SKILL.md`。
- 已针对真实服务器验证：全部五种软件包类型的仓库列表接口，以及完整的
  "设备名解析 → 仓库查找 → 错误处理"流程（`ota file list --type deploy`
  返回的真实仓库数据与本CLI的结构体完全一致）。每种类型最终推送到设备
  的部署请求是根据 DeviceOn-EIM-v2.3.0.md 文档记录的请求/响应格式实现
  的，但本轮尚未在真实设备上实际执行过。

### 服务器配置

```
eim-skill server get-config                                        # 客户端接入配置数据
```

`server get-config` 不针对任何具体设备——返回服务器自身的客户端接入
配置数据（`{"id":"WP.SRP","data":"<加密字符串>"}`）。把 `data` 直接
复制粘贴到设备agent自己的配置界面里，就能让它连接到这台服务器，不需要
在agent里手动逐项填写服务器地址/凭据——这对应网页控制台的二维码设备
配对功能（网页上生成的二维码内容就是这段数据）。未在
DeviceOn-EIM-v2.3.0.md 中记录；已用真实服务器验证过。

## 配置

`eim-skill` 需要 DeviceOn-EIM 服务器的地址和登录凭证，OTA 相关命令还
需要软件仓库服务的地址，`subdevice` 相关命令还需要 ithings 服务的
地址。配置按以下优先级从低到高读取：

1. `conf/config.yaml`（或通过 `--config` 指定的路径）：
   ```yaml
   base_url: http://127.0.0.1:30010
   username: admin
   password: admin
   repo_base_url: http://127.0.0.1:30001
   ithings_base_url: http://127.0.0.1:8082
   ```
2. 环境变量：`EIM_BASE_URL`、`EIM_USERNAME`、`EIM_PASSWORD`、`EIM_REPO_BASE_URL`、`EIM_ITHINGS_BASE_URL`
3. 命令行参数：`--base-url`、`--username`、`--password`、`--repo-base-url`、`--ithings-base-url`

软件仓库服务和 ithings 服务都接受和主服务器 `/login` 签发的同一个
`accesstoken`，因此都不需要单独配置登录凭证。

### TLS 证书信任（自签名 / 内部 CA 证书的服务器）

如果 `base_url`（或软件仓库/ithings 服务地址）使用 `https://`，且证书由
内部私有 CA 签发，请求会报错 `x509: certificate signed by unknown
authority`。可以用以下两种方式之一解决：

```yaml
# 推荐：额外信任一个 CA（PEM 文件），其余校验照常进行。
ca_cert_file: /path/to/internal-ca.pem

# 或者：完全跳过证书校验 —— 仅在明确信任该网络路径时使用，
# 因为这会失去对该连接的中间人攻击防护。
insecure_skip_verify: true
```

也可以通过环境变量（`EIM_CA_CERT_FILE`、`EIM_INSECURE_SKIP_VERIFY`）或
命令行参数（`--ca-cert-file`、`--insecure`）设置。同时设置时，
`--insecure`/`insecure_skip_verify` 优先级高于
`--ca-cert-file`/`ca_cert_file`。

## MCP 服务器模式

```
eim-skill mcp
```

以 [Model Context Protocol](https://modelcontextprotocol.io) 服务器模式
运行 `eim-skill`（通过 stdio 通信），这样支持 MCP 的 Agent/客户端可以
直接把命令当作工具调用，而不需要另外调起CLI进程。每个可执行的命令都会
自动生成一个对应的 MCP 工具，工具名为命令路径用 `_` 连接（例如
`device list` → `device_list`，`ota windows-exe apply-group` →
`ota_windows-exe_apply-group`）。工具的输入 schema 是根据命令的位置参数
（从 `Use` 字符串解析，例如 `<endpoint-or-name>` 是必填，`[id...]` 是
可选的可重复数组）和 flag（根据底层 pflag 类型映射为
`string`/`boolean`/`integer`/`number`，通过 `MarkFlagRequired` 标记必填）
自动生成的。每次工具调用返回的内容和对应CLI调用完全相同的JSON信封，
当 `status` 为 `500` 时会把 `IsError` 设为 true。

要把它注册到某个 MCP 客户端，只需把编译好的二进制路径和 `mcp` 参数
配置进去，例如：

```json
{
  "mcpServers": {
    "eim-skill": {
      "command": "/path/to/eim-skill",
      "args": ["mcp"]
    }
  }
}
```

配置项（服务器地址、登录凭证、仓库服务地址）的读取方式和直接命令行
调用一样（见上方"配置"一节）——可以通过 `conf/config.yaml`、环境变量，
或者在 `args` 里额外传入CLI参数（例如
`["mcp", "--config", "/path/to/config.yaml"]`）。

## Skills（技能文档）

[`skills/`](skills/) 目录下每个能力领域对应一个 `SKILL.md`（设备管理、
设备监控、日志/记录、OTA部署、子设备），分别说明什么场景下该用它、以及
具体的命令——是写给 Agent（或人）在决定该用本CLI的哪部分功能时参考的。

## 如何构建

```
$ make        ## 编译 eim-skill 二进制
$ make clean  ## 清理项目
```

## 参与贡献

如果你有兴趣参与贡献，请查看
[CONTRIBUTING](CONTRIBUTING.md) 了解提交补丁和贡献流程的详细说明。
