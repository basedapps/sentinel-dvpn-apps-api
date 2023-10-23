package sentinel

import "time"

type SentinelSessionBandwidthDTO struct {
	Download int64 `json:"download"`
	Upload   int64 `json:"upload"`
}

type SentinelNodeBandwidthDTO struct {
	Download int64 `json:"download"`
	Upload   int64 `json:"upload"`
}

type SentinelSessionDTO struct {
	ID             int64                       `json:"id"`
	SubscriptionID int64                       `json:"subscriptionId"`
	NodeAddress    string                      `json:"nodeAddress"`
	Address        string                      `json:"address"`
	Duration       int64                       `json:"duration"`
	Bandwidth      SentinelSessionBandwidthDTO `json:"bandwidth"`
	Status         int64                       `json:"status"`
}

type SentinelSubscriptionBaseDTO struct {
	ID         int64     `json:"id"`
	Address    string    `json:"address"`
	Status     int64     `json:"status"`
	InactiveAt time.Time `json:"inactiveAt"`
}

type SentinelSubscriptionDTO struct {
	Base        SentinelSubscriptionBaseDTO `json:"base"`
	Plan        int64                       `json:"plan,omitempty"`
	NodeAddress string                      `json:"nodeAddress,omitempty"`
	Deposit     int64                       `json:"deposit,omitempty"`
	Gigabytes   int64                       `json:"gigabytes,omitempty"`
	Hours       int64                       `json:"hours,omitempty"`
}

type SentinelAllocationDTO struct {
	Address       string `json:"address"`
	GrantedBytes  int64  `json:"grantedBytes"`
	UtilisedBytes int64  `json:"utilisedBytes"`
}

type SentinelCredentialsDTO struct {
	Uid        string `json:"uid,omitempty"`
	PrivateKey string `json:"privateKey,omitempty"`
	Payload    string `json:"payload"`
}

type SentinelAllowanceDTO struct {
	Grantee string `json:"grantee"`
	Granter string `json:"granter"`
}
