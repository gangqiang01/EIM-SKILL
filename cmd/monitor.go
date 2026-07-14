package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// parseOnOff parses an "on"/"off" (or "true"/"false") toggle argument.
func parseOnOff(s string) (bool, error) {
	switch s {
	case "on", "true":
		return true, nil
	case "off", "false":
		return false, nil
	default:
		return false, fmt.Errorf("expected \"on\" or \"off\", got %q", s)
	}
}

var deviceSetMonitorCmd = &cobra.Command{
	Use:   "set-monitor <endpoint-or-name> <on|off>",
	Short: "Enable or disable overall hardware monitoring (CPU/memory/storage/network/battery thresholds)",
	Args:  cobra.ExactArgs(2),
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
		if err := client.SetMonitorEnabled(endpoint, enabled); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var deviceBoardInfoCmd = &cobra.Command{
	Use:   "get-monitor <endpoint-or-name>",
	Short: "Get a device's board-level monitoring detail (best effort, see NOTE in internal/apphub)",
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
		info, err := client.GetBoardInfo(endpoint)
		if err != nil {
			return err
		}
		printJSON(info)
		return nil
	},
}

// threshold

var deviceThresholdCmd = &cobra.Command{
	Use:   "threshold",
	Short: "Get/set CPU, memory, storage, network, and battery alert thresholds",
}

var deviceThresholdGetCmd = &cobra.Command{
	Use:   "get <endpoint-or-name>",
	Short: "Get the device's current threshold configuration",
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
		settings, err := client.GetThresholds(endpoint)
		if err != nil {
			return err
		}
		printJSON(settings)
		return nil
	},
}

var deviceThresholdSetCmd = &cobra.Command{
	Use:   "set <endpoint-or-name> <metric> <on|off>",
	Short: "Enable/disable and configure one threshold metric",
	Long: `Supported metrics: cpu_temp, cpu_usage, memory_usage, storage_usage,
battery_available, network_upload_speed, network_download_speed.`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		metric := args[1]
		enabled, err := parseOnOff(args[2])
		if err != nil {
			return err
		}
		warn, err := cmd.Flags().GetFloat64("warn")
		if err != nil {
			return err
		}
		errorLevel, err := cmd.Flags().GetFloat64("error")
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
		if err := client.SetThreshold(endpoint, metric, enabled, warn, errorLevel); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

// process monitoring

var deviceProcessCmd = &cobra.Command{
	Use:   "process",
	Short: "List processes and control application/process monitoring",
}

var deviceProcessListCmd = &cobra.Command{
	Use:   "list <endpoint-or-name>",
	Short: "List running processes/threads and their monitor status",
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
		procs, err := client.ListProcesses(endpoint)
		if err != nil {
			return err
		}
		printJSON(procs)
		return nil
	},
}

var deviceProcessMonitoredCmd = &cobra.Command{
	Use:   "monitored <endpoint-or-name>",
	Short: "List the processes currently selected for application/process monitoring",
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
		procs, err := client.GetMonitoredProcesses(endpoint)
		if err != nil {
			return err
		}
		printJSON(procs)
		return nil
	},
}

var deviceProcessSetMonitorCmd = &cobra.Command{
	Use:   "set-monitor <endpoint-or-name> <on|off>",
	Short: "Enable or disable application/process monitoring",
	Args:  cobra.ExactArgs(2),
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
		if err := client.SetProcessMonitorEnabled(endpoint, enabled); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

// docker monitoring

var deviceDockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Get Docker container info and control Docker monitoring",
}

var deviceDockerInfoCmd = &cobra.Command{
	Use:   "info <endpoint-or-name>",
	Short: "Get the Docker compose/swarm/image groups deployed on a device",
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
		groups, err := client.GetDockerInfo(endpoint)
		if err != nil {
			return err
		}
		printJSON(groups)
		return nil
	},
}

var deviceDockerSetMonitorCmd = &cobra.Command{
	Use:   "set-monitor <endpoint-or-name> <on|off>",
	Short: "Enable or disable Docker container monitoring",
	Args:  cobra.ExactArgs(2),
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
		if err := client.SetDockerMonitorEnabled(endpoint, enabled); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

// USB peripheral monitoring

var deviceUsbCmd = &cobra.Command{
	Use:   "usb",
	Short: "Get/set USB peripheral change monitoring",
}

var deviceUsbGetMonitorCmd = &cobra.Command{
	Use:   "get-monitor <endpoint-or-name>",
	Short: "Report whether USB peripheral change monitoring is enabled",
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
		enabled, err := client.GetUSBMonitorEnabled(endpoint)
		if err != nil {
			return err
		}
		printJSON(map[string]bool{"enabled": enabled})
		return nil
	},
}

var deviceUsbSetMonitorCmd = &cobra.Command{
	Use:   "set-monitor <endpoint-or-name> <on|off>",
	Short: "Enable or disable USB peripheral change monitoring",
	Args:  cobra.ExactArgs(2),
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
		if err := client.SetUSBMonitorEnabled(endpoint, enabled); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var deviceUsbGetStorageCmd = &cobra.Command{
	Use:   "get-storage <endpoint-or-name>",
	Short: "Report whether USB mass-storage access is enabled",
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
		enabled, err := client.GetUSBStorageEnabled(endpoint)
		if err != nil {
			return err
		}
		printJSON(map[string]bool{"enabled": enabled})
		return nil
	},
}

var deviceUsbSetStorageCmd = &cobra.Command{
	Use:   "set-storage <endpoint-or-name> <on|off>",
	Short: "Enable or disable USB mass-storage access",
	Args:  cobra.ExactArgs(2),
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
		if err := client.SetUSBStorageEnabled(endpoint, enabled); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var deviceSyncTimeCmd = &cobra.Command{
	Use:   "sync-time <endpoint-or-name>",
	Short: "Trigger an immediate clock sync on a device (best effort, see NOTE in internal/apphub)",
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
		if err := client.SyncTime(endpoint); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

func init() {
	deviceCmd.AddCommand(deviceSetMonitorCmd)
	deviceCmd.AddCommand(deviceBoardInfoCmd)
	deviceCmd.AddCommand(deviceSyncTimeCmd)

	deviceCmd.AddCommand(deviceThresholdCmd)
	deviceThresholdCmd.AddCommand(deviceThresholdGetCmd)
	deviceThresholdCmd.AddCommand(deviceThresholdSetCmd)
	deviceThresholdSetCmd.Flags().Float64("warn", 0, "warning level")
	deviceThresholdSetCmd.Flags().Float64("error", 0, "error level")

	deviceCmd.AddCommand(deviceProcessCmd)
	deviceProcessCmd.AddCommand(deviceProcessListCmd)
	deviceProcessCmd.AddCommand(deviceProcessMonitoredCmd)
	deviceProcessCmd.AddCommand(deviceProcessSetMonitorCmd)

	deviceCmd.AddCommand(deviceDockerCmd)
	deviceDockerCmd.AddCommand(deviceDockerInfoCmd)
	deviceDockerCmd.AddCommand(deviceDockerSetMonitorCmd)

	deviceCmd.AddCommand(deviceUsbCmd)
	deviceUsbCmd.AddCommand(deviceUsbGetMonitorCmd)
	deviceUsbCmd.AddCommand(deviceUsbSetMonitorCmd)
	deviceUsbCmd.AddCommand(deviceUsbGetStorageCmd)
	deviceUsbCmd.AddCommand(deviceUsbSetStorageCmd)
}
