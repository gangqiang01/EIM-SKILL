package cmd

import (
	"fmt"

	"github.com/gangqiang01/EIM-SKILL/internal/apphub"
	"github.com/spf13/cobra"
)

// pageResult is the shared "data" payload for the paged asset/record query
// commands: count is the total matching rows across all pages (so the caller
// can tell when one page isn't everything), data is the current page's rows.
type pageResult struct {
	Count int64       `json:"count"`
	Data  interface{} `json:"data"`
}

// resolveDeviceOrEmpty resolves identifier to an endpoint id, returning "" for
// an empty identifier. Unlike resolveDeviceEndpoint (which errors on a miss)
// it only validates when identifier is non-empty, so these commands can accept
// an optional --device filter.
func resolveDeviceOrEmpty(client *apphub.Client, identifier string) (string, error) {
	if identifier == "" {
		return "", nil
	}
	return resolveDeviceEndpoint(client, identifier)
}

var assetCmd = &cobra.Command{
	Use:   "asset",
	Short: "Query device-management records: asset ledger, maintenance, inspection, repair, spare parts",
	Long: `Query the five device-management record families kept by DeviceOn-EIM:

  asset      设备台账  - one row per device (sn/vendor/manufacturer/asset status/
             warranty dates/repair & maintenance mode)
  maintenance 保养   - maintenance/servicing work records
  inspection  巡检   - inspection rounds
  repair     维修   - fault/repair records
  spare      备品    - spare-part inventory

Every list command supports filtering by device name or endpoint id
(--device <endpoint-or-name>), a keyword (--keywords, which the server matches
against the record's main text field), and for the record families a status
(--status). All are read-only queries.

设备管理台账的五个查询族：设备台账(asset)、保养(maintenance)、巡检(inspection)、
维修(repair)、备品(spare)。每个 list 命令都支持按设备名称或 endpoint id 过滤
（--device）、关键字（--keywords，服务端按记录的主要内容字段模糊匹配）、以及
记录类的状态（--status）。均为只读查询。`,
}

var assetListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the device-asset (设备台账) rows, or one device's record by name",
	Long: `List device-asset (设备台账) rows.

- With --device <endpoint-or-name>: return that one device's ledger record
  (the ledger's byPage endpoint can't filter by device, so this resolves the
  name to an endpoint id and fetches that device's record directly).
- Otherwise: list all ledger rows; --keywords is an optional server-side
  substring match on sn / vendor / manufacturer (not the device name).

列出设备台账。
- 带 --device <endpoint-or-name>：返回该设备自己的台账记录（台账 byPage 接口
  不支持按设备过滤，所以先把名称解析成 endpoint id，再直接查该设备的台账）。
- 不带时：列出所有台账行；--keywords 可选，服务端按 sn/厂商/制造商 模糊匹配
  （不是设备名称）。

Example 示例:
  eim-skill asset list
  eim-skill asset list --device 146打包机
  eim-skill asset list --keywords dell

Example output 输出示例 (trimmed 已截断):
  {
    "status": 200, "msg": "",
    "data": {
      "count": 3,
      "data": [
        {"id":1,"deviceId":"C4:00:AD:11:DA:97","devName":"146打包机","sn":"SN123",
         "vendor":"Dell","manufacturer":"Dell","assetStatus":"在用",
         "warrantyEndDate":1893427200000,"warrantyStatus":"在保"}
      ]
    }
  }

Each row's date fields (purchaseDate/warrantyEndDate/ts) are Unix
milliseconds. warrantyStatus is derived server-side: "" (no date), "在保"
(in warranty), or "过保" (out of warranty).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		page, _ := cmd.Flags().GetInt("page")
		limit, _ := cmd.Flags().GetInt("limit")
		keywords, _ := cmd.Flags().GetString("keywords")
		device, _ := cmd.Flags().GetString("device")
		if page < 1 {
			return fmt.Errorf("--page must be >= 1")
		}

		client, err := newClient()
		if err != nil {
			return err
		}

		if device != "" {
			// Single-device lookup: the byPage endpoint can't filter by device,
			// so resolve the name and fetch that device's record directly. The
			// single-record endpoint doesn't join devName (unlike byPage), so
			// fill it in from the name the user gave us when they passed a name
			// rather than an endpoint id.
			deviceID, err := resolveDeviceOrEmpty(client, device)
			if err != nil {
				return err
			}
			info, err := client.GetDeviceAssetInfo(deviceID)
			if err != nil {
				return err
			}
			if info.DevName == "" && device != deviceID {
				info.DevName = device
			}
			printJSON(pageResult{Count: 1, Data: info})
			return nil
		}

		count, rows, err := client.ListDeviceAssetInfo(page, limit, keywords)
		if err != nil {
			return err
		}
		printJSON(pageResult{Count: count, Data: rows})
		return nil
	},
}

var assetMaintenanceCmd = &cobra.Command{
	Use:   "maintenance",
	Short: "Query 保养 (maintenance) records",
}

var assetMaintenanceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List 保养 (maintenance) records, filtered by device / keyword / status",
	Long: `List 保养 (maintenance) records. --keywords matches the record's work
