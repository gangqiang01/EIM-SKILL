package repo

// LinuxPackage is a Linux software package stored in the repo server,
// available for OTA install/update to devices (see RepoLinuxPkg entity in
// m2m-management-edge).
type LinuxPackage struct {
	Rlid        int64  `json:"rlid"`
	ProductName string `json:"productname"`
	Type        string `json:"type"`
	Filename    string `json:"filename"`
	Version     string `json:"version"`
	Ts          int64  `json:"ts"`
	Address     string `json:"address"`
	Tags        string `json:"tags"`
	Description string `json:"description"`
	Size        int64  `json:"size"`
}
