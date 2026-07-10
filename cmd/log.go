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
	Short: "Retrieve device system logs, operation records, and agent (client) logs",
}

var logSystemCmd = &cobra.Command{
	Use:   "system",
	Short: "Query and delete device system log entries",
}

var logOperationsCmd = &cobra.Command{
	Use:   "operations",
	Short: "Query and delete operation records (an audit log of admin actions)",
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
"log agent download".`,
}

func init() {
	rootCmd.AddCommand(logCmd)
	logCmd.AddCommand(logSystemCmd)
	logCmd.AddCommand(logOperationsCmd)
	logCmd.AddCommand(logAgentCmd)

	logSystemCmd.AddCommand(logSystemListCmd)
	logSystemCmd.AddCommand(logSystemDeleteCmd)
	logSystemCmd.AddCommand(logSystemDeleteByDeviceCmd)
	logSystemCmd.AddCommand(logSystemBatchDeleteCmd)

	logOperationsCmd.AddCommand(logOperationsListCmd)
	logOperationsCmd.AddCommand(logOperationsDeleteCmd)
	logOperationsCmd.AddCommand(logOperationsDeleteAllCmd)
	logOperationsCmd.AddCommand(logOperationsBatchDeleteCmd)

	logAgentCmd.AddCommand(logAgentRequestCmd)
	logAgentCmd.AddCommand(logAgentStatusCmd)
	logAgentCmd.AddCommand(logAgentDownloadCmd)

	logSystemListCmd.Flags().Int("page", 1, "page number")
	logSystemListCmd.Flags().Int("limit", 20, "page size")
	logSystemListCmd.Flags().String("device-id", "", "only return entries for this device")
	logSystemListCmd.Flags().String("os", "", "only return entries for this device OS")
	logSystemListCmd.Flags().String("keywords", "", "filter by keywords")
	logSystemListCmd.Flags().Int64("begin", 0, "only return entries at or after this Unix millisecond timestamp")
	logSystemListCmd.Flags().Int64("end", 0, "only return entries at or before this Unix millisecond timestamp")

	logOperationsListCmd.Flags().Int("page", 1, "page number")
	logOperationsListCmd.Flags().Int("limit", 20, "page size")
	logOperationsListCmd.Flags().String("target", "", "filter by target (substring match)")
	logOperationsListCmd.Flags().Int64("begin", 0, "only return entries at or after this Unix millisecond timestamp")
	logOperationsListCmd.Flags().Int64("end", 0, "only return entries at or before this Unix millisecond timestamp")

	logAgentDownloadCmd.Flags().String("out", "", "file path to save the downloaded log to (default: <endpoint>.zip)")
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

var logSystemListCmd = &cobra.Command{
	Use:   "list",
	Short: "List device system log entries",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		page, _ := cmd.Flags().GetInt("page")
		limit, _ := cmd.Flags().GetInt("limit")
		deviceID, _ := cmd.Flags().GetString("device-id")
		os, _ := cmd.Flags().GetString("os")
		keywords, _ := cmd.Flags().GetString("keywords")
		begin, _ := cmd.Flags().GetInt64("begin")
		end, _ := cmd.Flags().GetInt64("end")

		client, err := newClient()
		if err != nil {
			return err
		}
		logs, err := client.ListSystemLogs(page, limit, deviceID, os, keywords, begin, end)
		if err != nil {
			return err
		}
		printJSON(logs)
		return nil
	},
}

var logSystemDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a single system log entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return err
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		if err := client.DeleteSystemLog(id); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var logSystemDeleteByDeviceCmd = &cobra.Command{
	Use:   "delete-by-device <device-id>",
	Short: "Delete every system log entry for a device",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		if err := client.DeleteSystemLogsByDevice(args[0]); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var logSystemBatchDeleteCmd = &cobra.Command{
	Use:   "batch-delete <id> [id...]",
	Short: "Delete multiple system log entries",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ids, err := parseInt64Args(args)
		if err != nil {
			return err
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		if err := client.BatchDeleteSystemLogs(ids); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var logOperationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List operation records (audit log)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		page, _ := cmd.Flags().GetInt("page")
		limit, _ := cmd.Flags().GetInt("limit")
		target, _ := cmd.Flags().GetString("target")
		begin, _ := cmd.Flags().GetInt64("begin")
		end, _ := cmd.Flags().GetInt64("end")

		client, err := newClient()
		if err != nil {
			return err
		}
		records, err := client.ListOperationRecords(page, limit, target, begin, end)
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
	Args:  cobra.ExactArgs(1),
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
	Args:  cobra.NoArgs,
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
	Args:  cobra.MinimumNArgs(1),
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
	Use:   "request <endpoint> <callback-url> <action-id>",
	Short: "Ask a device's agent to upload its client log to callback-url",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		if err := client.RequestAgentLogUpload(args[0], args[1], args[2]); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var logAgentStatusCmd = &cobra.Command{
	Use:   "status <action-id>",
	Short: "Poll the status of a previously requested agent log upload",
	Args:  cobra.ExactArgs(1),
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
	Use:   "download <endpoint>",
	Short: "Download a device agent's uploaded client log (can only be downloaded once)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		out, _ := cmd.Flags().GetString("out")
		if out == "" {
			out = strings.ReplaceAll(args[0], ":", "") + ".zip"
		}

		client, err := newClient()
		if err != nil {
			return err
		}
		data, err := client.DownloadAgentLog(args[0])
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
