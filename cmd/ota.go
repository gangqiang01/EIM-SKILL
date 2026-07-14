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
the DeviceLink web console uses.`,
}

func init() {
	rootCmd.AddCommand(otaCmd)
	otaCmd.AddCommand(otaLinuxCmd)
	otaCmd.AddCommand(otaAndroidCmd)
	otaCmd.AddCommand(otaWindowsExeCmd)
	otaCmd.AddCommand(otaWindowsZipCmd)
	otaCmd.AddCommand(otaDockerCmd)
	otaCmd.AddCommand(otaFileCmd)
}

func otaTaskname() string {
	return fmt.Sprintf("ota-%d", time.Now().UnixMilli())
}

// ==================== Linux ====================

var otaLinuxCmd = &cobra.Command{
	Use:   "linux",
	Short: "OTA-deploy Linux packages (deb/tar)",
}

func init() {
	otaLinuxCmd.AddCommand(otaLinuxListCmd)
	otaLinuxCmd.AddCommand(otaLinuxInstalledCmd)
	otaLinuxCmd.AddCommand(otaLinuxApplyCmd)
	otaLinuxCmd.AddCommand(otaLinuxApplyGroupCmd)
	otaLinuxApplyGroupCmd.Flags().String("label", "", "only target devices with this label")
}

var otaLinuxListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Linux packages available in the repo server",
	Args:  cobra.NoArgs,
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
	Args:  cobra.ExactArgs(1),
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
	Args:  cobra.ExactArgs(3),
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
	Args:  cobra.ExactArgs(3),
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
}

func init() {
	otaAndroidCmd.AddCommand(otaAndroidListCmd)
	otaAndroidCmd.AddCommand(otaAndroidInstalledCmd)
	otaAndroidCmd.AddCommand(otaAndroidApplyCmd)
	otaAndroidCmd.AddCommand(otaAndroidApplyGroupCmd)
	otaAndroidCmd.AddCommand(otaAndroidUninstallCmd)
	otaAndroidApplyGroupCmd.Flags().String("label", "", "only target devices with this label")
}

var otaAndroidListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Android APKs available in the repo server",
	Args:  cobra.NoArgs,
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
	Args:  cobra.ExactArgs(1),
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
	Args:  cobra.ExactArgs(3),
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
	Args:  cobra.ExactArgs(3),
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
	Args:  cobra.ExactArgs(2),
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
}

func init() {
	otaWindowsExeCmd.AddCommand(otaWindowsExeListCmd)
	otaWindowsExeCmd.AddCommand(otaWindowsExeInstalledCmd)
	otaWindowsExeCmd.AddCommand(otaWindowsExeInstallCmd)
	otaWindowsExeCmd.AddCommand(otaWindowsExeInstallGroupCmd)
	otaWindowsExeCmd.AddCommand(otaWindowsExeUpgradeCmd)
	otaWindowsExeCmd.AddCommand(otaWindowsExeUpgradeGroupCmd)
	otaWindowsExeCmd.AddCommand(otaWindowsExeUninstallCmd)
}

var otaWindowsExeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Windows exe installers available in the repo server",
	Args:  cobra.NoArgs,
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
	Args:  cobra.ExactArgs(1),
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
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		return applyWindowsExeSingle(args[0], args[1], args[2], false)
	},
}

var otaWindowsExeUpgradeCmd = &cobra.Command{
	Use:   "upgrade <endpoint-or-name> <project-name> <version>",
	Short: "Upgrade an already-installed exe package on a single device",
	Args:  cobra.ExactArgs(3),
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
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		return applyWindowsExeGroup(args[0], args[1], args[2], false)
	},
}

var otaWindowsExeUpgradeGroupCmd = &cobra.Command{
	Use:   "upgrade-group <group-name> <project-name> <version>",
	Short: "Upgrade an exe package on every device in a group",
	Args:  cobra.ExactArgs(3),
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
	Use:   "uninstall <endpoint-or-name> <project-name> <version> --tool <tool>",
	Short: "Uninstall an exe package from a single device",
	Args:  cobra.ExactArgs(3),
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
}

func init() {
	otaWindowsZipCmd.AddCommand(otaWindowsZipListCmd)
	otaWindowsZipCmd.AddCommand(otaWindowsZipInstalledCmd)
	otaWindowsZipCmd.AddCommand(otaWindowsZipInstallCmd)
	otaWindowsZipCmd.AddCommand(otaWindowsZipInstallGroupCmd)
	otaWindowsZipCmd.AddCommand(otaWindowsZipUpdateCmd)
	otaWindowsZipCmd.AddCommand(otaWindowsZipUpdateGroupCmd)
	otaWindowsZipCmd.AddCommand(otaWindowsZipUninstallCmd)

	for _, c := range []*cobra.Command{otaWindowsZipInstallCmd, otaWindowsZipInstallGroupCmd, otaWindowsZipUpdateCmd, otaWindowsZipUpdateGroupCmd} {
		c.Flags().String("parameter", "", "optional arguments passed to the extracted package")
	}
	otaWindowsZipInstallGroupCmd.Flags().String("label", "", "only target devices with this label")
	otaWindowsZipUpdateGroupCmd.Flags().String("label", "", "only target devices with this label")
}

var otaWindowsZipListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Windows zip packages available in the repo server",
	Args:  cobra.NoArgs,
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
	Args:  cobra.ExactArgs(1),
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

var otaWindowsZipInstallCmd = &cobra.Command{
	Use:   "install <endpoint-or-name> <project-name> <version>",
	Short: "Install a zip package on a single device",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		parameter, _ := cmd.Flags().GetString("parameter")
		return applyWindowsZipSingle(args[0], args[1], args[2], parameter, false)
	},
}

var otaWindowsZipUpdateCmd = &cobra.Command{
	Use:   "update <endpoint-or-name> <project-name> <version>",
	Short: "Update an already-installed zip package on a single device",
	Args:  cobra.ExactArgs(3),
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
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		parameter, _ := cmd.Flags().GetString("parameter")
		label, _ := cmd.Flags().GetString("label")
		return applyWindowsZipGroup(args[0], args[1], args[2], parameter, label, false)
	},
}

var otaWindowsZipUpdateGroupCmd = &cobra.Command{
	Use:   "update-group <group-name> <project-name> <version>",
	Short: "Update a zip package on every device in a group",
	Args:  cobra.ExactArgs(3),
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
	Args:  cobra.ExactArgs(3),
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
}

func init() {
	otaDockerCmd.AddCommand(otaDockerListCmd)
	otaDockerCmd.AddCommand(otaDockerApplyCmd)
	otaDockerCmd.AddCommand(otaDockerApplyGroupCmd)
	otaDockerCmd.AddCommand(otaDockerUninstallCmd)
	otaDockerApplyGroupCmd.Flags().String("label", "", "only target devices with this label")
}

var otaDockerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Docker compose/swarm packages available in the repo server",
	Args:  cobra.NoArgs,
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
	Args:  cobra.ExactArgs(3),
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
	Args:  cobra.ExactArgs(3),
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
	Args:  cobra.ExactArgs(3),
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
}

func init() {
	otaFileCmd.AddCommand(otaFileListCmd)
	otaFileCmd.AddCommand(otaFileDeployCmd)
	otaFileCmd.AddCommand(otaFileDeployGroupCmd)

	otaFileListCmd.Flags().String("type", "", "repo file category/type (required)")
	_ = otaFileListCmd.MarkFlagRequired("type")

	for _, c := range []*cobra.Command{otaFileDeployCmd, otaFileDeployGroupCmd} {
		c.Flags().String("type", "", "repo file category/type (required, matches how the file was uploaded)")
		c.Flags().Bool("exec", false, "make the deployed file executable and run it")
		_ = c.MarkFlagRequired("type")
	}
	otaFileDeployGroupCmd.Flags().String("label", "", "only target devices with this label")
}

var otaFileListCmd = &cobra.Command{
	Use:   "list <file-name>",
	Short: "List files available in the repo server for a given --type",
	Args:  cobra.NoArgs,
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
	Args:  cobra.ExactArgs(3),
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
	Args:  cobra.ExactArgs(3),
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
