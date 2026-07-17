package cmd

import (
	"fmt"
	"time"

	"github.com/gangqiang01/EIM-SKILL/internal/apphub"
	"github.com/gangqiang01/EIM-SKILL/internal/repo"
	"github.com/spf13/cobra"
)

var otaCmd = &cobra.Command{
	Use:   "ota",
	Short: "OTA-deploy Android/Windows/Docker/Linux packages and generic files to devices",
	Long: `Each package type follows the same workflow: "list" queries the repo server
(m2m-management-edge) for packages available to deploy, then "apply"/
"apply-group" (or "install"/"upgrade" for Windows) look up the requested
package by name+version in that listing, build its download path, and
push it to a device or a whole group via the same devicecontrol resource
the DeviceLink web console uses.

每种软件包类型都遵循相同的流程："list" 命令查询软件仓库（
m2m-management-edge）里有哪些可部署的包，然后 "apply"/"apply-group"
（Windows 是 "install"/"upgrade"）根据你指定的包名+版本号，在那份
列表里找到对应的包、拼出下载地址，再通过 DeviceLink 控制台使用的同一个
设备控制接口推送给单台设备或整个设备组。

Package types 软件包类型:
  linux         Linux deb/tar 软件包
  android       Android APK 应用
  windows-exe   Windows exe 安装包（Inno Setup / WiX Toolset）
  windows-zip   Windows zip/压缩包
  docker        Docker compose/swarm 编排包
  file          通用文件（不区分类型的任意文件）
  bsp           BSP/系统镜像更新（仅支持 status 查询，见下）

--name/--version must exactly match an entry from that type's "list"
command, name/version 必须和对应 "list" 命令返回结果里的名称/版本号
完全一致才能找到并部署。

Every type except "bsp" also has a "status" command to check a device's
most recent OTA task status (queued/running/success/failed) - see each
type's "status" subcommand help for important caveats about how reliable
that specific type's status query is.

除 "bsp" 外，每种类型都有一个 "status" 命令用来查询设备最近一次OTA
任务的状态（排队/执行中/成功/失败）——具体可靠程度请看各类型 "status"
子命令自己的详细说明。`,
}

func init() {
	rootCmd.AddCommand(otaCmd)
	otaCmd.AddCommand(otaLinuxCmd)
	otaCmd.AddCommand(otaAndroidCmd)
	otaCmd.AddCommand(otaWindowsExeCmd)
	otaCmd.AddCommand(otaWindowsZipCmd)
	otaCmd.AddCommand(otaDockerCmd)
	otaCmd.AddCommand(otaFileCmd)
	otaCmd.AddCommand(otaBspCmd)
}

// appDeployStatusLongNote is appended to every appstatusmgr-backed type's
// "status" command help (every type except "bsp", which uses its own
// dedicated, type-param-free endpoint - see otaBspStatusCmd).
const appDeployStatusLongNote = `

UNVERIFIED 未验证: not documented in AppHub-v2.3.0.md - found via an
apifox-exported OpenAPI spec and confirmed to respond successfully
against a live server, but that server had zero OTA task history
recorded at the time, so the non-empty response shape and the "type"
business-classifier's value for this specific package type could not be
confirmed against real data. Only Android's value ("sg") is documented
(as a "fixed value") anywhere; this command reuses "sg" for every
package type on the assumption it's shared across the whole appstatusmgr
manager rather than varying per type - verify against your own server
before relying on this in an unattended workflow.

未在 AppHub-v2.3.0.md 中记录——通过 apifox 导出的 OpenAPI 文档找到，
已确认对真实服务器请求能成功返回，但该服务器当时没有任何OTA任务历史
记录，因此无法验证有数据时的返回结构、也无法验证这个具体包类型对应的
"type" 分类参数取值是否正确。文档里唯一明确记录的取值是 Android 的
"sg"（且被描述为"固定值"）；本命令对所有包类型都复用 "sg"，假设它是
整个 appstatusmgr 管理器共用的取值而非按类型区分——正式无人值守流程
使用前请先在你自己的服务器上验证。`

func otaTaskname() string {
	return fmt.Sprintf("ota-%d", time.Now().UnixMilli())
}

// ==================== Linux ====================

var otaLinuxCmd = &cobra.Command{
	Use:   "linux",
	Short: "OTA-deploy Linux packages (deb/tar)",
	Long: `OTA-deploy Linux packages (deb/tar) - install/update has a single shared
device-side resource (the server distinguishes install vs. update by
whether the package is already present), so "apply"/"apply-group" cover
both cases.

部署 Linux 软件包（deb/tar 格式）——安装和更新在设备端共用同一个接口
（服务器会根据包是否已安装来判断是全新安装还是更新），所以
"apply"/"apply-group" 同时覆盖这两种场景。`,
}

func init() {
	otaLinuxCmd.AddCommand(otaLinuxListCmd)
	otaLinuxCmd.AddCommand(otaLinuxInstalledCmd)
	otaLinuxCmd.AddCommand(otaLinuxStatusCmd)
	otaLinuxCmd.AddCommand(otaLinuxApplyCmd)
	otaLinuxCmd.AddCommand(otaLinuxApplyGroupCmd)
	otaLinuxApplyGroupCmd.Flags().String("label", "", "only target devices in the group that also carry this label (只对组内带有该标签的设备生效)")
}

var otaLinuxListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Linux packages available in the repo server",
	Long: `List Linux packages available in the repo server (m2m-management-edge) -
these are the name/version pairs you can pass to "apply"/"apply-group".

列出软件仓库（m2m-management-edge）里可部署的 Linux 软件包——这里的
名称/版本号就是你之后调用 "apply"/"apply-group" 时要填的值。

Example 示例: eim-skill ota linux list

Example output 输出示例 (empty when the repo has no Linux packages
uploaded yet 仓库尚未上传任何 Linux 包时返回空数组):
  {"status": 200, "msg": "", "data": []}`,
	Args: requireArgs(cobra.NoArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		repoClient, err := newRepoClient(client)
		if err != nil {
			return err
		}
		pkgs, err := repoClient.ListLinuxPackages()
		if err != nil {
			return err
		}
		printJSON(pkgs)
		return nil
	},
}

