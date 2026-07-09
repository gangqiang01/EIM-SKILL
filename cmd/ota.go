package cmd

import (
	"fmt"
	"time"

	"github.com/gangqiang01/EIM-SKILL/internal/repo"
	"github.com/spf13/cobra"
)

var otaCmd = &cobra.Command{
	Use:   "ota",
	Short: "OTA-deploy Linux software packages to devices via the repo server",
	Long: `ota wraps the OTA workflow for Linux devices: list the packages available
in the repo server, list what's already installed on a device, then push an
install/update to a single device or a whole device group.`,
}

func init() {
	rootCmd.AddCommand(otaCmd)
	otaCmd.AddCommand(otaListAvailableCmd)
	otaCmd.AddCommand(otaListInstalledCmd)
	otaCmd.AddCommand(otaApplyCmd)
	otaCmd.AddCommand(otaApplyGroupCmd)

	otaApplyGroupCmd.Flags().String("label", "", "only target devices with this label (alternative/addition to the group name)")
}

var otaListAvailableCmd = &cobra.Command{
	Use:   "list-available",
	Short: "List Linux packages available in the repo server (OTA-able apps)",
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

var otaListInstalledCmd = &cobra.Command{
	Use:   "list-installed <endpoint>",
	Short: "List the software packages currently installed on a Linux device",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		pkgs, err := client.GetInstalledLinuxPackages(args[0])
		if err != nil {
			return err
		}
		printJSON(pkgs)
		return nil
	},
}

// findLinuxPackage looks up the repo package matching productName and
// version, returning an error listing the available versions if not found.
func findLinuxPackage(pkgs []repo.LinuxPackage, productName, version string) (*repo.LinuxPackage, error) {
	var availableVersions []string
	for i := range pkgs {
		if pkgs[i].ProductName != productName {
			continue
		}
		if pkgs[i].Version == version {
			return &pkgs[i], nil
		}
		availableVersions = append(availableVersions, pkgs[i].Version)
	}
	if len(availableVersions) == 0 {
		return nil, fmt.Errorf("no package named %q found in repo", productName)
	}
	return nil, fmt.Errorf("package %q has no version %q in repo, available versions: %v", productName, version, availableVersions)
}

func otaTaskname() string {
	return fmt.Sprintf("ota-%d", time.Now().UnixMilli())
}

var otaApplyCmd = &cobra.Command{
	Use:   "apply <endpoint> <product-name> <version>",
	Short: "Install or update a Linux package on a single device",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		endpoint, productName, version := args[0], args[1], args[2]

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
		pkg, err := findLinuxPackage(pkgs, productName, version)
		if err != nil {
			return err
		}

		downloadPath := pkg.Address + "/" + pkg.Filename
		taskname := otaTaskname()
		if err := client.DeployLinuxPackage(endpoint, taskname, productName, version, downloadPath); err != nil {
			return err
		}
		printJSON(map[string]string{
			"status":       "CHANGED",
			"taskname":     taskname,
			"downloadpath": downloadPath,
		})
		return nil
	},
}

var otaApplyGroupCmd = &cobra.Command{
	Use:   "apply-group <group-name> <product-name> <version>",
	Short: "Install or update a Linux package on every device in a group",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		groupName, productName, version := args[0], args[1], args[2]
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
		pkg, err := findLinuxPackage(pkgs, productName, version)
		if err != nil {
			return err
		}

		downloadPath := pkg.Address + "/" + pkg.Filename
		taskname := otaTaskname()
		if err := client.DeployLinuxPackageToGroup(groupName, taskname, productName, version, downloadPath, label); err != nil {
			return err
		}
		printJSON(map[string]string{
			"status":       "CHANGED",
			"taskname":     taskname,
			"downloadpath": downloadPath,
		})
		return nil
	},
}
