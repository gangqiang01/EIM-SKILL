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

// AndroidApp is an Android APK package stored in the repo server (GET
// /repoapps, RepoApp entity).
type AndroidApp struct {
	Raid        int64  `json:"raid"`
	Filename    string `json:"filename"`
	PkgName     string `json:"pkgname"`
	VersionName string `json:"versionname"`
	VersionCode string `json:"versioncode"`
	Description string `json:"description"`
	Address     string `json:"address"`
	Tags        string `json:"tags"`
	Size        int64  `json:"size"`
}

// WindowsPackage is a Windows exe installer or zip/compressed package
// stored in the repo server (GET /repoexe, /exeByType; RepoExe entity).
// Type discriminates "exe" vs the zip variant; Tool ("Inno Setup"/"WiX
// Toolset") only applies to exe packages.
type WindowsPackage struct {
	Reid        int64  `json:"reid"`
	Filename    string `json:"filename"`
	ProjectName string `json:"projectname"`
	Version     string `json:"version"`
	Tool        string `json:"tool"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Address     string `json:"address"`
	Tags        string `json:"tags"`
	Size        int64  `json:"size"`
}

// DockerContainerSpec is one container image inside a DockerComposePackage.
type DockerContainerSpec struct {
	Rdid      int64  `json:"rdid"`
	Container string `json:"container"`
	Image     string `json:"image"`
	Version   string `json:"version"`
}

// DockerComposePackage is a docker-compose/swarm yaml stored in the repo
// server (GET /repoDockerCompose - exact casing required by the server;
// RepoDockerCompose entity). Tag is the package's product name.
type DockerComposePackage struct {
	Rdcid       int64                 `json:"rdcid"`
	Tag         string                `json:"tag"`
	Filename    string                `json:"filename"`
	Version     string                `json:"version"`
	Type        string                `json:"type"`
	Description string                `json:"description"`
	Address     string                `json:"address"`
	Containers  []DockerContainerSpec `json:"repoDockers"`
}

// GeneralFile is a generic file stored in the repo server for file-deploy
// (GET /repofiles/bytype?type=; RepoFile entity).
type GeneralFile struct {
	Rfid        int64  `json:"rfid"`
	Filename    string `json:"filename"`
	Type        string `json:"type"`
	Format      string `json:"format"`
	Description string `json:"description"`
	Address     string `json:"address"`
	Size        int64  `json:"size"`
}
