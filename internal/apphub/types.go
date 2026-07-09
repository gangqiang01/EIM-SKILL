package apphub

// Device is a device as returned by the AppHub/DeviceLink device manager API.
type Device struct {
	Endpoint         string `json:"endpoint"`
	IsOnline         bool   `json:"isonline"`
	RegistrationDate string `json:"registrationDate"`
	UpdateDate       string `json:"updateDate"`
	IPAddress        string `json:"ipaddress"`
	DevName          string `json:"devName"`
	BoardName        string `json:"boardName"`
	OS               string `json:"os"`
	SoftwareStatus   int    `json:"softwarestatus"`
	HardwareStatus   int    `json:"hardwarestatus"`
	SubDeviceStatus  string `json:"subDeviceStatus"`
	IsBind           bool   `json:"isBind"`
	IthingsEnable    string `json:"ithingsEnable"`
	DeviceStatus     int    `json:"deviceStatus"`
	Details          string `json:"details"`
	Lng              string `json:"lng"`
	Lat              string `json:"lat"`
}

// DeviceGroup is a device group as returned by the group manager API.
type DeviceGroup struct {
	GID    int    `json:"gid"`
	Name   string `json:"name"`
	Desc   string `json:"desc"`
	Type   string `json:"type"`
	Total  int    `json:"total"`
	Online int    `json:"online"`
	Ts     int64  `json:"ts"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Status       string `json:"status"`
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
}

type statusResponse struct {
	Status string `json:"status"`
	ErrMsg string `json:"errmsg"`
}

// MonitorDataPoint is a single CPU or memory monitoring sample for a device.
type MonitorDataPoint struct {
	ID    int64   `json:"id"`
	Value float64 `json:"value"`
	Ts    int64   `json:"ts"`
}

type setGroupsRequest struct {
	Endpoint   string   `json:"endpoint"`
	GroupNames []string `json:"groupnames"`
}

type execActionRequest struct {
	ID    int    `json:"id"`
	Value string `json:"value"`
}

type actionValue struct {
	PluginName string `json:"pluginname"`
	Actions    string `json:"actions"`
}

// contentResponse wraps LwM2M "read"-style responses, where the actual
// payload is a JSON string nested in content.value.
type contentResponse struct {
	Status  string `json:"status"`
	Content struct {
		ID    int    `json:"id"`
		Value string `json:"value"`
	} `json:"content"`
}

// InstalledLinuxPackage is a software package already installed on a Linux
// device, as reported by the device's devicecontrol object (60001/0/28300).
type InstalledLinuxPackage struct {
	AppName     string `json:"appname"`
	Description string `json:"description"`
	PkgName     string `json:"pkgname"`
	PkgType     string `json:"pkgtype"`
	Version     string `json:"version"`
	Status      string `json:"status"`
	UpdateTime  string `json:"updatetime"`
}

type deployLinuxPackageRequest struct {
	Taskname     string `json:"taskname"`
	ProductName  string `json:"productname"`
	Version      string `json:"version"`
	DownloadPath string `json:"downloadpath"`
	Type         string `json:"type"`
}

type deployLinuxPackageGroupRequest struct {
	GroupName    string `json:"groupname"`
	Taskname     string `json:"taskname"`
	ProductName  string `json:"productname"`
	Version      string `json:"version"`
	DownloadPath string `json:"downloadpath"`
	Type         string `json:"type"`
	LabelName    string `json:"labelName"`
}