var otaLinuxInstalledCmd = &cobra.Command{
	Use:   "installed <endpoint-or-name>",
	Short: "List Linux packages installed on a device",
	Long: `List Linux packages currently installed on a device, with their status.

列出设备上当前已安装的 Linux 软件包及其状态。

Example 示例: eim-skill ota linux installed "146打包机"

Example output 输出示例 (real 真实返回):
  {
    "status": 200, "msg": "",
    "data": [
      {
        "appname": "AppHub-Edge", "pkgname": "AppHub-Edge_3.0.5.tar.gz",
        "pkgtype": "compressed", "version": "3.0.5", "status": "created",
        "updatetime": "2024-08-01 11:37:23"
      }
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
		pkgs, err := client.GetInstalledLinuxPackages(endpoint)
		if err != nil {
			return err
		}
		printJSON(pkgs)
		return nil
	},
}

var otaLinuxStatusCmd = &cobra.Command{
	Use:   "status <endpoint-or-name>",
	Short: "Get a device's most recent Linux OTA task status",
	Long: `Get the most recent Linux package OTA task status for a device (e.g.
queued/running/success/failed, per whatever "apply"/"apply-group" call
was last made against it).

获取设备最近一次 Linux 软件包 OTA 任务的状态（如排队/执行中/成功/
失败，对应最近一次对它调用 "apply"/"apply-group" 的结果）。

Example 示例: eim-skill ota linux status "146打包机"` + appDeployStatusLongNote,
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
		raw, err := client.GetAppDeployStatus(endpoint)
		if err != nil {
			return err
		}
		printRawJSON(raw)
		return nil
	},
}

func findLinuxPackage(pkgs []repo.LinuxPackage, productName, version string) (*repo.LinuxPackage, error) {
	var available []string
	for i := range pkgs {
		if pkgs[i].ProductName != productName {
			continue
		}
		if pkgs[i].Version == version {
			return &pkgs[i], nil
		}
		available = append(available, pkgs[i].Version)
	}
	if len(available) == 0 {
		return nil, fmt.Errorf("no Linux package named %q found in repo", productName)
	}
	return nil, fmt.Errorf("Linux package %q has no version %q in repo, available versions: %v", productName, version, available)
}

var otaLinuxApplyCmd = &cobra.Command{
	Use:   "apply <endpoint-or-name> <product-name> <version>",
	Short: "Install or update a Linux package on a single device",
	Long: `Install or update a Linux package on a single device. product-name and
version must exactly match an entry from "ota linux list".

在单台设备上安装/更新 Linux 软件包。product-name 和 version 必须和
"ota linux list" 返回结果里的名称/版本号完全一致。

Example 示例: eim-skill ota linux apply "146打包机" AppHub-Edge 3.0.5

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED", "taskname": "ota-1784...", "downloadpath": "http://.../AppHub-Edge_3.0.5.tar.gz"}}`,
	Args: requireArgs(cobra.ExactArgs(3)),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		repoClient, err := newRepoClient(client)
		if err != nil {
			return err
		}
		pkgs, err := repoClient.ListLinuxPackages()
		if err != nil {
			return err
		}
		pkg, err := findLinuxPackage(pkgs, args[1], args[2])
		if err != nil {
			return err
		}
		downloadPath := pkg.Address + "/" + pkg.Filename
		taskname := otaTaskname()
		if err := client.DeployLinuxPackage(endpoint, taskname, args[1], args[2], downloadPath); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED", "taskname": taskname, "downloadpath": downloadPath})
		return nil
	},
}

var otaLinuxApplyGroupCmd = &cobra.Command{
	Use:   "apply-group <group-name> <product-name> <version>",
	Short: "Install or update a Linux package on every device in a group",
	Long: `Install or update a Linux package on every device in group-name in a
single call. --label optionally narrows the target to devices in the
group that also carry that label.

一次性在整个设备组下的所有设备上安装/更新 Linux 软件包。--label 可选，
用来只对组内带有该标签的设备生效。

Example 示例: eim-skill ota linux apply-group default AppHub-Edge 3.0.5

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED", "taskname": "ota-1784...", "downloadpath": "http://.../AppHub-Edge_3.0.5.tar.gz"}}`,
	Args: requireArgs(cobra.ExactArgs(3)),
	RunE: func(cmd *cobra.Command, args []string) error {
		label, _ := cmd.Flags().GetString("label")
		client, err := newClient()
		if err != nil {
			return err
		}
		repoClient, err := newRepoClient(client)
		if err != nil {
			return err
		}
		pkgs, err := repoClient.ListLinuxPackages()
		if err != nil {
			return err
		}
		pkg, err := findLinuxPackage(pkgs, args[1], args[2])
		if err != nil {
			return err
		}
		downloadPath := pkg.Address + "/" + pkg.Filename
		taskname := otaTaskname()
		if err := client.DeployLinuxPackageToGroup(args[0], taskname, args[1], args[2], downloadPath, label); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED", "taskname": taskname, "downloadpath": downloadPath})
		return nil
	},
}

// ==================== Android ====================

var otaAndroidCmd = &cobra.Command{
	Use:   "android",
	Short: "OTA-deploy Android APKs",
	Long: `OTA-deploy Android APKs. Like Linux, install/update share one device-side
resource ("apply"/"apply-group" cover both); uninstall is separate.

部署 Android APK 应用。和 Linux 类似，安装和更新共用同一个接口
（"apply"/"apply-group" 同时覆盖两种场景）；卸载是单独的命令。`,
}

func init() {
	otaAndroidCmd.AddCommand(otaAndroidListCmd)
	otaAndroidCmd.AddCommand(otaAndroidInstalledCmd)
	otaAndroidCmd.AddCommand(otaAndroidStatusCmd)
	otaAndroidCmd.AddCommand(otaAndroidApplyCmd)
	otaAndroidCmd.AddCommand(otaAndroidApplyGroupCmd)
	otaAndroidCmd.AddCommand(otaAndroidUninstallCmd)
	otaAndroidApplyGroupCmd.Flags().String("label", "", "only target devices in the group that also carry this label (只对组内带有该标签的设备生效)")
}

var otaAndroidListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Android APKs available in the repo server",
	Long: `List Android APKs available in the repo server - the name/version pairs
you can pass to "apply"/"apply-group"/"uninstall".

列出软件仓库里可部署的 Android APK——这里的名称/版本号就是之后调用
"apply"/"apply-group"/"uninstall" 时要填的值。

Example 示例: eim-skill ota android list

Example output 输出示例 (empty when none uploaded 仓库尚未上传任何
APK 时返回空数组): {"status": 200, "msg": "", "data": []}`,
	Args: requireArgs(cobra.NoArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		repoClient, err := newRepoClient(client)
		if err != nil {
			return err
		}
		apps, err := repoClient.ListAndroidApps()
		if err != nil {
			return err
		}
		printJSON(apps)
		return nil
	},
}

var otaAndroidInstalledCmd = &cobra.Command{
	Use:   "installed <endpoint-or-name>",
	Short: "List apps installed on an Android device",
	Long: `List apps currently installed on an Android device.

列出 Android 设备上当前已安装的应用。

Example 示例: eim-skill ota android installed "<android-device>"`,
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
		raw, err := client.GetInstalledAndroidApps(endpoint)
		if err != nil {
			return err
		}
		printRawJSON(raw)
		return nil
	},
}

var otaAndroidStatusCmd = &cobra.Command{
	Use:   "status <endpoint-or-name>",
	Short: "Get a device's most recent Android OTA task status",
	Long: `Get the most recent Android app OTA task status for a device (e.g.
queued/running/success/failed, per whatever "apply"/"apply-group" call
was last made against it).

获取设备最近一次 Android 应用 OTA 任务的状态（如排队/执行中/成功/
失败，对应最近一次对它调用 "apply"/"apply-group" 的结果）。

This is the one package type where the underlying "type" query value
("sg") is actually documented in AppHub-v2.3.0.md (as a fixed literal for
this exact query) - so this specific status command is on more solid
ground than its Linux/Windows/Docker siblings.

这是唯一一个底层 "type" 查询取值（"sg"）在 AppHub-v2.3.0.md 中确有
文档记录（该文档明确标注为此查询的固定值）的包类型——所以这条
status 命令比 Linux/Windows/Docker 的对应命令更有把握。

Example 示例: eim-skill ota android status "<android-device>"`,
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
		raw, err := client.GetAppDeployStatus(endpoint)
		if err != nil {
			return err
		}
		printRawJSON(raw)
		return nil
	},
}

