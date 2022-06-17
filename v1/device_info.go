package moynalog

const (
	UserAgent  string = "Mozilla/5.0 (Macintosh; Intel Mac OS X 11_2_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.192 Safari/537.36"
	SourceType string = "WEB"
	AppVersion string = "1.0.0"
)

type DeviceInfo struct {
	Type        string `json:"sourceType"`
	DeviceID    string `json:"sourceDeviceId"`
	AppVersion  string `json:"appVersion"`
	MetaDetails struct {
		UserAgent string `json:"userAgent"`
	} `json:"metaDetails"`
}

func NewDeviceInfo(deviceID string) *DeviceInfo {
	deviceInfo := &DeviceInfo{
		Type:       SourceType,
		DeviceID:   deviceID,
		AppVersion: AppVersion,
	}

	deviceInfo.MetaDetails.UserAgent = UserAgent
	return deviceInfo
}
