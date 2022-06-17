package moynalog

const (
	UserAgent  string = "Mozilla/5.0 (Macintosh; Intel Mac OS X 11_2_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.192 Safari/537.36"
	SourceType        = "WEB"
	AppVersion        = "1.0.0"
)

type deviceInfo struct {
	Type        string `json:"sourceType"`
	DeviceId    string `json:"sourceDeviceId"`
	AppVersion  string `json:"appVersion"`
	MetaDetails struct {
		UserAgent string `json:"userAgent"`
	} `json:"metaDetails"`
}

func NewDeviceInfo(deviceId string) *deviceInfo {
	deviceInfo := &deviceInfo{
		Type:       SourceType,
		DeviceId:   deviceId,
		AppVersion: AppVersion,
	}

	deviceInfo.MetaDetails.UserAgent = UserAgent
	return deviceInfo
}