func findAndroidApp(apps []repo.AndroidApp, pkgName, version string) (*repo.AndroidApp, error) {
	var available []string
	for i := range apps {
		if apps[i].PkgName != pkgName {
			continue
		}
		if apps[i].VersionName == version {
			return &apps[i], nil
		}
		available = append(available, apps[i].VersionName)
	}
	if len(available) == 0 {
		return nil, fmt.Errorf("no Android app with package name %q found in repo", pkgName)
	}
	return nil, fmt.Errorf("Android app %q has no version %q in repo, available versions: %v", pkgName, version, available)
}

var otaAndroidApplyCmd = &cobra.Command{
	Use:   "apply <endpoint-or-name> <pkg-name> <version>",
	Short: "Install or update an Android app on a single device",
	Long: `Install or update an Android app on a single device. pkg-name is the
Android package name (e.g. "com.example.app"); version must exactly match
an entry from "ota android list".

在单台 Android 设备上安装/更新应用。pkg-name 是 Android 包名（如
"com.example.app"）；version 必须和 "ota android list" 返回结果里的
版本号完全一致。

Example 示例: eim-skill ota android apply "<android-device>" com.example.app 1.2.0`,
	Args: requireArgs(cobra.ExactArgs(3)),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		repoClient, err := newRepoClient(client)
		if err != nil {
			return err
		}
		apps, err := repoClient.ListAndroidApps()
		if err != nil {
			return err
		}
		app, err := findAndroidApp(apps, args[1], args[2])
		if err != nil {
			return err
		}
		downloadPath := app.Address + "/" + app.Filename
		taskname := otaTaskname()
		if err := client.DeployAndroidApp(endpoint, taskname, app.Filename, args[1], app.VersionCode, args[2], downloadPath); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED", "taskname": taskname, "downloadpath": downloadPath})
		return nil
	},
}

var otaAndroidApplyGroupCmd = &cobra.Command{
	Use:   "apply-group <group-name> <pkg-name> <version>",
	Short: "Install or update an Android app on every device in a group",
	Long: `Install or update an Android app on every device in group-name in a
single call. --label optionally narrows the target to devices in the
group that also carry that label.

一次性在整个设备组下的所有 Android 设备上安装/更新应用。--label 可选，
用来只对组内带有该标签的设备生效。

Example 示例: eim-skill ota android apply-group default com.example.app 1.2.0`,
	Args: requireArgs(cobra.ExactArgs(3)),
	RunE: func(cmd *cobra.Command, args []string) error {
		label, _ := cmd.Flags().GetString("label")
		client, err := newClient()
		if err != nil {
			return err
		}
		repoClient, err := newRepoClient(client)
		if err != nil {
			return err
		}
		apps, err := repoClient.ListAndroidApps()
		if err != nil {
			return err
		}
		app, err := findAndroidApp(apps, args[1], args[2])
		if err != nil {
			return err
		}
		downloadPath := app.Address + "/" + app.Filename
		taskname := otaTaskname()
		if err := client.DeployAndroidAppToGroup(args[0], taskname, app.Filename, args[1], app.VersionCode, args[2], downloadPath, label); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED", "taskname": taskname, "downloadpath": downloadPath})
		return nil
	},
}

var otaAndroidUninstallCmd = &cobra.Command{
	Use:   "uninstall <endpoint-or-name> <pkg-name>",
	Short: "Uninstall an Android app from a single device",
	Long: `Uninstall an Android app from a single device by its package name.

按包名从单台 Android 设备上卸载应用。

Example 示例: eim-skill ota android uninstall "<android-device>" com.example.app

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED"}}`,
	Args: requireArgs(cobra.ExactArgs(2)),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		if err := client.UninstallAndroidApp(endpoint, args[1]); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

// ==================== Windows exe ====================

var otaWindowsExeCmd = &cobra.Command{
	Use:   "windows-exe",
	Short: "OTA-deploy Windows exe installers (Inno Setup / WiX Toolset)",
	Long: `OTA-deploy Windows exe installers. Unlike Linux/Android/Docker, Windows
has genuinely separate install vs. upgrade device-side resources, so they
are exposed as distinct "install"/"upgrade" commands instead of one
collapsed "apply".

部署 Windows exe 安装包。和 Linux/Android/Docker 不同，Windows 的
安装与升级在设备端是两个真正独立的接口，所以这里分成 "install" 和
"upgrade" 两个命令，而不是合并成一个 "apply"。`,
}

func init() {
	otaWindowsExeCmd.AddCommand(otaWindowsExeListCmd)
	otaWindowsExeCmd.AddCommand(otaWindowsExeInstalledCmd)
	otaWindowsExeCmd.AddCommand(otaWindowsExeStatusCmd)
	otaWindowsExeCmd.AddCommand(otaWindowsExeInstallCmd)
	otaWindowsExeCmd.AddCommand(otaWindowsExeInstallGroupCmd)
	otaWindowsExeCmd.AddCommand(otaWindowsExeUpgradeCmd)
	otaWindowsExeCmd.AddCommand(otaWindowsExeUpgradeGroupCmd)
	otaWindowsExeCmd.AddCommand(otaWindowsExeUninstallCmd)
	otaWindowsExeUninstallCmd.Flags().String("tool", "Inno Setup", `packaging tool used to build the installer: "Inno Setup" or "WiX Toolset" (default: "Inno Setup") - 打包该安装包所用的工具（默认："Inno Setup"）`)
}

var otaWindowsExeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Windows exe installers available in the repo server",
	Long: `List Windows exe installers available in the repo server - the
name/version pairs you can pass to "install"/"upgrade".

列出软件仓库里可部署的 Windows exe 安装包——这里的名称/版本号就是
之后调用 "install"/"upgrade" 时要填的值。

Example 示例: eim-skill ota windows-exe list

Example output 输出示例 (empty when none uploaded 仓库尚未上传任何
exe 包时返回空数组): {"status": 200, "msg": "", "data": []}`,
	Args: requireArgs(cobra.NoArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		repoClient, err := newRepoClient(client)
		if err != nil {
			return err
		}
		pkgs, err := repoClient.ListWindowsPackages("exe")
		if err != nil {
			return err
		}
		printJSON(pkgs)
		return nil
	},
}

var otaWindowsExeInstalledCmd = &cobra.Command{
	Use:   "installed <endpoint-or-name>",
	Short: "List exe packages installed on a device",
	Long: `List exe packages currently installed on a Windows device.

列出 Windows 设备上当前已安装的 exe 软件包。

Example 示例: eim-skill ota windows-exe installed "<windows-device>"`,
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
		raw, err := client.GetInstalledWindowsExePackages(endpoint)
		if err != nil {
			return err
		}
		printRawJSON(raw)
		return nil
	},
}

