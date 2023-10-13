package sentinel

import (
	"strconv"
	"time"
)

type SentinelNodePrice struct {
	Amount string `json:"amount"`
	Denom  string `json:"denom"`
}

type SentinelNode struct {
	Address        string              `json:"address"`
	GigabytePrices []SentinelNodePrice `json:"gigabyte_prices"`
	HourlyPrices   []SentinelNodePrice `json:"hourly_prices"`
	RemoteURL      string              `json:"remote_url"`
	Status         int64               `json:"status"`
}

type SentinelSessionBandwidth struct {
	Download string `json:"download"`
	Upload   string `json:"upload"`
}

func (ssb SentinelSessionBandwidth) DTO() SentinelSessionBandwidthDTO {
	download, _ := strconv.ParseInt(ssb.Download, 10, 64)
	upload, _ := strconv.ParseInt(ssb.Upload, 10, 64)

	return SentinelSessionBandwidthDTO{
		Download: download,
		Upload:   upload,
	}
}

type SentinelNodeBandwidth struct {
	Download int64 `json:"download"`
	Upload   int64 `json:"upload"`
}

func (snb SentinelNodeBandwidth) DTO() SentinelNodeBandwidthDTO {
	return SentinelNodeBandwidthDTO{
		Download: snb.Download,
		Upload:   snb.Upload,
	}
}

type SentinelNodeLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	City      string  `json:"city"`
	Country   string  `json:"country"`
}

type SentinelNodeQoS struct {
	MaxPeers int64 `json:"max_peers"`
}

type SentinelNodeStatus struct {
	Address   string                `json:"address"`
	Bandwidth SentinelNodeBandwidth `json:"bandwidth"`
	Location  SentinelNodeLocation  `json:"location"`
	Moniker   string                `json:"moniker"`
	Peers     int64                 `json:"peers"`
	QoS       SentinelNodeQoS       `json:"qos"`
	Type      int64                 `json:"type"`
	Version   string                `json:"version"`
}

type SentinelBalance struct {
	Amount string `json:"amount"`
	Denom  string `json:"denom"`
}

type SentinelTransactionEventAttribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type SentinelTransactionEvent struct {
	Type       string                              `json:"type"`
	Attributes []SentinelTransactionEventAttribute `json:"attributes"`
}

type SentinelTransaction struct {
	Height int64                      `json:"height"`
	TxHash string                     `json:"txhash"`
	Events []SentinelTransactionEvent `json:"events"`
}

type SentinelSessionStatus int64

const (
	SentinelSessionStatusUnspecified     SentinelSessionStatus = 0
	SentinelSessionStatusActive          SentinelSessionStatus = 1
	SentinelSessionStatusInactivePending SentinelSessionStatus = 2
	SentinelSessionStatusInactive        SentinelSessionStatus = 3
)

type SentinelSession struct {
	ID             int64                    `json:"id"`
	SubscriptionID int64                    `json:"subscription_id"`
	NodeAddress    string                   `json:"node_address"`
	Address        string                   `json:"address"`
	Duration       int64                    `json:"duration"`
	Bandwidth      SentinelSessionBandwidth `json:"bandwidth"`
	Status         SentinelSessionStatus    `json:"status"`
}

func (s SentinelSession) DTO() SentinelSessionDTO {
	return SentinelSessionDTO{
		ID:             s.ID,
		SubscriptionID: s.SubscriptionID,
		NodeAddress:    s.NodeAddress,
		Address:        s.Address,
		Duration:       s.Duration,
		Bandwidth:      s.Bandwidth.DTO(),
		Status:         int64(s.Status),
	}
}

type SentinelSubscriptionBase struct {
	ID         int64     `json:"id"`
	Address    string    `json:"address"`
	Status     int64     `json:"status"`
	InactiveAt time.Time `json:"inactive_at"`
}

func (ssb SentinelSubscriptionBase) DTO() SentinelSubscriptionBaseDTO {
	return SentinelSubscriptionBaseDTO{
		ID:         ssb.ID,
		Address:    ssb.Address,
		Status:     ssb.Status,
		InactiveAt: ssb.InactiveAt,
	}
}

type SentinelSubscription struct {
	Base        SentinelSubscriptionBase `json:"base"`
	PlanId      int64                    `json:"plan_id,omitempty"`
	NodeAddress string                   `json:"node_address,omitempty"`
	Deposit     SentinelBalance          `json:"deposit,omitempty"`
	Gigabytes   int64                    `json:"gigabytes,omitempty"`
	Hours       int64                    `json:"hours,omitempty"`
}

func (ss SentinelSubscription) DTO() SentinelSubscriptionDTO {
	deposit, _ := strconv.ParseInt(ss.Deposit.Amount, 10, 64)

	return SentinelSubscriptionDTO{
		Base:        ss.Base.DTO(),
		Plan:        ss.PlanId,
		NodeAddress: ss.NodeAddress,
		Deposit:     deposit,
		Gigabytes:   ss.Gigabytes,
		Hours:       ss.Hours,
	}
}

type SentinelAllocation struct {
	Address       string `json:"address"`
	GrantedBytes  string `json:"granted_bytes"`
	UtilisedBytes string `json:"utilised_bytes"`
}

func (sa SentinelAllocation) DTO() SentinelAllocationDTO {
	grantedBytes, _ := strconv.ParseInt(sa.GrantedBytes, 10, 64)
	utilisedBytes, _ := strconv.ParseInt(sa.UtilisedBytes, 10, 64)

	return SentinelAllocationDTO{
		Address:       sa.Address,
		GrantedBytes:  grantedBytes,
		UtilisedBytes: utilisedBytes,
	}
}

type SentinelCredentials struct {
	Uid        string `json:"uid,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
	Result     string `json:"result"`
}

func (sc SentinelCredentials) DTO() SentinelCredentialsDTO {
	return SentinelCredentialsDTO{
		Uid:        sc.Uid,
		PrivateKey: sc.PrivateKey,
		Payload:    sc.Result,
	}
}
