package models

import (
	"encoding/json"
)

type DevicePlatform string

const (
	IOS     DevicePlatform = "IOS"
	MacOS   DevicePlatform = "MACOS"
	Android DevicePlatform = "ANDROID"
	Windows DevicePlatform = "WINDOWS"
	Linux   DevicePlatform = "LINUX"
	Other   DevicePlatform = "OTHER"
)

type Device struct {
	Generic

	Platform DevicePlatform
	Token    string `gorm:"not null; unique"`

	IsBanned bool `gorm:"not null; default:false"`

	WalletAddress  string `gorm:"not null; unique"`
	WalletEntropy  []byte `gorm:"not null; unique"`
	SubscriptionId *int64
	CurrentBalance int64 `gorm:"not null; default:-1"`
}

func (d Device) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID            uint   `json:"id"`
		Platform      string `json:"platform"`
		Token         string `json:"token"`
		IsBanned      bool   `json:"is_banned"`
		WalletAddress string `json:"wallet_address"`
	}{
		ID:            d.ID,
		Platform:      string(d.Platform),
		Token:         d.Token,
		IsBanned:      d.IsBanned,
		WalletAddress: d.WalletAddress,
	})
}