var otaWindowsExeStatusCmd = &cobra.Command{
	Use:   "status <endpoint-or-name>",
	Short: "Get a device's most recent Windows exe OTA task status",
	Long: `Get the most recent Windows exe package OTA task status for a device
(e.g. queued/running/success/failed, per whatever "install"/"upgrade"
call was last made against it).

获取设备最近一次 Windows exe 软件包 OTA 任务的状态（如排队/执行中/
成功/失败，对应最近一次对它调用 "install"/"upgrade" 的结果）。

Example 示例: eim-skill ota windows-exe status "<windows-device>"` + appDeployStatusLongNote,
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
		raw, err := client.GetAppDeployStatus(endpoint)
		if err != nil {
			return err
		}
		printRawJSON(raw)
		return nil
	},
}

func findWindowsPackage(pkgs []repo.WindowsPackage, projectName, version string) (*repo.WindowsPackage, error) {
	var available []string
	for i := range pkgs {
		if pkgs[i].ProjectName != projectName {
			continue
		}
		if pkgs[i].Version == version {
			return &pkgs[i], nil
		}
		available = append(available, pkgs[i].Version)
	}
	if len(available) == 0 {
		return nil, fmt.Errorf("no Windows package named %q found in repo", projectName)
	}
	return nil, fmt.Errorf("Windows package %q has no version %q in repo, available versions: %v", projectName, version, available)
}

var otaWindowsExeInstallCmd = &cobra.Command{
	Use:   "install <endpoint-or-name> <project-name> <version>",
	Short: "Install an exe package on a single device",
	Long: `Install an exe package on a single device. project-name and version must
exactly match an entry from "ota windows-exe list"; the packaging tool
(Inno Setup/WiX Toolset) is read from that repo entry automatically.

在单台 Windows 设备上安装 exe 软件包。project-name 和 version 必须和
"ota windows-exe list" 返回结果里的名称/版本号完全一致；打包工具
（Inno Setup/WiX Toolset）会自动从仓库信息里读取，无需手动指定。

Example 示例: eim-skill ota windows-exe install "<windows-device>" MyApp 1.0.0

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED", "taskname": "ota-1784...", "downloadpath": "http://.../MyApp-1.0.0.exe", "tool": "Inno Setup"}}`,
	Args: requireArgs(cobra.ExactArgs(3)),
	RunE: func(cmd *cobra.Command, args []string) error {
		return applyWindowsExeSingle(args[0], args[1], args[2], false)
	},
}

var otaWindowsExeUpgradeCmd = &cobra.Command{
	Use:   "upgrade <endpoint-or-name> <project-name> <version>",
	Short: "Upgrade an already-installed exe package on a single device",
	Long: `Upgrade an already-installed exe package on a single device to
project-name/version (must exist in "ota windows-exe list").

把单台 Windows 设备上已安装的 exe 软件包升级到指定的 project-name/
version（必须存在于 "ota windows-exe list" 返回结果里）。

Example 示例: eim-skill ota windows-exe upgrade "<windows-device>" MyApp 1.1.0`,
	Args: requireArgs(cobra.ExactArgs(3)),
	RunE: func(cmd *cobra.Command, args []string) error {
		return applyWindowsExeSingle(args[0], args[1], args[2], true)
	},
}

func applyWindowsExeSingle(endpointOrName, projectName, version string, upgrade bool) error {
	client, err := newClient()
	if err != nil {
		return err
	}
	endpoint, err := resolveDeviceEndpoint(client, endpointOrName)
	if err != nil {
		return err
	}
	pkg, taskname, downloadPath, err := lookupWindowsExe(client, projectName, version)
	if err != nil {
		return err
	}
	if upgrade {
		err = client.UpgradeWindowsExe(endpoint, taskname, projectName, pkg.Tool, version, downloadPath)
	} else {
		err = client.InstallWindowsExe(endpoint, taskname, projectName, pkg.Tool, version, downloadPath)
	}
	if err != nil {
		return err
	}
	printJSON(map[string]string{"status": "CHANGED", "taskname": taskname, "downloadpath": downloadPath, "tool": pkg.Tool})
	return nil
}

var otaWindowsExeInstallGroupCmd = &cobra.Command{
	Use:   "install-group <group-name> <project-name> <version>",
	Short: "Install an exe package on every device in a group",
	Long: `Install an exe package on every device in group-name in a single call.

一次性在整个设备组下的所有 Windows 设备上安装 exe 软件包。

Example 示例: eim-skill ota windows-exe install-group win-group MyApp 1.0.0`,
	Args: requireArgs(cobra.ExactArgs(3)),
	RunE: func(cmd *cobra.Command, args []string) error {
		return applyWindowsExeGroup(args[0], args[1], args[2], false)
	},
}

var otaWindowsExeUpgradeGroupCmd = &cobra.Command{
	Use:   "upgrade-group <group-name> <project-name> <version>",
	Short: "Upgrade an exe package on every device in a group",
	Long: `Upgrade an already-installed exe package on every device in group-name in
a single call.

一次性把整个设备组下所有 Windows 设备已安装的 exe 软件包升级到指定
版本。

Example 示例: eim-skill ota windows-exe upgrade-group win-group MyApp 1.1.0`,
	Args: requireArgs(cobra.ExactArgs(3)),
	RunE: func(cmd *cobra.Command, args []string) error {
		return applyWindowsExeGroup(args[0], args[1], args[2], true)
	},
}

func applyWindowsExeGroup(groupName, projectName, version string, upgrade bool) error {
	client, err := newClient()
	if err != nil {
		return err
	}
	pkg, taskname, downloadPath, err := lookupWindowsExe(client, projectName, version)
	if err != nil {
		return err
	}
	if upgrade {
		err = client.UpgradeWindowsExeToGroup(groupName, taskname, projectName, pkg.Tool, version, downloadPath)
	} else {
		err = client.InstallWindowsExeToGroup(groupName, taskname, projectName, pkg.Tool, version, downloadPath)
	}
	if err != nil {
		return err
	}
	printJSON(map[string]string{"status": "CHANGED", "taskname": taskname, "downloadpath": downloadPath, "tool": pkg.Tool})
	return nil
}

// lookupWindowsExe finds the repo package for projectName/version and
// returns it along with a fresh taskname and its download path.
func lookupWindowsExe(client *apphub.Client, projectName, version string) (*repo.WindowsPackage, string, string, error) {
	repoClient, err := newRepoClient(client)
	if err != nil {
		return nil, "", "", err
	}
	pkgs, err := repoClient.ListWindowsPackages("exe")
	if err != nil {
		return nil, "", "", err
	}
	pkg, err := findWindowsPackage(pkgs, projectName, version)
	if err != nil {
		return nil, "", "", err
	}
	return pkg, otaTaskname(), pkg.Address + "/" + pkg.Filename, nil
}

