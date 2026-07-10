package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var storageCmd = &cobra.Command{
	Use:   "storage",
	Short: "Configure the repo server's storage backend (local MinIO / AWS S3 / Alibaba OSS / Azure Blob)",
	Long: `The repo server (m2m-management-edge) stores OTA packages, app icons, etc.
in a pluggable storage backend. "Local" deployments use a bundled MinIO
instance (there is no raw-filesystem mode); MinIO/AWS S3/Alibaba OSS are
all configured the same way since they share an S3-compatible API.`,
}

func init() {
	rootCmd.AddCommand(storageCmd)
	storageCmd.AddCommand(storageListCmd)
	storageCmd.AddCommand(storageAddS3Cmd)
	storageCmd.AddCommand(storageAddAzureCmd)
	storageCmd.AddCommand(storageActivateCmd)
	storageCmd.AddCommand(storageDeleteCmd)
}

var storageListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured storage backends (secret keys are never returned)",
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
		storages, err := repoClient.ListStorages()
		if err != nil {
			return err
		}
		printJSON(storages)
		return nil
	},
}

var storageAddS3Cmd = &cobra.Command{
	Use:   "add-s3 <platform> <endpoint-url> <bucket> <access-key> <secret-key>",
	Short: "Register a MinIO / AWS S3 / Alibaba OSS storage backend",
	Long: `platform is stored verbatim as this backend's type - the server enforces
no fixed enum, so use a consistent convention such as "minio", "s3", or "oss".`,
	Args: cobra.ExactArgs(5),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		repoClient, err := newRepoClient(client)
		if err != nil {
			return err
		}
		if err := repoClient.AddS3CompatibleStorage(args[0], args[1], args[2], args[3], args[4]); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var storageAddAzureCmd = &cobra.Command{
	Use:   "add-azure <container> <connection-string>",
	Short: "Register an Azure Blob storage backend",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		repoClient, err := newRepoClient(client)
		if err != nil {
			return err
		}
		if err := repoClient.AddAzureBlobStorage(args[0], args[1]); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var storageActivateCmd = &cobra.Command{
	Use:   "activate <sid>",
	Short: "Re-validate and activate a storage backend (used for all subsequent uploads)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sid, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("sid must be an integer: %w", err)
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		repoClient, err := newRepoClient(client)
		if err != nil {
			return err
		}
		if err := repoClient.SetActiveStorage(sid); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var storageDeleteCmd = &cobra.Command{
	Use:   "delete <sid>",
	Short: "Delete a storage backend and every repo object stored in it",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sid, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("sid must be an integer: %w", err)
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		repoClient, err := newRepoClient(client)
		if err != nil {
			return err
		}
		if err := repoClient.DeleteStorage(sid); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}
