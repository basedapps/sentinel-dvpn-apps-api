package models

import (
	"encoding/json"
	"gorm.io/datatypes"
)

type ServerProtocol string

const (
	ServerProtocolWireGuard ServerProtocol = "WIREGUARD"
	ServerProtocolV2Ray     ServerProtocol = "V2RAY"
)

func (sp ServerProtocol) Value() (datatypes.JSON, error) {
	return json.Marshal(sp)
}

func (sp *ServerProtocol) Scan(value interface{}) error {
	return json.Unmarshal(value.([]byte), sp)
}

type ServerConfiguration struct {
	RemoteURL         string  `json:"remoteURL"`
	Address           string  `json:"address"`
	BandwidthDownload int64   `json:"bandwidthDownload"`
	BandwidthUpload   int64   `json:"bandwidthUpload"`
	LocationCity      string  `json:"locationCity"`
	LocationCountry   string  `json:"locationCountry"`
	LocationLat       float64 `json:"locationLat"`
	LocationLon       float64 `json:"locationLon"`
	PricePerGB        int64   `json:"pricePerGB"`
	PricePerHour      int64   `json:"pricePerHour"`
	Version           string  `json:"version"`
}

type Server struct {
	Generic

	CountryID uint    `gorm:"index;not null" json:"country_id"`
	Country   Country `json:"-"`

	CityID uint `gorm:"index;not null" json:"city_id"`
	City   City `json:"-"`

	Name             string                                  `gorm:"not null"`
	IsBanned         bool                                    `gorm:"not null; default:false"`
	IsActive         bool                                    `gorm:"not null"`
	IsIncludedInPlan bool                                    `gorm:"not null; default:false"`
	CurrentLoad      float64                                 `gorm:"not null"`
	Protocols        datatypes.JSONType[[]ServerProtocol]    `gorm:"type:json;not null"`
	Configuration    datatypes.JSONType[ServerConfiguration] `gorm:"type:json;not null"`
	Revision         int64                                   `gorm:"not null"`
}

func (s Server) MarshalJSON() ([]byte, error) {
	type serverJSON struct {
		ID            uint    `json:"id"`
		CountryID     uint    `json:"country_id"`
		CityID        uint    `json:"city_id"`
		Name          string  `json:"name"`
		Address       string  `json:"address"`
		IsAvailable   bool    `json:"is_available"`
		Load          float64 `json:"load"`
		Version       string  `json:"version"`
		Latitude      float64 `json:"latitude"`
		Longitude     float64 `json:"longitude"`
		UploadSpeed   int64   `json:"upload_speed"`
		DownloadSpeed int64   `json:"download_speed"`
		RemoteUrl     string  `json:"remote_url"`
		Protocol      string  `json:"protocol"`
	}

	server := serverJSON{
		ID:          s.ID,
		CountryID:   s.CountryID,
		CityID:      s.CityID,
		Name:        s.Name,
		Address:     s.Configuration.Data().Address,
		IsAvailable: s.IsActive,
		Load:        s.CurrentLoad,
		Version:     s.Configuration.Data().Version,
		Latitude:    s.Configuration.Data().LocationLat,
		Longitude:   s.Configuration.Data().LocationLon,
		UploadSpeed: s.Configuration.Data().BandwidthUpload,
		RemoteUrl:   s.Configuration.Data().RemoteURL,
		Protocol:    string(s.Protocols.Data()[0]),
	}

	return json.Marshal(server)
}