var otaWindowsExeUninstallCmd = &cobra.Command{
	Use:   "uninstall <endpoint-or-name> <project-name> <version> [--tool <tool>]",
	Short: `Uninstall an exe package from a single device (--tool defaults to "Inno Setup")`,
	Long: `Uninstall an exe package from a single device. Unlike install/upgrade
(which read the packaging tool from the repo listing automatically),
uninstall needs to know which tool built the installer - pass --tool
explicitly if the package was built with "WiX Toolset"; it defaults to
"Inno Setup" when omitted.

从单台设备上卸载 exe 软件包。和 install/upgrade（会自动从仓库信息里
读取打包工具）不同，卸载需要知道安装包是用哪个工具打包的——如果是
"WiX Toolset" 打包的，需要显式传入 --tool；不传时默认为 "Inno Setup"。

Example 示例: eim-skill ota windows-exe uninstall "<windows-device>" MyApp 1.0.0
       eim-skill ota windows-exe uninstall "<windows-device>" MyApp 1.0.0 --tool "WiX Toolset"

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED", "taskname": "ota-1784..."}}`,
	Args: requireArgs(cobra.ExactArgs(3)),
	RunE: func(cmd *cobra.Command, args []string) error {
		tool, _ := cmd.Flags().GetString("tool")
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		taskname := otaTaskname()
		if err := client.UninstallWindowsExe(endpoint, taskname, args[1], tool, args[2]); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED", "taskname": taskname})
		return nil
	},
}

// ==================== Windows zip ====================

var otaWindowsZipCmd = &cobra.Command{
	Use:   "windows-zip",
	Short: "OTA-deploy Windows zip/compressed packages",
	Long: `OTA-deploy Windows zip/compressed packages (extracted and optionally run
with --parameter arguments, unlike the installer-driven windows-exe type).

部署 Windows zip/压缩包类型的软件包（解压后可选通过 --parameter 传参
运行，和走安装程序流程的 windows-exe 类型不同）。`,
}

func init() {
	otaWindowsZipCmd.AddCommand(otaWindowsZipListCmd)
	otaWindowsZipCmd.AddCommand(otaWindowsZipInstalledCmd)
	otaWindowsZipCmd.AddCommand(otaWindowsZipStatusCmd)
	otaWindowsZipCmd.AddCommand(otaWindowsZipInstallCmd)
	otaWindowsZipCmd.AddCommand(otaWindowsZipInstallGroupCmd)
	otaWindowsZipCmd.AddCommand(otaWindowsZipUpdateCmd)
	otaWindowsZipCmd.AddCommand(otaWindowsZipUpdateGroupCmd)
	otaWindowsZipCmd.AddCommand(otaWindowsZipUninstallCmd)

	for _, c := range []*cobra.Command{otaWindowsZipInstallCmd, otaWindowsZipInstallGroupCmd, otaWindowsZipUpdateCmd, otaWindowsZipUpdateGroupCmd} {
		c.Flags().String("parameter", "", "optional arguments passed to the extracted package (传给解压后程序的可选启动参数)")
	}
	otaWindowsZipInstallGroupCmd.Flags().String("label", "", "only target devices in the group that also carry this label (只对组内带有该标签的设备生效)")
	otaWindowsZipUpdateGroupCmd.Flags().String("label", "", "only target devices in the group that also carry this label (只对组内带有该标签的设备生效)")
}

var otaWindowsZipListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Windows zip packages available in the repo server",
	Long: `List Windows zip packages available in the repo server - the
name/version pairs you can pass to "install"/"update".

列出软件仓库里可部署的 Windows zip 软件包——这里的名称/版本号就是
之后调用 "install"/"update" 时要填的值。

Example 示例: eim-skill ota windows-zip list

Example output 输出示例 (empty when none uploaded 仓库尚未上传任何
zip 包时返回空数组): {"status": 200, "msg": "", "data": []}`,
	Args: requireArgs(cobra.NoArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		repoClient, err := newRepoClient(client)
		if err != nil {
			return err
		}
		pkgs, err := repoClient.ListWindowsPackages("zip")
		if err != nil {
			return err
		}
		printJSON(pkgs)
		return nil
	},
}

var otaWindowsZipInstalledCmd = &cobra.Command{
	Use:   "installed <endpoint-or-name>",
	Short: "List zip packages installed on a device",
	Long: `List zip packages currently installed on a Windows device.

列出 Windows 设备上当前已安装的 zip 软件包。

Example 示例: eim-skill ota windows-zip installed "<windows-device>"`,
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
		raw, err := client.GetInstalledWindowsZipPackages(endpoint)
		if err != nil {
			return err
		}
		printRawJSON(raw)
		return nil
	},
}

var otaWindowsZipStatusCmd = &cobra.Command{
	Use:   "status <endpoint-or-name>",
	Short: "Get a device's most recent Windows zip OTA task status",
	Long: `Get the most recent Windows zip package OTA task status for a device
(e.g. queued/running/success/failed, per whatever "install"/"update"
call was last made against it).

获取设备最近一次 Windows zip 软件包 OTA 任务的状态（如排队/执行中/
成功/失败，对应最近一次对它调用 "install"/"update" 的结果）。

Example 示例: eim-skill ota windows-zip status "<windows-device>"` + appDeployStatusLongNote,
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
		raw, err := client.GetAppDeployStatus(endpoint)
		if err != nil {
			return err
		}
		printRawJSON(raw)
		return nil
	},
}

var otaWindowsZipInstallCmd = &cobra.Command{
	Use:   "install <endpoint-or-name> <project-name> <version>",
	Short: "Install a zip package on a single device",
	Long: `Install a zip package on a single device. project-name and version must
exactly match an entry from "ota windows-zip list". --parameter is
optional (arguments passed to the extracted package once run).

在单台 Windows 设备上安装 zip 软件包。project-name 和 version 必须和
"ota windows-zip list" 返回结果里的名称/版本号完全一致。--parameter
可选，是解压后运行程序时传入的参数。

Example 示例: eim-skill ota windows-zip install "<windows-device>" MyTool 2.0.0 --parameter "--silent"`,
	Args: requireArgs(cobra.ExactArgs(3)),
	RunE: func(cmd *cobra.Command, args []string) error {
		parameter, _ := cmd.Flags().GetString("parameter")
		return applyWindowsZipSingle(args[0], args[1], args[2], parameter, false)
	},
}

var otaWindowsZipUpdateCmd = &cobra.Command{
	Use:   "update <endpoint-or-name> <project-name> <version>",
	Short: "Update an already-installed zip package on a single device",
	Long: `Update an already-installed zip package on a single device to
project-name/version (must exist in "ota windows-zip list").

把单台 Windows 设备上已安装的 zip 软件包更新到指定的 project-name/
version（必须存在于 "ota windows-zip list" 返回结果里）。

Example 示例: eim-skill ota windows-zip update "<windows-device>" MyTool 2.1.0`,
	Args: requireArgs(cobra.ExactArgs(3)),
	RunE: func(cmd *cobra.Command, args []string) error {
		parameter, _ := cmd.Flags().GetString("parameter")
		return applyWindowsZipSingle(args[0], args[1], args[2], parameter, true)
	},
}

func applyWindowsZipSingle(endpointOrName, projectName, version, parameter string, update bool) error {
	client, err := newClient()
	if err != nil {
		return err
	}
	endpoint, err := resolveDeviceEndpoint(client, endpointOrName)
	if err != nil {
		return err
	}
	repoClient, err := newRepoClient(client)
	if err != nil {
		return err
	}
	pkgs, err := repoClient.ListWindowsPackages("zip")
	if err != nil {
		return err
	}
	pkg, err := findWindowsPackage(pkgs, projectName, version)
	if err != nil {
		return err
	}
	downloadPath := pkg.Address + "/" + pkg.Filename
	taskname := otaTaskname()
	if update {
		err = client.UpdateWindowsZip(endpoint, taskname, projectName, version, downloadPath, parameter)
	} else {
		err = client.InstallWindowsZip(endpoint, taskname, projectName, version, downloadPath, parameter)
	}
	if err != nil {
		return err
	}
	printJSON(map[string]string{"status": "CHANGED", "taskname": taskname, "downloadpath": downloadPath})
	return nil
}

