package moynalog

import (
	"log"

	"github.com/denisbrodbeck/machineid"
)

const (
	SourceType string = "WEB"
	AppVersion string = "1.0.0"

	DeviceIDLen int = 21
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

	return deviceInfo
}

func generateDeviceID() string {
	id, err := machineid.ProtectedID("go-moy-nalog")
	if err != nil {
		log.Fatal(err)
	}

	if len(id) > 21 {
		return id[:DeviceIDLen]
	}

	return id
}
