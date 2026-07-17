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
eim-skill device reboot <endpoint-or-name>                         # 立即重启设备
eim-skill device shutdown <endpoint-or-name>                       # 立即关闭设备
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
真实反馈确认此问题。

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

eim-skill device process monitored <endpoint-or-name>               # 当前被选中监控的进程

eim-skill device docker monitored <endpoint-or-name>                 # 当前被选中监控的docker容器

eim-skill device usb get-monitor <endpoint-or-name>                 # USB存储设备访问是否开启？
eim-skill device usb set-monitor <endpoint-or-name> <on|off>        # USB存储设备访问开关
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

没有 `device set-monitor`（整体硬件监控总开关）命令——它使用
`set_isMonitor` 设备控制动作，已被移除；`device threshold set` 已经
支持按指标单独关闭告警。也没有 `device process set-monitor`/
`device docker set-monitor` 命令：早期版本曾提供这两个开关（使用
`set_isProcessMonitor`/`set_isDockerMonitor` 设备控制动作），但已用
真实服务器验证过——返回成功但实际不生效，因此已移除，而不是保留一个
实际用不了的功能。

也没有 `device process list`（不论是否被监控、列出全部进程）命令——
只保留 `device process monitored`，已经覆盖了实际使用场景。
`device docker monitored` 现在是 Docker 版本的 `device process monitored`：
只返回当前被选中监控的容器（`ismonitor == true`），而不是所有已部署的
容器。早期版本读取的是另一个未文档化的 LwM2M 资源（`.../60001/0/
27300`），假设返回值按 compose 项目分组——但在一台真实部署了容器的
服务器上测试时返回了空结果，因此已切换成能正常工作的 `get_docker_info`
设备控制动作（已通过真实抓包确认），并在客户端过滤出被监控的子集，
与 `device process monitored` 的形态/用途保持一致。

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
这是一个真实生效的开关。）USB存储设备访问权限本身
（`get_usb_storage_status`/`set_usb_storage_status`）目前没有做成
命令，因为它的GET一侧没能验证成功。

`device monitor` 返回的 `value` 是所选 `--type` 对应的使用率百分比
（0-100）——`--type cpu` 时是CPU使用率，`--type memory` 时是内存使用率
（根据观察到的真实数值推断，与 `device get-monitor` 的
`cpu.usage_percent`/`memory.usage_percent` 字段吻合，并非来自官方规范）。
采用宽松解析（JSON数字或数字字符串都会被归一化），因为有的服务器
返回的是字符串。

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
eim-skill log system delete-by-device <endpoint-or-name>

eim-skill log operations list [--limit N] [--target <substr>] [--last <30m|12h|3d>]
eim-skill log operations delete-all
```

`--last`（用在 `log system list`、`log operations list` 和 `device
monitor` 上）接受相对时间窗口，而不是原始的 Unix 毫秒时间戳——例如
`--last 30m`（30分钟）、`--last 12h`（12小时）、`--last 3d`（3天）——
表示从这段时间之前到现在；不填则不限制时间范围。`--limit`（默认20）
只是限制返回最近多少条记录，不需要关心分页（`--page`），因为这个命令
总是从结果集的最前面开始取。

`log operations`（管理员操作审计日志）未在 DeviceOn-EIM-v2.3.0.md 中
记录——是从 `OperationRecordServlet.java` 反推得到的，其字段名/结构
（`orid`、`type`、`event`、`target`、`username`、`ts`）已经过真实服务器
返回结果验证（`data` 是一个普通的 JSON 数组，而不是像 servlet 命名那样
是双重编码的字符串）。

`log system set-monitor` 同样未文档化（通过抓包分析
`set_monitor_data_report`/`systemd_log_monitor` 设备控制动作得到），但
已在真实设备上验证执行成功（执行后能在 `log operations list` 的审计
记录里看到）。需要先对目标设备打开这个开关，`log system list` 才会有
该设备的数据。

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

## 配置

`eim-skill` 需要 DeviceOn-EIM 服务器的地址和登录凭证，OTA 相关命令还
需要软件仓库服务的地址。配置按以下优先级从低到高读取：

1. `conf/config.yaml`（或通过 `--config` 指定的路径）：
   ```yaml
   base_url: http://127.0.0.1:30010
   username: admin
   password: admin
   repo_base_url: http://127.0.0.1:30001
   ```
2. 环境变量：`EIM_BASE_URL`、`EIM_USERNAME`、`EIM_PASSWORD`、`EIM_REPO_BASE_URL`
3. 命令行参数：`--base-url`、`--username`、`--password`、`--repo-base-url`

软件仓库服务接受和主服务器 `/login` 签发的同一个 `accesstoken`，因此
不需要单独配置仓库服务的登录凭证。

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
设备监控、日志/记录、OTA部署），分别说明什么场景下该用它、以及具体的
命令——是写给 Agent（或人）在决定该用本CLI的哪部分功能时参考的。

## 如何构建

```
$ make        ## 编译 eim-skill 二进制
$ make clean  ## 清理项目
```

## 参与贡献

如果你有兴趣参与贡献，请查看
[CONTRIBUTING](CONTRIBUTING.md) 了解提交补丁和贡献流程的详细说明。