var otaWindowsZipInstallGroupCmd = &cobra.Command{
	Use:   "install-group <group-name> <project-name> <version>",
	Short: "Install a zip package on every device in a group",
	Long: `Install a zip package on every device in group-name in a single call.
--label optionally narrows the target to devices in the group that also
carry that label.

一次性在整个设备组下的所有 Windows 设备上安装 zip 软件包。--label
可选，用来只对组内带有该标签的设备生效。

Example 示例: eim-skill ota windows-zip install-group win-group MyTool 2.0.0`,
	Args: requireArgs(cobra.ExactArgs(3)),
	RunE: func(cmd *cobra.Command, args []string) error {
		parameter, _ := cmd.Flags().GetString("parameter")
		label, _ := cmd.Flags().GetString("label")
		return applyWindowsZipGroup(args[0], args[1], args[2], parameter, label, false)
	},
}

var otaWindowsZipUpdateGroupCmd = &cobra.Command{
	Use:   "update-group <group-name> <project-name> <version>",
	Short: "Update a zip package on every device in a group",
	Long: `Update an already-installed zip package on every device in group-name in
a single call. --label optionally narrows the target to devices in the
group that also carry that label.

一次性把整个设备组下所有 Windows 设备已安装的 zip 软件包更新到指定
版本。--label 可选，用来只对组内带有该标签的设备生效。

Example 示例: eim-skill ota windows-zip update-group win-group MyTool 2.1.0`,
	Args: requireArgs(cobra.ExactArgs(3)),
	RunE: func(cmd *cobra.Command, args []string) error {
		parameter, _ := cmd.Flags().GetString("parameter")
		label, _ := cmd.Flags().GetString("label")
		return applyWindowsZipGroup(args[0], args[1], args[2], parameter, label, true)
	},
}

func applyWindowsZipGroup(groupName, projectName, version, parameter, label string, update bool) error {
	client, err := newClient()
	if err != nil {
		return err
	}
	repoClient, err := newRepoClient(client)
	if err != nil {
		return err
	}
	pkgs, err := repoClient.ListWindowsPackages("zip")
	if err != nil {
		return err
	}
	pkg, err := findWindowsPackage(pkgs, projectName, version)
	if err != nil {
		return err
	}
	downloadPath := pkg.Address + "/" + pkg.Filename
	taskname := otaTaskname()
	if update {
		err = client.UpdateWindowsZipToGroup(groupName, taskname, projectName, version, downloadPath, parameter, label)
	} else {
		err = client.InstallWindowsZipToGroup(groupName, taskname, projectName, version, downloadPath, parameter, label)
	}
	if err != nil {
		return err
	}
	printJSON(map[string]string{"status": "CHANGED", "taskname": taskname, "downloadpath": downloadPath})
	return nil
}

var otaWindowsZipUninstallCmd = &cobra.Command{
	Use:   "uninstall <endpoint-or-name> <project-name> <version>",
	Short: "Uninstall a zip package from a single device",
	Long: `Uninstall a zip package from a single device.

从单台设备上卸载 zip 软件包。

Example 示例: eim-skill ota windows-zip uninstall "<windows-device>" MyTool 2.0.0

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED", "taskname": "ota-1784..."}}`,
	Args: requireArgs(cobra.ExactArgs(3)),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		taskname := otaTaskname()
		if err := client.UninstallWindowsZip(endpoint, taskname, args[1], args[2]); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED", "taskname": taskname})
		return nil
	},
}

// ==================== Docker ====================

var otaDockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "OTA-deploy Docker compose/swarm packages",
	Long: `OTA-deploy Docker compose/swarm packages. Like Linux/Android, install and
update share one device-side resource, so "apply"/"apply-group" cover
both.

部署 Docker compose/swarm 编排包。和 Linux/Android 类似，安装和更新
共用同一个接口，所以 "apply"/"apply-group" 同时覆盖两种场景。This
group has no "what's currently deployed/running" query - "device docker
monitored" (part of the device-monitoring feature set) only reports the
subset of containers selected for monitoring, not every deployed
container. 这个命令组没有"当前已部署/运行了什么"的查询命令——
"device docker monitored"（属于设备监控功能）只返回被选中监控的那部分
容器，并非所有已部署的容器。`,
}

func init() {
	otaDockerCmd.AddCommand(otaDockerListCmd)
	otaDockerCmd.AddCommand(otaDockerStatusCmd)
	otaDockerCmd.AddCommand(otaDockerApplyCmd)
	otaDockerCmd.AddCommand(otaDockerApplyGroupCmd)
	otaDockerCmd.AddCommand(otaDockerUninstallCmd)
	otaDockerApplyGroupCmd.Flags().String("label", "", "only target devices in the group that also carry this label (只对组内带有该标签的设备生效)")
}

var otaDockerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Docker compose/swarm packages available in the repo server",
	Long: `List Docker compose/swarm packages available in the repo server - the
tag/version pairs you can pass to "apply"/"apply-group"/"uninstall".

列出软件仓库里可部署的 Docker compose/swarm 编排包——这里的
tag/版本号就是之后调用 "apply"/"apply-group"/"uninstall" 时要填的值。

Example 示例: eim-skill ota docker list

Example output 输出示例 (empty when none uploaded 仓库尚未上传任何
Docker 包时返回空数组): {"status": 200, "msg": "", "data": []}`,
	Args: requireArgs(cobra.NoArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		repoClient, err := newRepoClient(client)
		if err != nil {
			return err
		}
		pkgs, err := repoClient.ListDockerComposePackages()
		if err != nil {
			return err
		}
		printJSON(pkgs)
		return nil
	},
}

var otaDockerStatusCmd = &cobra.Command{
	Use:   "status <endpoint-or-name>",
	Short: "Get a device's most recent Docker OTA task status",
	Long: `Get the most recent Docker compose/swarm package OTA task status for a
device (e.g. queued/running/success/failed, per whatever "apply"/
"apply-group" call was last made against it). This is task history, not a
live snapshot - "device docker monitored" (device-monitoring feature set)
only reports containers currently selected for monitoring, not every
deployed container, so it's not a full substitute.

获取设备最近一次 Docker compose/swarm 编排包 OTA 任务的状态（如排队/
执行中/成功/失败，对应最近一次对它调用 "apply"/"apply-group" 的
结果）。这是任务历史，不是实时快照——"device docker monitored"
（设备监控功能）只返回当前被选中监控的容器，并非所有已部署的容器，
所以不能完全替代这里的查询。

Example 示例: eim-skill ota docker status "146打包机"` + appDeployStatusLongNote,
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
		raw, err := client.GetAppDeployStatus(endpoint)
		if err != nil {
			return err
		}
		printRawJSON(raw)
		return nil
	},
}