content (content). --status: 0=未保养, 1=已保养.

列出保养记录。--keywords 按工作内容(content)匹配；--status：0=未保养，1=已保养。

Example 示例:
  eim-skill asset maintenance list
  eim-skill asset maintenance list --device 146打包机
  eim-skill asset maintenance list --status 1

Example output 输出示例 (trimmed 已截断):
  {
    "status": 200, "msg": "",
    "data": {
      "count": 1,
      "data": [
        {"id":1,"deviceId":"C4:00:AD:11:DA:97","deviceName":"146打包机",
         "startTime":1719907200000,"provider":"厂内","content":"更换滤芯",
         "status":1}
      ]
    }
  }`,
	RunE: func(cmd *cobra.Command, args []string) error {
		page, _ := cmd.Flags().GetInt("page")
		limit, _ := cmd.Flags().GetInt("limit")
		device, _ := cmd.Flags().GetString("device")
		keywords, _ := cmd.Flags().GetString("keywords")
		status, _ := cmd.Flags().GetString("status")
		if page < 1 {
			return fmt.Errorf("--page must be >= 1")
		}
		if status != "" && status != "0" && status != "1" {
			return fmt.Errorf("--status must be 0 or 1")
		}

		client, err := newClient()
		if err != nil {
			return err
		}
		deviceID, err := resolveDeviceOrEmpty(client, device)
		if err != nil {
			return err
		}
		count, rows, err := client.ListMaintenanceRecords(page, limit, deviceID, keywords, status)
		if err != nil {
			return err
		}
		printJSON(pageResult{Count: count, Data: rows})
		return nil
	},
}

var assetInspectionCmd = &cobra.Command{
	Use:   "inspection",
	Short: "Query 巡检 (inspection) records",
}

var assetInspectionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List 巡检 (inspection) records, filtered by device / keyword / status",
	Long: `List 巡检 (inspection) records. --keywords matches the record's remark.
--status: 0=未巡检, 1=已巡检.

列出巡检记录。--keywords 按备注(remark)匹配；--status：0=未巡检，1=已巡检。

Example 示例:
  eim-skill asset inspection list
  eim-skill asset inspection list --device 146打包机 --status 1

Example output 输出示例 (trimmed 已截断):
  {
    "status": 200, "msg": "",
    "data": {
      "count": 1,
      "data": [
        {"id":1,"deviceId":"C4:00:AD:11:DA:97","deviceName":"146打包机",
         "date":1719907200000,"responsibleParty":"张三","status":1}
      ]
    }
  }`,
	RunE: func(cmd *cobra.Command, args []string) error {
		page, _ := cmd.Flags().GetInt("page")
		limit, _ := cmd.Flags().GetInt("limit")
		device, _ := cmd.Flags().GetString("device")
		keywords, _ := cmd.Flags().GetString("keywords")
		status, _ := cmd.Flags().GetString("status")
		if page < 1 {
			return fmt.Errorf("--page must be >= 1")
		}
		if status != "" && status != "0" && status != "1" {
			return fmt.Errorf("--status must be 0 or 1")
		}

		client, err := newClient()
		if err != nil {
			return err
		}
		deviceID, err := resolveDeviceOrEmpty(client, device)
		if err != nil {
			return err
		}
		count, rows, err := client.ListInspectionRecords(page, limit, deviceID, keywords, status)
		if err != nil {
			return err
		}
		printJSON(pageResult{Count: count, Data: rows})
		return nil
	},
}

var assetRepairCmd = &cobra.Command{
	Use:   "repair",
	Short: "Query 维修 (repair) records",
}

var assetRepairListCmd = &cobra.Command{
	Use:   "list",
	Short: "List 维修 (repair) records, filtered by device / keyword",
	Long: `List 维修 (repair) records. --keywords matches the fault description
(description). Repair records have no status field.

列出维修记录。--keywords 按故障描述(description)匹配。维修记录没有状态字段。

Example 示例:
  eim-skill asset repair list
  eim-skill asset repair list --device 146打包机

