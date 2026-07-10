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

// Storage is a storage backend configured on the repo server (m2m-management-edge
// StorageController), used to store OTA packages, app icons, etc. GET /storage
// never returns the secret key.
type Storage struct {
	SID       int64  `json:"sid"`
	Type      string `json:"type"`
	BlobURL   string `json:"bloburl"`
	Bucket    string `json:"blobbucket"`
	AccessKey string `json:"blobaccesskey"`
	Choose    int    `json:"choose"` // 1 = active, 0 = inactive
	Ts        int64  `json:"ts"`
}

// addStorageRequest is the untyped body POST /storage expects; which fields
// matter depends on Platform ("azure" uses Container+ConnectionString,
// anything else - minio/s3/oss - uses BlobURL/Bucket/AccessKey/SecretKey).
type addStorageRequest struct {
	Platform         string `json:"platform"`
	BlobURL          string `json:"bloburl,omitempty"`
	Bucket           string `json:"bucket,omitempty"`
	AccessKey        string `json:"accesskey,omitempty"`
	SecretKey        string `json:"secretkey,omitempty"`
	Container        string `json:"container,omitempty"`
	ConnectionString string `json:"connectionString,omitempty"`
}