func findDockerPackage(pkgs []repo.DockerComposePackage, tag, version string) (*repo.DockerComposePackage, error) {
	var available []string
	for i := range pkgs {
		if pkgs[i].Tag != tag {
			continue
		}
		if pkgs[i].Version == version {
			return &pkgs[i], nil
		}
		available = append(available, pkgs[i].Version)
	}
	if len(available) == 0 {
		return nil, fmt.Errorf("no Docker package tagged %q found in repo", tag)
	}
	return nil, fmt.Errorf("Docker package %q has no version %q in repo, available versions: %v", tag, version, available)
}

var otaDockerApplyCmd = &cobra.Command{
	Use:   "apply <endpoint-or-name> <tag> <version>",
	Short: "Install or update a Docker compose/swarm package on a single device",
	Long: `Install or update a Docker compose/swarm package on a single device. tag
and version must exactly match an entry from "ota docker list".

在单台设备上安装/更新 Docker compose/swarm 编排包。tag 和 version
必须和 "ota docker list" 返回结果里的值完全一致。

Example 示例: eim-skill ota docker apply "146打包机" myapp-stack 1.0.0`,
	Args: requireArgs(cobra.ExactArgs(3)),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		repoClient, err := newRepoClient(client)
		if err != nil {
			return err
		}
		pkgs, err := repoClient.ListDockerComposePackages()
		if err != nil {
			return err
		}
		pkg, err := findDockerPackage(pkgs, args[1], args[2])
		if err != nil {
			return err
		}
		downloadPath := pkg.Address + "/" + pkg.Filename
		taskname := otaTaskname()
		if err := client.DeployDockerCompose(endpoint, taskname, pkg.Filename, args[1], args[2], downloadPath); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED", "taskname": taskname, "downloadpath": downloadPath})
		return nil
	},
}

var otaDockerApplyGroupCmd = &cobra.Command{
	Use:   "apply-group <group-name> <tag> <version>",
	Short: "Install or update a Docker compose/swarm package on every device in a group",
	Long: `Install or update a Docker compose/swarm package on every device in
group-name in a single call. --label optionally narrows the target to
devices in the group that also carry that label.

一次性在整个设备组下的所有设备上安装/更新 Docker compose/swarm
编排包。--label 可选，用来只对组内带有该标签的设备生效。

Example 示例: eim-skill ota docker apply-group default myapp-stack 1.0.0`,
	Args: requireArgs(cobra.ExactArgs(3)),
	RunE: func(cmd *cobra.Command, args []string) error {
		label, _ := cmd.Flags().GetString("label")
		client, err := newClient()
		if err != nil {
			return err
		}
		repoClient, err := newRepoClient(client)
		if err != nil {
			return err
		}
		pkgs, err := repoClient.ListDockerComposePackages()
		if err != nil {
			return err
		}
		pkg, err := findDockerPackage(pkgs, args[1], args[2])
		if err != nil {
			return err
		}
		downloadPath := pkg.Address + "/" + pkg.Filename
		taskname := otaTaskname()
		if err := client.DeployDockerComposeToGroup(args[0], taskname, pkg.Filename, args[1], args[2], downloadPath, label); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED", "taskname": taskname, "downloadpath": downloadPath})
		return nil
	},
}

var otaDockerUninstallCmd = &cobra.Command{
	Use:   "uninstall <endpoint-or-name> <tag> <version>",
	Short: "Uninstall a Docker compose/swarm package from a single device",
	Long: `Uninstall a Docker compose/swarm package from a single device.

从单台设备上卸载 Docker compose/swarm 编排包。

Example 示例: eim-skill ota docker uninstall "146打包机" myapp-stack 1.0.0

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED", "taskname": "ota-1784..."}}`,
	Args: requireArgs(cobra.ExactArgs(3)),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		taskname := otaTaskname()
		if err := client.UninstallDockerCompose(endpoint, taskname, args[1], args[2]); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED", "taskname": taskname})
		return nil
	},
}

// ==================== Generic file deploy ====================

var otaFileCmd = &cobra.Command{
	Use:   "file",
	Short: "Deploy generic files to devices",
	Long: `Deploy generic files (of any kind, not just recognized package formats)
to devices - no install/uninstall concept, just "push this file to this
path". --type is required on every command in this group and must match
whatever category the file was uploaded under in the repo server; there is
no unconditional "list every file regardless of type" endpoint.

把任意文件（不局限于已识别的软件包格式）推送到设备——没有"安装/卸载"
的概念，就是"把这个文件放到这个路径"。这个命令组下每个命令都必须传
--type，且必须和文件当初上传到仓库时所属的分类一致；没有"不区分类型
列出所有文件"的接口。`,
}

func init() {
	otaFileCmd.AddCommand(otaFileListCmd)
	otaFileCmd.AddCommand(otaFileStatusCmd)
	otaFileCmd.AddCommand(otaFileDeployCmd)
	otaFileCmd.AddCommand(otaFileDeployGroupCmd)

	otaFileListCmd.Flags().String("type", "", "repo file category/type (required) - 仓库里文件的分类/类型（必填）")
	_ = otaFileListCmd.MarkFlagRequired("type")

	otaFileStatusCmd.Flags().String("type", "", "repo file category/type (required, matches how the file was uploaded) - 仓库里文件的分类/类型（必填，需和上传时一致）")
	_ = otaFileStatusCmd.MarkFlagRequired("type")

	for _, c := range []*cobra.Command{otaFileDeployCmd, otaFileDeployGroupCmd} {
		c.Flags().String("type", "", "repo file category/type (required, matches how the file was uploaded) - 仓库里文件的分类/类型（必填，需和上传时一致）")
		c.Flags().Bool("exec", false, "make the deployed file executable and run it (部署后赋予可执行权限并运行)")
		_ = c.MarkFlagRequired("type")
	}
	otaFileDeployGroupCmd.Flags().String("label", "", "only target devices in the group that also carry this label (只对组内带有该标签的设备生效)")
}

var otaFileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List files available in the repo server for a given --type",
	Long: `List files available in the repo server under the given --type category -
the file names you can pass to "deploy"/"deploy-group".

按 --type 指定的分类，列出软件仓库里可部署的文件——这里的文件名
就是之后调用 "deploy"/"deploy-group" 时要填的值。

Example 示例: eim-skill ota file list --type deploy

Example output 输出示例 (empty when nothing uploaded under that type
该分类下尚未上传任何文件时返回空数组): {"status": 200, "msg": "", "data": []}`,
	Args: requireArgs(cobra.NoArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		fileType, _ := cmd.Flags().GetString("type")
		client, err := newClient()
		if err != nil {
			return err
		}
		repoClient, err := newRepoClient(client)
		if err != nil {
			return err
		}
		files, err := repoClient.ListFiles(fileType)
		if err != nil {
			return err
		}
		printJSON(files)
		return nil
	},
}