Example output 输出示例 (trimmed 已截断):
  {
    "status": 200, "msg": "",
    "data": {
      "count": 1,
      "data": [
        {"id":1,"deviceId":"C4:00:AD:11:DA:97","deviceName":"146打包机",
         "description":"电机不转","provider":"厂家","result":"已修复",
         "spareParts":"SP001"}
      ]
    }
  }`,
	RunE: func(cmd *cobra.Command, args []string) error {
		page, _ := cmd.Flags().GetInt("page")
		limit, _ := cmd.Flags().GetInt("limit")
		device, _ := cmd.Flags().GetString("device")
		keywords, _ := cmd.Flags().GetString("keywords")
		if page < 1 {
			return fmt.Errorf("--page must be >= 1")
		}

		client, err := newClient()
		if err != nil {
			return err
		}
		deviceID, err := resolveDeviceOrEmpty(client, device)
		if err != nil {
			return err
		}
		count, rows, err := client.ListRepairRecords(page, limit, deviceID, keywords)
		if err != nil {
			return err
		}
		printJSON(pageResult{Count: count, Data: rows})
		return nil
	},
}

var assetSpareCmd = &cobra.Command{
	Use:   "spare",
	Short: "Query 备品 (spare-part) inventory",
}

var assetSpareListCmd = &cobra.Command{
	Use:   "list",
	Short: "List 备品 (spare-part) rows, filtered by device / keyword / status",
	Long: `List 备品 (spare-part) inventory rows. --keywords matches name / sn /
vendor. --status is a free string (the part's own status field). Each row
includes deviceNames/deviceIds - the devices currently associated with the
part.

列具备品。--keywords 按名称/SN/厂商匹配；--status 为自由字符串（备品自身状态
字段）。每行包含 deviceNames/deviceIds（当前使用该备品的设备）。

Example 示例:
  eim-skill asset spare list
  eim-skill asset spare list --keywords 滤芯
  eim-skill asset spare list --device 146打包机

Example output 输出示例 (trimmed 已截断):
  {
    "status": 200, "msg": "",
    "data": {
      "count": 1,
      "data": [
        {"id":1,"name":"滤芯","sn":"SP001","vendor":"某厂","status":"在库",
         "deviceNames":["146打包机"],"deviceIds":["C4:00:AD:11:DA:97"]}
      ]
    }
  }`,
	RunE: func(cmd *cobra.Command, args []string) error {
		page, _ := cmd.Flags().GetInt("page")
		limit, _ := cmd.Flags().GetInt("limit")
		device, _ := cmd.Flags().GetString("device")
		keywords, _ := cmd.Flags().GetString("keywords")
		status, _ := cmd.Flags().GetString("status")
		if page < 1 {
			return fmt.Errorf("--page must be >= 1")
		}

		client, err := newClient()
		if err != nil {
			return err
		}
		deviceID, err := resolveDeviceOrEmpty(client, device)
		if err != nil {
			return err
		}
		count, rows, err := client.ListSpareParts(page, limit, keywords, deviceID, status)
		if err != nil {
			return err
		}
		printJSON(pageResult{Count: count, Data: rows})
		return nil
	},
}

func init() {
	rootCmd.AddCommand(assetCmd)
	assetCmd.AddCommand(assetListCmd)
	assetCmd.AddCommand(assetMaintenanceCmd)
	assetCmd.AddCommand(assetInspectionCmd)
	assetCmd.AddCommand(assetRepairCmd)
	assetCmd.AddCommand(assetSpareCmd)

	assetMaintenanceCmd.AddCommand(assetMaintenanceListCmd)
	assetInspectionCmd.AddCommand(assetInspectionListCmd)
	assetRepairCmd.AddCommand(assetRepairListCmd)
	assetSpareCmd.AddCommand(assetSpareListCmd)

	// asset list: --device does a single-device lookup (by name or endpoint);
	// --keywords filters the full listing by sn/vendor/manufacturer.
	assetListCmd.Flags().Int("page", 1, "1-based page number (页码，从1开始)")
	assetListCmd.Flags().Int("limit", 20, "rows per page (每页条数)")
	assetListCmd.Flags().String("device", "", "return just this device's ledger record - endpoint id or device name (只查该设备的台账，填 endpoint id 或设备名称)")
	assetListCmd.Flags().String("keywords", "", "substring match on sn / vendor / manufacturer (按 sn/厂商/制造商 模糊匹配)")

	// record families: device + keyword + (status where applicable).
	for _, c := range []*cobra.Command{assetMaintenanceListCmd, assetInspectionListCmd, assetRepairListCmd, assetSpareListCmd} {
		c.Flags().Int("page", 1, "1-based page number (页码，从1开始)")
		c.Flags().Int("limit", 20, "rows per page (每页条数)")
		c.Flags().String("device", "", "only records for this device - endpoint id or device name (只看该设备的记录，填 endpoint id 或设备名称)")
		c.Flags().String("keywords", "", "substring match on the record's main text field (按记录主要内容字段模糊匹配)")
	}

	assetMaintenanceListCmd.Flags().String("status", "", "0=未保养, 1=已保养 (0=未保养，1=已保养)")
	assetInspectionListCmd.Flags().String("status", "", "0=未巡检, 1=已巡检 (0=未巡检，1=已巡检)")
	assetSpareListCmd.Flags().String("status", "", "the part's status string (备品状态，自由字符串)")
}