var otaFileStatusCmd = &cobra.Command{
	Use:   "status <endpoint-or-name> --type <type>",
	Short: "Get a device's most recent generic-file OTA task status",
	Long: `Get the most recent generic-file deployment task status for a device
under repo category --type (e.g. queued/running/success/failed, per
whatever "deploy"/"deploy-group" call was last made against it).

获取设备最近一次某个 --type 分类下的通用文件部署任务状态（如排队/
执行中/成功/失败，对应最近一次对它调用 "deploy"/"deploy-group" 的
结果）。

Example 示例: eim-skill ota file status "146打包机" --type deploy

UNVERIFIED 未验证: not documented in AppHub-v2.3.0.md - found via an
apifox-exported OpenAPI spec (fileDeployApi.js) and confirmed to respond
successfully against a live server, but that server had zero file-deploy
task history recorded at the time, so the non-empty response shape could
not be confirmed against real data.

未在 AppHub-v2.3.0.md 中记录——通过 apifox 导出的 OpenAPI 文档
（fileDeployApi.js）找到，已确认对真实服务器请求能成功返回，但该
服务器当时没有任何文件部署任务历史记录，因此无法验证有数据时的
实际返回结构。`,
	Args: requireArgs(cobra.ExactArgs(1)),
	RunE: func(cmd *cobra.Command, args []string) error {
		fileType, _ := cmd.Flags().GetString("type")
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		raw, err := client.GetFileDeployStatus(endpoint, fileType)
		if err != nil {
			return err
		}
		printRawJSON(raw)
		return nil
	},
}

func findRepoFile(files []repo.GeneralFile, filename string) (*repo.GeneralFile, error) {
	for i := range files {
		if files[i].Filename == filename {
			return &files[i], nil
		}
	}
	return nil, fmt.Errorf("no file named %q found in repo for this type", filename)
}

var otaFileDeployCmd = &cobra.Command{
	Use:   "deploy <endpoint-or-name> <file-name> <save-path>",
	Short: "Deploy a file to a single device",
	Long: `Deploy a file to a single device. file-name must exactly match an entry
from "ota file list --type <type>" (same --type required here); save-path
is where on the device's filesystem to write it. --exec additionally makes
it executable and runs it after deployment.

把文件部署到单台设备。file-name 必须和 "ota file list --type <type>"
返回结果里的文件名完全一致（这里同样需要传相同的 --type）；save-path
是文件在设备上的保存路径。--exec 会额外赋予可执行权限并在部署后运行它。

Example 示例: eim-skill ota file deploy "146打包机" report.pdf /home/user/report.pdf --type deploy

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED", "taskname": "ota-1784...", "downloadpath": "http://.../report.pdf"}}`,
	Args: requireArgs(cobra.ExactArgs(3)),
	RunE: func(cmd *cobra.Command, args []string) error {
		fileType, _ := cmd.Flags().GetString("type")
		isExec, _ := cmd.Flags().GetBool("exec")

		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		repoClient, err := newRepoClient(client)
		if err != nil {
			return err
		}
		files, err := repoClient.ListFiles(fileType)
		if err != nil {
			return err
		}
		file, err := findRepoFile(files, args[1])
		if err != nil {
			return err
		}
		downloadPath := file.Address + "/" + file.Filename
		taskname := otaTaskname()
		if err := client.DeployFile(endpoint, taskname, args[2], file.Filename, downloadPath, isExec); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED", "taskname": taskname, "downloadpath": downloadPath})
		return nil
	},
}

var otaFileDeployGroupCmd = &cobra.Command{
	Use:   "deploy-group <group-name> <file-name> <save-path>",
	Short: "Deploy a file to every device in a group",
	Long: `Deploy a file to every device in group-name in a single call. Same
--type/--exec semantics as "deploy". --label optionally narrows the target
to devices in the group that also carry that label.

一次性把文件部署到整个设备组下的所有设备。--type/--exec 语义和
"deploy" 相同。--label 可选，用来只对组内带有该标签的设备生效。

Example 示例: eim-skill ota file deploy-group default report.pdf /home/user/report.pdf --type deploy`,
	Args: requireArgs(cobra.ExactArgs(3)),
	RunE: func(cmd *cobra.Command, args []string) error {
		fileType, _ := cmd.Flags().GetString("type")
		isExec, _ := cmd.Flags().GetBool("exec")
		label, _ := cmd.Flags().GetString("label")

		client, err := newClient()
		if err != nil {
			return err
		}
		repoClient, err := newRepoClient(client)
		if err != nil {
			return err
		}
		files, err := repoClient.ListFiles(fileType)
		if err != nil {
			return err
		}
		file, err := findRepoFile(files, args[1])
		if err != nil {
			return err
		}
		downloadPath := file.Address + "/" + file.Filename
		taskname := otaTaskname()
		if err := client.DeployFileToGroup(args[0], taskname, args[2], file.Filename, downloadPath, isExec, label); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED", "taskname": taskname, "downloadpath": downloadPath})
		return nil
	},
}

// ==================== BSP / system update ====================

var otaBspCmd = &cobra.Command{
	Use:   "bsp",
	Short: "Query BSP/system-image update status",
	Long: `Query BSP (Board Support Package)/system-image update deployment
status. Unlike the other OTA package types, this CLI only exposes a
"status" query for BSP - there is no "list"/"apply" here, since browsing
and pushing BSP images was out of scope for this feature; only checking
a device's most recent BSP update status was requested.

查询 BSP（板级支持包）/系统镜像更新的部署状态。和其他 OTA
包类型不同，这里只提供 "status" 查询——没有 "list"/"apply"，
因为浏览和推送 BSP 镜像本身不在这次功能范围内，只需要能查询
设备最近一次 BSP 更新的状态。`,
}

func init() {
	otaBspCmd.AddCommand(otaBspStatusCmd)
}

var otaBspStatusCmd = &cobra.Command{
	Use:   "status <endpoint-or-name>",
	Short: "Get a device's most recent BSP/system-update task status",
	Long: `Get the most recent BSP (Board Support Package)/system-image update
task status for a device (e.g. queued/running/success/failed).

获取设备最近一次 BSP（板级支持包）/系统镜像更新任务的状态（如排队/
执行中/成功/失败）。

Example 示例: eim-skill ota bsp status "146打包机"

UNVERIFIED 未验证: not documented in AppHub-v2.3.0.md - found via an
apifox-exported OpenAPI spec (updateBSPApi.js) and confirmed to respond
successfully against a live server, but that server had zero BSP-update
task history recorded at the time, so the non-empty response shape could
not be confirmed against real data. Unlike the appstatusmgr-backed status
commands (linux/android/windows-exe/windows-zip/docker), this endpoint
takes no "type" business-classifier parameter at all, so there's no
per-type value to guess - the only real uncertainty here is the shape of
a populated response.

未在 AppHub-v2.3.0.md 中记录——通过 apifox 导出的 OpenAPI 文档
（updateBSPApi.js）找到，已确认对真实服务器请求能成功返回，但该
服务器当时没有任何 BSP 更新任务历史记录，因此无法验证有数据时的
实际返回结构。和依赖 appstatusmgr 的其他 status 命令（linux/android/
windows-exe/windows-zip/docker）不同，这个接口完全不需要 "type"
业务分类参数，所以不存在猜测取值的问题——唯一的不确定性只是"有数据
时具体长什么样"。`,
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
		raw, err := client.GetBSPUpdateStatus(endpoint)
		if err != nil {
			return err
		}
		printRawJSON(raw)
		return nil
	},
}
